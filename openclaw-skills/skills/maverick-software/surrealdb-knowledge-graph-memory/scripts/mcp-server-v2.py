#!/usr/bin/env python3
"""
MCP Server v2 for SurrealDB Knowledge Graph Memory

Enhanced with:
- Working memory integration
- Episode storage and retrieval
- Synchronous fact writes for important discoveries
- Context-aware retrieval

Run with: python3 mcp-server-v2.py
"""

import json
import sys
import os
from pathlib import Path
from datetime import datetime

# Add the scripts directory to path for imports
SCRIPT_DIR = Path(__file__).parent
sys.path.insert(0, str(SCRIPT_DIR))

# Configuration
SURREAL_CONFIG = {
    "connection": os.environ.get("SURREAL_URL", "http://localhost:8000"),
    "namespace": "openclaw",
    "database": "memory",
    "user": os.environ.get("SURREAL_USER", "root"),
    "password": os.environ.get("SURREAL_PASS", "root"),
}

SYNC_WRITE_THRESHOLD = 0.7

try:
    from surrealdb import Surreal
    import openai
    DEPS_AVAILABLE = True
except ImportError:
    DEPS_AVAILABLE = False

# Import our new modules
try:
    from working_memory import WorkingMemory
    from episodes import EpisodicMemory, SyncFactWriter
    MODULES_AVAILABLE = True
except ImportError:
    MODULES_AVAILABLE = False


# ============================================
# MCP Protocol helpers
# ============================================

def send_response(id, result=None, error=None):
    """Send JSON-RPC response."""
    response = {"jsonrpc": "2.0", "id": id}
    if error:
        response["error"] = error
    else:
        response["result"] = result
    print(json.dumps(response), flush=True)


def send_notification(method, params=None):
    """Send JSON-RPC notification."""
    notification = {"jsonrpc": "2.0", "method": method}
    if params:
        notification["params"] = params
    print(json.dumps(notification), flush=True)


# ============================================
# Database helpers
# ============================================

def get_db():
    """Get database connection."""
    if not DEPS_AVAILABLE:
        return None
    db = Surreal(SURREAL_CONFIG["connection"])
    db.signin({"username": SURREAL_CONFIG["user"], "password": SURREAL_CONFIG["password"]})
    db.use(SURREAL_CONFIG["namespace"], SURREAL_CONFIG["database"])
    return db


def close_db(db):
    """Safely close database connection."""
    if db:
        try:
            db.close()
        except (NotImplementedError, AttributeError):
            pass


def get_embedding(text: str) -> list:
    """Generate embedding for text."""
    if not DEPS_AVAILABLE:
        return []
    api_key = os.environ.get("OPENAI_API_KEY")
    if not api_key:
        return []
    client = openai.OpenAI(api_key=api_key)
    response = client.embeddings.create(input=text, model="text-embedding-3-small")
    return response.data[0].embedding


# ============================================
# Original Tool implementations (v1)
# ============================================

def knowledge_search(query: str, limit: int = 10, min_confidence: float = 0.3) -> dict:
    """Search for facts in the knowledge graph."""
    if not DEPS_AVAILABLE:
        return {"error": "Dependencies not available (surrealdb, openai)"}
    
    try:
        db = get_db()
        embedding = get_embedding(query)
        
        if not embedding:
            results = db.query("""
                SELECT id, content, confidence, source, tags
                FROM fact
                WHERE archived = false AND content CONTAINS $query
                ORDER BY confidence DESC
                LIMIT $limit
            """, {"query": query, "limit": limit})
        else:
            results = db.query("""
                SELECT id, content, confidence, source, tags,
                    vector::similarity::cosine(embedding, $embedding) AS similarity
                FROM fact
                WHERE archived = false AND confidence >= $min_conf
                ORDER BY similarity DESC
                LIMIT $limit
            """, {"embedding": embedding, "limit": limit, "min_conf": min_confidence})
        
        close_db(db)
        
        facts = results if isinstance(results, list) else []
        return {
            "query": query,
            "count": len(facts),
            "facts": [
                {
                    "id": str(f.get("id", "")),
                    "content": f.get("content", ""),
                    "confidence": f.get("confidence", 0),
                    "similarity": f.get("similarity", None),
                    "source": f.get("source", ""),
                }
                for f in facts
            ]
        }
    except Exception as e:
        return {"error": str(e)}


def knowledge_recall(fact_id: str = None, query: str = None) -> dict:
    """Recall a specific fact with its full context."""
    if not DEPS_AVAILABLE:
        return {"error": "Dependencies not available"}
    
    try:
        db = get_db()
        
        if query and not fact_id:
            search_result = knowledge_search(query, limit=1)
            if search_result.get("facts"):
                fact_id = search_result["facts"][0]["id"]
            else:
                return {"error": "No matching fact found", "query": query}
        
        if not fact_id:
            return {"error": "Either fact_id or query required"}
        
        fact_result = db.select(fact_id)
        if not fact_result:
            return {"error": f"Fact not found: {fact_id}"}
        
        fact = fact_result if isinstance(fact_result, dict) else fact_result[0] if fact_result else {}
        
        supporting = db.query("""
            SELECT in.id AS id, in.content AS content, in.confidence AS confidence, strength
            FROM relates_to WHERE out = $fact_id AND relationship = "supports"
        """, {"fact_id": fact_id})
        
        contradicting = db.query("""
            SELECT in.id AS id, in.content AS content, in.confidence AS confidence, strength
            FROM relates_to WHERE out = $fact_id AND relationship = "contradicts"
        """, {"fact_id": fact_id})
        
        entities = db.query("""
            SELECT out.id AS id, out.name AS name, out.type AS type, role
            FROM mentions WHERE in = $fact_id
        """, {"fact_id": fact_id})
        
        close_db(db)
        
        return {
            "fact": {
                "id": str(fact.get("id", "")),
                "content": fact.get("content", ""),
                "confidence": fact.get("confidence", 0),
                "source": fact.get("source", ""),
                "tags": fact.get("tags", []),
                "created_at": str(fact.get("created_at", "")),
                "success_count": fact.get("success_count", 0),
                "failure_count": fact.get("failure_count", 0),
            },
            "supporting_facts": supporting if isinstance(supporting, list) else [],
            "contradicting_facts": contradicting if isinstance(contradicting, list) else [],
            "entities": entities if isinstance(entities, list) else [],
        }
    except Exception as e:
        return {"error": str(e)}


def knowledge_store(content: str, source: str = "explicit", confidence: float = 0.9, tags: list = None) -> dict:
    """Store a new fact in the knowledge graph."""
    if not DEPS_AVAILABLE:
        return {"error": "Dependencies not available"}
    
    try:
        db = get_db()
        embedding = get_embedding(content)
        
        result = db.create("fact", {
            "content": content,
            "embedding": embedding,
            "source": source,
            "confidence": confidence,
            "tags": tags or [],
            "success_count": 0,
            "failure_count": 0,
        })
        
        fact_id = result[0]["id"] if isinstance(result, list) else result.get("id")
        close_db(db)
        
        return {
            "success": True,
            "fact_id": str(fact_id),
            "content": content,
            "confidence": confidence,
        }
    except Exception as e:
        return {"error": str(e)}


def knowledge_stats() -> dict:
    """Get knowledge graph statistics."""
    if not DEPS_AVAILABLE:
        return {"error": "Dependencies not available"}
    
    try:
        db = get_db()
        
        facts = db.query("SELECT count() FROM fact WHERE archived = false GROUP ALL")
        entities = db.query("SELECT count() FROM entity GROUP ALL")
        relations = db.query("SELECT count() FROM relates_to GROUP ALL")
        episodes = db.query("SELECT count() FROM episode GROUP ALL")
        avg_conf = db.query("SELECT math::mean(confidence) AS avg FROM fact WHERE archived = false GROUP ALL")
        
        close_db(db)
        
        return {
            "facts": facts[0].get("count", 0) if facts else 0,
            "entities": entities[0].get("count", 0) if entities else 0,
            "relations": relations[0].get("count", 0) if relations else 0,
            "episodes": episodes[0].get("count", 0) if episodes else 0,
            "avg_confidence": avg_conf[0].get("avg", 0) if avg_conf else 0,
        }
    except Exception as e:
        return {"error": str(e)}


# ============================================
# NEW v2 Tool implementations
# ============================================

def knowledge_store_sync(
    content: str,
    importance: float = 0.5,
    source: str = "inferred",
    confidence: float = None,
    tags: list = None,
    context: str = None
) -> dict:
    """
    Store a fact with importance-based routing.
    High importance (>0.7) → immediate write
    Low importance → returns queue_for_batch=True
    """
    if importance >= SYNC_WRITE_THRESHOLD:
        # Immediate sync write
        if confidence is None:
            confidence = 0.85 if source == "explicit" else 0.7
        
        result = knowledge_store(content, source, confidence, tags)
        result["sync_written"] = True
        result["importance"] = importance
        return result
    else:
        # Queue for batch (caller should log to daily file)
        return {
            "queued_for_batch": True,
            "content": content,
            "importance": importance,
            "message": "Importance below threshold; log to daily file for batch extraction"
        }


def episode_search(query: str, limit: int = 5, outcome: str = None) -> dict:
    """Search for similar past episodes/tasks."""
    if not MODULES_AVAILABLE:
        return {"error": "Episode module not available"}
    
    em = EpisodicMemory()
    episodes = em.find_similar_episodes(query, limit, outcome)
    
    return {
        "query": query,
        "count": len(episodes),
        "episodes": episodes
    }


def episode_learnings(task_goal: str, limit: int = 10) -> dict:
    """Get relevant learnings from past similar tasks."""
    if not MODULES_AVAILABLE:
        return {"error": "Episode module not available"}
    
    em = EpisodicMemory()
    learnings = em.get_learnings_for_task(task_goal, limit)
    
    return {
        "task_goal": task_goal,
        "count": len(learnings),
        "learnings": learnings
    }


def episode_store(episode_data: dict) -> dict:
    """Store a completed episode."""
    if not MODULES_AVAILABLE:
        return {"error": "Episode module not available"}
    
    em = EpisodicMemory()
    episode_id = em.store_episode(episode_data)
    
    if episode_id:
        return {
            "success": True,
            "episode_id": episode_id,
            "outcome": episode_data.get("outcome", "unknown")
        }
    else:
        return {"error": "Failed to store episode"}


def working_memory_status() -> dict:
    """Get current working memory state."""
    if not MODULES_AVAILABLE:
        return {"error": "Working memory module not available"}
    
    wm = WorkingMemory()
    return wm.get_progress()


def context_aware_search(
    query: str,
    task_context: str = None,
    limit: int = 10,
    include_episodes: bool = True
) -> dict:
    """
    Search with awareness of current task context.
    Boosts facts relevant to both query AND current task.
    """
    if not DEPS_AVAILABLE:
        return {"error": "Dependencies not available"}
    
    try:
        db = get_db()
        query_embedding = get_embedding(query)
        task_embedding = get_embedding(task_context) if task_context else None
        
        if not query_embedding:
            return {"error": "Could not generate embedding"}
        
        # Base semantic search
        base_results = db.query("""
            SELECT id, content, confidence, source, tags,
                vector::similarity::cosine(embedding, $embedding) AS similarity
            FROM fact
            WHERE archived = false
            ORDER BY similarity DESC
            LIMIT $limit
        """, {"embedding": query_embedding, "limit": limit * 2})
        
        facts = base_results if isinstance(base_results, list) else []
        
        # If we have task context, boost relevant facts
        if task_embedding:
            for fact in facts:
                fact_embedding = fact.get("embedding", [])
                if fact_embedding:
                    # Calculate task relevance
                    task_sim = sum(a*b for a,b in zip(fact_embedding[:100], task_embedding[:100])) / 100
                    if task_sim > 0.3:
                        fact["similarity"] = fact.get("similarity", 0) * 1.5
                        fact["task_relevant"] = True
        
        # Sort by boosted similarity and limit
        facts = sorted(facts, key=lambda f: f.get("similarity", 0), reverse=True)[:limit]
        
        close_db(db)
        
        result = {
            "query": query,
            "task_context": task_context,
            "count": len(facts),
            "facts": [
                {
                    "id": str(f.get("id", "")),
                    "content": f.get("content", ""),
                    "confidence": f.get("confidence", 0),
                    "similarity": f.get("similarity", 0),
                    "task_relevant": f.get("task_relevant", False),
                }
                for f in facts
            ]
        }
        
        # Optionally include relevant episodes
        if include_episodes and MODULES_AVAILABLE and task_context:
            em = EpisodicMemory()
            episodes = em.find_similar_episodes(task_context, limit=3)
            result["related_episodes"] = episodes
        
        return result
        
    except Exception as e:
        return {"error": str(e)}


# ============================================
# MCP Tool definitions
# ============================================

TOOLS = [
    # Original v1 tools
    {
        "name": "knowledge_search",
        "description": "Search the knowledge graph for facts matching a query. Returns semantically similar facts ranked by relevance and confidence.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Search query"},
                "limit": {"type": "integer", "description": "Max results (default: 10)", "default": 10},
                "min_confidence": {"type": "number", "description": "Min confidence 0-1 (default: 0.3)", "default": 0.3}
            },
            "required": ["query"]
        }
    },
    {
        "name": "knowledge_recall",
        "description": "Recall a specific fact with full context (supporting/contradicting facts, entities).",
        "inputSchema": {
            "type": "object",
            "properties": {
                "fact_id": {"type": "string", "description": "Fact ID (e.g., 'fact:abc123')"},
                "query": {"type": "string", "description": "Alternative: search query to find the fact"}
            }
        }
    },
    {
        "name": "knowledge_store",
        "description": "Store a new fact in the knowledge graph.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "content": {"type": "string", "description": "The fact content"},
                "source": {"type": "string", "description": "Source type: 'explicit' or 'inferred'", "default": "explicit"},
                "confidence": {"type": "number", "description": "Confidence 0-1 (default: 0.9)", "default": 0.9},
                "tags": {"type": "array", "items": {"type": "string"}, "description": "Optional tags"}
            },
            "required": ["content"]
        }
    },
    {
        "name": "knowledge_stats",
        "description": "Get knowledge graph statistics (facts, entities, relations, episodes, confidence).",
        "inputSchema": {"type": "object", "properties": {}}
    },
    
    # NEW v2 tools
    {
        "name": "knowledge_store_sync",
        "description": "Store a fact with importance-based routing. High importance (>0.7) writes immediately; low importance queues for batch extraction.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "content": {"type": "string", "description": "The fact content"},
                "importance": {"type": "number", "description": "How important (0-1). >0.7 = sync write", "default": 0.5},
                "source": {"type": "string", "description": "Source type", "default": "inferred"},
                "confidence": {"type": "number", "description": "Confidence (auto-set if not provided)"},
                "tags": {"type": "array", "items": {"type": "string"}},
                "context": {"type": "string", "description": "Current task context"}
            },
            "required": ["content"]
        }
    },
    {
        "name": "episode_search",
        "description": "Search for similar past episodes/tasks. Learn from what worked or failed before.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Search query (usually current task goal)"},
                "limit": {"type": "integer", "default": 5},
                "outcome": {"type": "string", "description": "Filter: 'success', 'failure', or omit for all"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "episode_learnings",
        "description": "Get relevant learnings from past similar tasks. Returns actionable insights.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "task_goal": {"type": "string", "description": "Current task goal"},
                "limit": {"type": "integer", "default": 10}
            },
            "required": ["task_goal"]
        }
    },
    {
        "name": "episode_store",
        "description": "Store a completed episode (task history). Usually called by working memory on task completion.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "episode_data": {
                    "type": "object",
                    "description": "Episode data from WorkingMemory.complete_task()",
                    "properties": {
                        "task": {"type": "string"},
                        "goal": {"type": "string"},
                        "outcome": {"type": "string"},
                        "decisions": {"type": "array"},
                        "problems": {"type": "array"},
                        "solutions": {"type": "array"},
                        "key_learnings": {"type": "array"}
                    }
                }
            },
            "required": ["episode_data"]
        }
    },
    {
        "name": "working_memory_status",
        "description": "Get current working memory state (active task, progress, confidence).",
        "inputSchema": {"type": "object", "properties": {}}
    },
    {
        "name": "context_aware_search",
        "description": "Search with awareness of current task. Boosts facts relevant to both query AND task context.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "What you're looking for"},
                "task_context": {"type": "string", "description": "Current task goal/context"},
                "limit": {"type": "integer", "default": 10},
                "include_episodes": {"type": "boolean", "default": True, "description": "Include related episodes"}
            },
            "required": ["query"]
        }
    }
]


# ============================================
# Request handlers
# ============================================

def handle_initialize(id, params):
    """Handle initialize request."""
    send_response(id, {
        "protocolVersion": "2024-11-05",
        "capabilities": {"tools": {}},
        "serverInfo": {"name": "surrealdb-memory", "version": "2.0.0"}
    })


def handle_tools_list(id, params):
    """Handle tools/list request."""
    send_response(id, {"tools": TOOLS})


def handle_tools_call(id, params):
    """Handle tools/call request."""
    tool_name = params.get("name")
    arguments = params.get("arguments", {})
    
    # Route to appropriate handler
    handlers = {
        "knowledge_search": lambda: knowledge_search(
            arguments.get("query", ""),
            arguments.get("limit", 10),
            arguments.get("min_confidence", 0.3)
        ),
        "knowledge_recall": lambda: knowledge_recall(
            arguments.get("fact_id"),
            arguments.get("query")
        ),
        "knowledge_store": lambda: knowledge_store(
            arguments.get("content", ""),
            arguments.get("source", "explicit"),
            arguments.get("confidence", 0.9),
            arguments.get("tags")
        ),
        "knowledge_stats": knowledge_stats,
        "knowledge_store_sync": lambda: knowledge_store_sync(
            arguments.get("content", ""),
            arguments.get("importance", 0.5),
            arguments.get("source", "inferred"),
            arguments.get("confidence"),
            arguments.get("tags"),
            arguments.get("context")
        ),
        "episode_search": lambda: episode_search(
            arguments.get("query", ""),
            arguments.get("limit", 5),
            arguments.get("outcome")
        ),
        "episode_learnings": lambda: episode_learnings(
            arguments.get("task_goal", ""),
            arguments.get("limit", 10)
        ),
        "episode_store": lambda: episode_store(arguments.get("episode_data", {})),
        "working_memory_status": working_memory_status,
        "context_aware_search": lambda: context_aware_search(
            arguments.get("query", ""),
            arguments.get("task_context"),
            arguments.get("limit", 10),
            arguments.get("include_episodes", True)
        )
    }
    
    if tool_name in handlers:
        result = handlers[tool_name]()
        send_response(id, {
            "content": [{"type": "text", "text": json.dumps(result, indent=2, default=str)}]
        })
    else:
        send_response(id, error={"code": -32601, "message": f"Unknown tool: {tool_name}"})


def main():
    """Main MCP server loop."""
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        
        try:
            request = json.loads(line)
        except json.JSONDecodeError:
            continue
        
        method = request.get("method")
        id = request.get("id")
        params = request.get("params", {})
        
        if method == "initialize":
            handle_initialize(id, params)
        elif method == "tools/list":
            handle_tools_list(id, params)
        elif method == "tools/call":
            handle_tools_call(id, params)
        elif method == "notifications/initialized":
            pass
        else:
            if id is not None:
                send_response(id, error={"code": -32601, "message": f"Unknown method: {method}"})


if __name__ == "__main__":
    main()
