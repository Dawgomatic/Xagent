#!/usr/bin/env python3
"""
NIMA Lazy Reconstruction Recall v4 - LadybugDB Backend
=======================================================

Uses LadybugDB with HNSW vector index for memory retrieval.

Author: NIMA Core Team
Date: 2026-02-14
"""

import sys
import os
import time
import json
from datetime import datetime, timedelta
from typing import Optional, List, Dict, Any

# Add venv path
sys.path.insert(0, os.path.expanduser("~/.openclaw/workspace/.venv/lib/python3.11/site-packages"))

import real_ladybug as lb

# Config
LADYBUG_DB = os.path.expanduser("~/.nima/memory/ladybug.lbug")
MAX_RESULTS = 7
TIME_WINDOW_DAYS = 90
EMBEDDING_THRESHOLD = 0.35

# =============================================================================
# CONNECTION MANAGEMENT
# =============================================================================

_db = None
_conn = None

def connect():
    """Get or create database connection."""
    global _db, _conn
    
    if _conn is None:
        _db = lb.Database(LADYBUG_DB)
        _conn = lb.Connection(_db)
        # Load vector extension
        try:
            _conn.execute("LOAD VECTOR")
        except:
            pass
    
    return _conn

def disconnect():
    """Close database connection."""
    global _db, _conn
    _db = None
    _conn = None

# =============================================================================
# SEARCH FUNCTIONS
# =============================================================================

def text_search(query: str, limit: int = MAX_RESULTS) -> List[Dict]:
    """Search memories by text content (CONTAINS).
    
    SECURITY: Uses parameterized queries to prevent Cypher injection.
    Fixed 2026-02-16 - CVE pending assignment.
    """
    conn = connect()
    
    start = time.time()
    # SECURITY FIX: No string escaping - use parameterized queries instead
    
    min_timestamp = int((datetime.now() - timedelta(days=TIME_WINDOW_DAYS)).timestamp() * 1000)
    
    try:
        # SECURITY FIX: Parameterized query prevents injection
        result = conn.execute("""
            MATCH (n:MemoryNode)
            WHERE (n.text CONTAINS $query OR n.summary CONTAINS $query)
            AND n.timestamp >= $min_ts
            RETURN n.id, n.text, n.summary, n.who, n.layer, n.timestamp
            LIMIT $result_limit
        """, {"query": query, "min_ts": min_timestamp, "result_limit": limit * 2})
        
        memories = []
        for row in result:
            memories.append({
                'id': row[0],
                'text': row[1] or "",
                'summary': row[2] or "",
                'who': row[3] or "unknown",
                'layer': row[4] or "unknown",
                'timestamp': row[5] or 0,
                'fts_score': 1.0
            })
        
        elapsed = (time.time() - start) * 1000
        return memories, elapsed
    except Exception as e:
        print(f"[ladybug_recall] Text search error: {e}", file=sys.stderr)
        return [], 0

def vector_search(query_embedding: List[float], limit: int = MAX_RESULTS) -> List[Dict]:
    """Search memories by vector similarity (HNSW).
    
    SECURITY: Uses parameterized queries to prevent injection.
    Fixed 2026-02-16 - CVE pending assignment.
    """
    conn = connect()
    
    start = time.time()
    
    # SECURITY FIX: Validate embedding is a list of floats only
    if not isinstance(query_embedding, list):
        print(f"[ladybug_recall] Invalid embedding type: {type(query_embedding)}", file=sys.stderr)
        return [], 0
    
    # Sanitize: ensure all values are valid floats (no injection via array elements)
    try:
        sanitized_embedding = [float(x) for x in query_embedding]
    except (ValueError, TypeError) as e:
        print(f"[ladybug_recall] Invalid embedding values: {e}", file=sys.stderr)
        return [], 0
    
    min_timestamp = int((datetime.now() - timedelta(days=TIME_WINDOW_DAYS)).timestamp() * 1000)
    
    try:
        # SECURITY FIX: Parameterized query for vector search
        # Note: LadybugDB QUERY_VECTOR_INDEX uses positional params for vectors
        result = conn.execute("""
            CALL QUERY_VECTOR_INDEX(
                'MemoryNode',
                'embedding_idx',
                $embedding,
                $top_k
            )
            RETURN node.id, node.text, node.summary, node.who, node.layer, node.timestamp, distance
            ORDER BY distance
        """, {"embedding": sanitized_embedding, "top_k": limit * 3})
        
        memories = []
        for row in result:
            # Convert distance to similarity score (1 - distance for cosine)
            distance = row[6] if row[6] is not None else 1.0
            score = 1.0 - distance
            
            if score < EMBEDDING_THRESHOLD:
                continue
            
            timestamp = row[5] or 0
            if timestamp < min_timestamp:
                continue
            
            memories.append({
                'id': row[0],
                'text': row[1] or "",
                'summary': row[2] or "",
                'who': row[3] or "unknown",
                'layer': row[4] or "unknown",
                'timestamp': timestamp,
                'emb_score': score
            })
        
        elapsed = (time.time() - start) * 1000
        return memories[:limit], elapsed
    except Exception as e:
        print(f"[ladybug_recall] Vector search error: {e}", file=sys.stderr)
        return [], 0

def who_search(who: str, limit: int = MAX_RESULTS) -> List[Dict]:
    """Search memories by who said it.
    
    SECURITY: Uses parameterized queries to prevent Cypher injection.
    Fixed 2026-02-16 - CVE pending assignment.
    """
    conn = connect()
    
    start = time.time()
    # SECURITY FIX: No string escaping - use parameterized queries instead
    
    min_timestamp = int((datetime.now() - timedelta(days=TIME_WINDOW_DAYS)).timestamp() * 1000)
    
    try:
        # SECURITY FIX: Parameterized query prevents injection
        result = conn.execute("""
            MATCH (n:MemoryNode {who: $who_param})
            WHERE n.timestamp >= $min_ts
            RETURN n.id, n.text, n.summary, n.who, n.layer, n.timestamp
            ORDER BY n.timestamp DESC
            LIMIT $result_limit
        """, {"who_param": who, "min_ts": min_timestamp, "result_limit": limit})
        
        memories = []
        for row in result:
            memories.append({
                'id': row[0],
                'text': row[1] or "",
                'summary': row[2] or "",
                'who': row[3] or "unknown",
                'layer': row[4] or "unknown",
                'timestamp': row[5] or 0,
                'who_score': 1.0
            })
        
        elapsed = (time.time() - start) * 1000
        return memories, elapsed
    except Exception as e:
        print(f"[ladybug_recall] Who search error: {e}", file=sys.stderr)
        return [], 0

def hybrid_search(query: str, query_embedding: List[float], 
                  limit: int = MAX_RESULTS,
                  text_weight: float = 0.3,
                  who_boost: float = 1.2) -> List[Dict]:
    """
    Hybrid search combining text, vector, and who-based signals.
    
    Strategy:
    1. Get vector search results (semantic similarity)
    2. Get text search results (keyword matches)
    3. Combine and deduplicate, scoring each
    4. Boost score for "self" or known users
    5. Return top N by combined score
    """
    # Get results from both methods
    text_results, text_time = text_search(query, limit * 2)
    vector_results, vector_time = vector_search(query_embedding, limit * 2)
    
    # Combine and deduplicate
    seen_ids = {}
    
    # Add vector results (higher priority for semantic matches)
    for m in vector_results:
        score = m.get('emb_score', 0.5) * (1 - text_weight)
        if m.get('who') == 'self':
            score *= who_boost
        m['combined_score'] = score
        seen_ids[m['id']] = m
    
    # Add/boost text results
    for m in text_results:
        if m['id'] in seen_ids:
            # Boost existing score
            seen_ids[m['id']]['combined_score'] += text_weight
        else:
            # Add new result
            score = text_weight
            if m.get('who') == 'self':
                score *= who_boost
            m['combined_score'] = score
            seen_ids[m['id']] = m
    
    # Sort by combined score
    combined = sorted(seen_ids.values(), key=lambda x: x['combined_score'], reverse=True)
    
    # Format output
    results = []
    for m in combined[:limit]:
        results.append({
            'turn_id': f"ladybug:{m['id']}",
            'timestamp': m['timestamp'],
            'who': m['who'],
            'layer': m['layer'],
            'summary': m['summary'][:200] if m['summary'] else m['text'][:200],
            'text': m['text'][:500],
            'score': m['combined_score'],
            'time_ms': text_time + vector_time
        })
    
    return results

def get_related(node_id: int, limit: int = 5) -> List[Dict]:
    """Get related memories through graph edges.
    
    SECURITY: Uses parameterized queries to prevent Cypher injection.
    Fixed 2026-02-16 - CVE pending assignment.
    """
    conn = connect()
    
    start = time.time()
    
    # SECURITY FIX: Validate node_id is an integer
    try:
        validated_node_id = int(node_id)
    except (ValueError, TypeError):
        print(f"[ladybug_recall] Invalid node_id: {node_id}", file=sys.stderr)
        return [], 0
    
    try:
        # SECURITY FIX: Parameterized query prevents injection
        result = conn.execute("""
            MATCH (n:MemoryNode {id: $node_id})-[r:relates_to]-(related:MemoryNode)
            RETURN related.id, related.text, related.summary, related.who, related.layer, 
                   related.timestamp, r.weight, r.relation
            ORDER BY r.weight DESC
            LIMIT $result_limit
        """, {"node_id": validated_node_id, "result_limit": limit})
        
        memories = []
        for row in result:
            memories.append({
                'id': row[0],
                'text': row[1] or "",
                'summary': row[2] or "",
                'who': row[3] or "unknown",
                'layer': row[4] or "unknown",
                'timestamp': row[5] or 0,
                'weight': row[6] if row[6] else 1.0,
                'relation': row[7] or "related"
            })
        
        elapsed = (time.time() - start) * 1000
        return memories, elapsed
    except Exception as e:
        print(f"[ladybug_recall] Get related error: {e}", file=sys.stderr)
        return [], 0

# =============================================================================
# COMPATIBILITY LAYER (drop-in replacement for lazy_recall_v3)
# =============================================================================

def lazy_recall(query: str, 
                 query_embedding: Optional[List[float]] = None,
                 who: Optional[str] = None,
                 top_k: int = MAX_RESULTS,
                 verbose: bool = False) -> List[Dict]:
    """
    Main recall function - compatible with lazy_recall_v3 interface.
    
    Args:
        query: Text query
        query_embedding: Optional embedding vector for semantic search
        who: Optional who filter
        top_k: Maximum results to return
        verbose: Print timing info
    
    Returns:
        List of memory dicts with turn_id, timestamp, who, summary, score
    """
    start = time.time()
    
    if verbose:
        print(f"[ladybug_recall] Query: '{query[:50]}...'", file=sys.stderr)
    
    # If we have an embedding, do hybrid search
    if query_embedding:
        results = hybrid_search(query, query_embedding, limit=top_k)
    else:
        # Text-only search
        text_results, text_time = text_search(query, limit=top_k)
        results = [{
            'turn_id': f"ladybug:{m['id']}",
            'timestamp': m['timestamp'],
            'who': m['who'],
            'layer': m['layer'],
            'summary': m['summary'][:200] if m['summary'] else m['text'][:200],
            'text': m['text'][:500],
            'score': m.get('fts_score', 1.0),
            'time_ms': text_time
        } for m in text_results]
    
    if verbose:
        total_time = (time.time() - start) * 1000
        print(f"[ladybug_recall] Found {len(results)} results in {total_time:.1f}ms", file=sys.stderr)
    
    return results

# =============================================================================
# CLI
# =============================================================================

def main():
    import argparse
    
    parser = argparse.ArgumentParser(description="NIMA LadybugDB Recall")
    parser.add_argument("query", nargs="?", default="memory", help="Search query")
    parser.add_argument("--top", type=int, default=5, help="Top results")
    parser.add_argument("--who", type=str, help="Filter by who")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    
    args = parser.parse_args()
    
    results = lazy_recall(
        args.query,
        who=args.who,
        top_k=args.top,
        verbose=args.verbose
    )
    
    if args.json:
        print(json.dumps(results, indent=2))
    else:
        print(f"\n🔍 Query: '{args.query}'")
        print(f"   Found {len(results)} results\n")
        for i, r in enumerate(results):
            print(f"{i+1}. [{r.get('who', '?')}] {r.get('summary', '')[:100]}...")
            print(f"   Score: {r.get('score', 0):.3f} | Time: {r.get('time_ms', 0):.1f}ms\n")

if __name__ == "__main__":
    main()