#!/usr/bin/env python3
"""
LadybugDB storage for NIMA Memory.
Called from index.js via execFileSync.

Author: NIMA Core Team
Date: 2026-02-15
"""
import sys
import json
import os
import time

# Add venv path for real_ladybug
VENV_PATHS = [
    os.path.expanduser("~/.openclaw/workspace/.venv/lib/python3.11/site-packages"),
    os.path.expanduser("~/.openclaw/workspace/.venv/lib/python3.13/site-packages"),
    os.path.expanduser("~/.openclaw/workspace/.venv/lib/python3.14/site-packages"),
]
for vp in VENV_PATHS:
    if os.path.exists(vp):
        sys.path.insert(0, vp)
        break

try:
    import real_ladybug as lb
    LADYBUG_AVAILABLE = True
except ImportError:
    LADYBUG_AVAILABLE = False

LADYBUG_DB = os.path.expanduser("~/.nima/memory/ladybug.lbug")
MAX_TEXT_LENGTH = 2000
MAX_SUMMARY_LENGTH = 500


def get_next_id(conn, retry_offset: int = 0) -> int:
    """
    Get next available node ID with retry offset for concurrency.
    
    Args:
        conn: Database connection
        retry_offset: Offset to add for retry attempts (0 for first try)
        
    Returns:
        Next available ID
    """
    result = conn.execute("MATCH (n:MemoryNode) RETURN max(n.id) as max_id")
    for row in result:
        max_id = row[0]
        return (max_id or 0) + 1 + retry_offset
    return 1 + retry_offset


def get_next_turn_id(conn, retry_offset: int = 0) -> int:
    """
    Get next available Turn ID with retry offset for concurrency.
    
    Args:
        conn: Database connection
        retry_offset: Offset to add for retry attempts (0 for first try)
        
    Returns:
        Next available ID
    """
    result = conn.execute("MATCH (t:Turn) RETURN max(t.id) as max_id")
    for row in result:
        max_id = row[0]
        return (max_id or 0) + 1 + retry_offset
    return 1 + retry_offset


def truncate(text: str, max_len: int) -> str:
    """Truncate text to max length."""
    if not text:
        return ""
    return text[:max_len] if len(text) <= max_len else text[:max_len-3] + "..."


def _store_memory_attempt(data: dict, conn, retry_offset: int = 0) -> tuple[bool, list[int]]:
    """
    Single attempt to store memory (for retry logic).
    
    Args:
        data: Parsed JSON memory data
        conn: Database connection
        retry_offset: ID offset for retry attempts
        
    Returns:
        (success: bool, created_node_ids: list)
    """
    created_nodes = []
    
    try:
        # Load vector extension if available
        try:
            conn.execute("LOAD VECTOR")
        except:
            pass
        
        # Get next IDs with retry offset
        base_id = get_next_id(conn, retry_offset)
        input_id = base_id
        contemplation_id = base_id + 1
        output_id = base_id + 2
        turn_db_id = get_next_turn_id(conn, retry_offset)
        
        timestamp = data.get("timestamp", int(time.time() * 1000))
        turn_id = data.get("turn_id", f"turn_{timestamp}")
        affect_json = data.get("affect_json", "{}")
        session_key = data.get("session_key", "")
        conversation_id = data.get("conversation_id", "")
        fe_score = data.get("fe_score", 0.5)
        
        # AUDIT FIX: Track all operations for potential rollback
        # Note: LadybugDB doesn't have native transactions, so we track for cleanup
        
        # Insert input node
        conn.execute("""
            CREATE (n:MemoryNode {
                id: $id,
                timestamp: $timestamp,
                layer: 'input',
                text: $text,
                summary: $summary,
                who: $who,
                affect_json: $affect_json,
                session_key: $session_key,
                conversation_id: $conversation_id,
                turn_id: $turn_id,
                fe_score: $fe_score
            })
        """, {
            "id": input_id,
            "timestamp": timestamp,
            "text": truncate(data["input"]["text"], MAX_TEXT_LENGTH),
            "summary": truncate(data["input"]["summary"], MAX_SUMMARY_LENGTH),
            "who": data["input"].get("who", "unknown"),
            "affect_json": affect_json,
            "session_key": session_key,
            "conversation_id": conversation_id,
            "turn_id": turn_id,
            "fe_score": fe_score
        })
        created_nodes.append(input_id)
        
        # Insert contemplation node
        conn.execute("""
            CREATE (n:MemoryNode {
                id: $id,
                timestamp: $timestamp,
                layer: 'contemplation',
                text: $text,
                summary: $summary,
                who: 'self',
                affect_json: $affect_json,
                session_key: $session_key,
                conversation_id: $conversation_id,
                turn_id: $turn_id,
                fe_score: $fe_score
            })
        """, {
            "id": contemplation_id,
            "timestamp": timestamp,
            "text": truncate(data["contemplation"]["text"], MAX_TEXT_LENGTH),
            "summary": truncate(data["contemplation"]["summary"], MAX_SUMMARY_LENGTH),
            "affect_json": affect_json,
            "session_key": session_key,
            "conversation_id": conversation_id,
            "turn_id": turn_id,
            "fe_score": fe_score
        })
        created_nodes.append(contemplation_id)
        
        # Insert output node
        conn.execute("""
            CREATE (n:MemoryNode {
                id: $id,
                timestamp: $timestamp,
                layer: 'output',
                text: $text,
                summary: $summary,
                who: 'self',
                affect_json: $affect_json,
                session_key: $session_key,
                conversation_id: $conversation_id,
                turn_id: $turn_id,
                fe_score: $fe_score
            })
        """, {
            "id": output_id,
            "timestamp": timestamp,
            "text": truncate(data["output"]["text"], MAX_TEXT_LENGTH),
            "summary": truncate(data["output"]["summary"], MAX_SUMMARY_LENGTH),
            "affect_json": affect_json,
            "session_key": session_key,
            "conversation_id": conversation_id,
            "turn_id": turn_id,
            "fe_score": fe_score
        })
        created_nodes.append(output_id)
        
        # Create edges: input → contemplation → output
        conn.execute("""
            MATCH (a:MemoryNode {id: $source}), (b:MemoryNode {id: $target})
            CREATE (a)-[:relates_to {relation: 'triggered', weight: 1.0}]->(b)
        """, {"source": input_id, "target": contemplation_id})
        
        conn.execute("""
            MATCH (a:MemoryNode {id: $source}), (b:MemoryNode {id: $target})
            CREATE (a)-[:relates_to {relation: 'produced', weight: 1.0}]->(b)
        """, {"source": contemplation_id, "target": output_id})
        
        conn.execute("""
            MATCH (a:MemoryNode {id: $source}), (b:MemoryNode {id: $target})
            CREATE (a)-[:relates_to {relation: 'responded_to', weight: 1.0}]->(b)
        """, {"source": output_id, "target": input_id})
        
        # Create turn node and relationships in a single atomic query
        conn.execute("""
            CREATE (t:Turn {
                id: $id,
                turn_id: $turn_id,
                timestamp: $timestamp,
                affect_json: $affect_json
            })
            WITH t
            MATCH (input_node:MemoryNode {id: $input_id})
            MATCH (contemplation_node:MemoryNode {id: $contemplation_id})
            MATCH (output_node:MemoryNode {id: $output_id})
            CREATE (t)-[:has_input]->(input_node)
            CREATE (t)-[:has_contemplation]->(contemplation_node)
            CREATE (t)-[:has_output]->(output_node)
        """, {
            "id": turn_db_id,
            "turn_id": turn_id,
            "timestamp": timestamp,
            "affect_json": affect_json,
            "input_id": input_id,
            "contemplation_id": contemplation_id,
            "output_id": output_id
        })
        
        # Success
        return (True, created_nodes, f"{input_id},{contemplation_id},{output_id}")
        
    except Exception as e:
        # AUDIT FIX: Attempt cleanup of partially created nodes (Issue #2)
        # Check if this is a constraint violation (ID conflict)
        error_msg = str(e).lower()
        is_constraint_violation = any(keyword in error_msg for keyword in [
            'duplicate', 'constraint', 'unique', 'primary key', 'already exists'
        ])
        
        if not is_constraint_violation:
            print(f"error:{e}", file=sys.stderr)
            import traceback
            traceback.print_exc(file=sys.stderr)
        
        # Try to clean up any nodes that were created before the failure
        if created_nodes:
            for node_id in created_nodes:
                try:
                    conn.execute("MATCH (n:MemoryNode {id: $id}) DETACH DELETE n", {"id": node_id})
                except:
                    pass  # Cleanup errors are not critical
        
        return (False, created_nodes, str(e))


def store_memory(data_file: str, max_retries: int = 5) -> bool:
    """
    Store memory turn to LadybugDB with retry-on-conflict for concurrent writes.
    
    CONCURRENCY FIX 2026-02-16: Added retry logic for race conditions in ID generation
    
    Args:
        data_file: Path to JSON file with memory data
        max_retries: Maximum number of retry attempts (default: 5)
        
    Returns:
        True if stored successfully
        
    Raises:
        Exception: On non-recoverable failure
    """
    if not LADYBUG_AVAILABLE:
        print("error:real_ladybug not installed", file=sys.stderr)
        return False
    
    if not os.path.exists(LADYBUG_DB):
        print(f"error:database not found at {LADYBUG_DB}", file=sys.stderr)
        return False
    
    # Parse JSON with explicit error handling
    try:
        with open(data_file, 'r') as f:
            data = json.load(f)
    except json.JSONDecodeError as e:
        print(f"error:invalid JSON in data file: {e}", file=sys.stderr)
        return False
    except FileNotFoundError:
        print(f"error:data file not found: {data_file}", file=sys.stderr)
        return False
    
    # Retry loop for constraint violations
    for attempt in range(max_retries):
        db = lb.Database(LADYBUG_DB)
        conn = lb.Connection(db)
        
        try:
            success, created_nodes, result = _store_memory_attempt(data, conn, retry_offset=attempt * 3)
            
            if success:
                print(f"stored:{result}")
                return True
            
            # Check if we should retry
            error_msg = result.lower()
            is_constraint_violation = any(keyword in error_msg for keyword in [
                'duplicate', 'constraint', 'unique', 'primary key', 'already exists'
            ])
            
            if not is_constraint_violation:
                # Non-retryable error
                print(f"error:{result}", file=sys.stderr)
                return False
            
            # Constraint violation - retry with offset
            if attempt < max_retries - 1:
                time.sleep(0.01 * (2 ** attempt))  # Exponential backoff
                continue
            else:
                # Max retries exceeded
                print(f"error:max retries ({max_retries}) exceeded due to ID conflicts", file=sys.stderr)
                return False
                
        finally:
            try:
                conn.close()
            except:
                pass
    
    return False


def health_check() -> dict:
    """Check LadybugDB health."""
    if not LADYBUG_AVAILABLE:
        return {"ok": False, "error": "real_ladybug not installed"}
    
    if not os.path.exists(LADYBUG_DB):
        return {"ok": False, "error": f"database not found at {LADYBUG_DB}"}
    
    try:
        db = lb.Database(LADYBUG_DB)
        conn = lb.Connection(db)
        
        result = conn.execute("MATCH (n:MemoryNode) RETURN count(n) as count")
        node_count = 0
        for row in result:
            node_count = row[0]
        
        result = conn.execute("MATCH ()-[r:relates_to]->() RETURN count(r) as count")
        edge_count = 0
        for row in result:
            edge_count = row[0]
        
        conn.close()
        
        return {
            "ok": True,
            "stats": {
                "nodes": node_count,
                "edges": edge_count,
                "db_path": LADYBUG_DB,
                "db_size_mb": round(os.path.getsize(LADYBUG_DB) / (1024 * 1024), 2)
            }
        }
    except Exception as e:
        return {"ok": False, "error": str(e)}


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: ladybug_store.py <data_file.json> | health", file=sys.stderr)
        sys.exit(1)
    
    if sys.argv[1] == "health":
        import json
        print(json.dumps(health_check(), indent=2))
        sys.exit(0)
    
    success = store_memory(sys.argv[1])
    sys.exit(0 if success else 1)
