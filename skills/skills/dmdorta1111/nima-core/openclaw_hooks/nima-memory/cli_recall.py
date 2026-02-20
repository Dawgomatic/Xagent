#!/usr/bin/env python3
"""
NIMA Graph Memory Recall CLI
Usage: python3 cli_recall.py "query" [--full] [--top N] [--who NAME] [--layer TYPE]
"""

import sqlite3
import json
import sys
import os
from datetime import datetime

GRAPH_DB = os.path.expanduser("~/.nima/memory/graph.sqlite")


def recall(query, max_results=3, max_summaries=15, traverse_depth=2, who_filter="", layer_filter="", full_text=False):
    if not os.path.exists(GRAPH_DB):
        print("❌ No memory database found. Capture some memories first.")
        return

    db = sqlite3.connect(GRAPH_DB)
    db.row_factory = sqlite3.Row

    terms = query.lower().split()
    if not terms:
        print("❌ Empty query")
        return

    # Step 1: Text search for entry points
    conditions = []
    params = []
    for term in terms[:10]:
        conditions.append("(LOWER(text) LIKE ? OR LOWER(summary) LIKE ?)")
        params.extend([f"%{term}%", f"%{term}%"])

    where = " OR ".join(conditions)
    if who_filter:
        where = f"({where}) AND LOWER(who) LIKE ?"
        params.append(f"%{who_filter.lower()}%")
    if layer_filter:
        where = f"({where}) AND layer = ?"
        params.append(layer_filter)

    # Score by matching terms
    score_expr = " + ".join(
        [f"(CASE WHEN LOWER(text) LIKE ? OR LOWER(summary) LIKE ? THEN 1 ELSE 0 END)" for _ in terms[:10]]
    )
    score_params = [p for t in terms[:10] for p in (f"%{t}%", f"%{t}%")]

    entry_points = db.execute(
        f"SELECT id, layer, summary, who, timestamp, turn_id, affect_json, ({score_expr}) as score "
        f"FROM memory_nodes WHERE {where} ORDER BY score DESC, timestamp DESC LIMIT ?",
        params + score_params + [max_summaries]
    ).fetchall()

    if not entry_points:
        print(f"🔍 No memories found for: '{query}'")
        db.close()
        return

    # Step 2: Graph traversal
    visited = set()
    related = {}

    def traverse(node_id, depth, base_score):
        if depth > traverse_depth or node_id in visited:
            return
        visited.add(node_id)

        edges = db.execute("""
            SELECT target_id as cid, relation FROM memory_edges WHERE source_id = ?
            UNION
            SELECT source_id as cid, relation FROM memory_edges WHERE target_id = ?
        """, (node_id, node_id)).fetchall()

        for e in edges:
            cid = e["cid"]
            if cid not in related:
                n = db.execute("SELECT id, layer, summary, who, timestamp, turn_id, affect_json FROM memory_nodes WHERE id = ?", (cid,)).fetchone()
                if n:
                    decay = 0.7 ** depth
                    related[cid] = dict(n) | {"score": base_score * decay, "depth": depth}
            traverse(cid, depth + 1, base_score * 0.7)

    for ep in entry_points:
        nid = ep["id"]
        related[nid] = dict(ep) | {"score": ep["score"] or 1, "depth": 0}
        traverse(nid, 1, ep["score"] or 1)

    # Step 3: Rank and get top turns
    ranked = sorted(related.values(), key=lambda x: (-x["score"], -x["timestamp"]))
    turns_seen = set()
    top_turns = []
    for node in ranked:
        tid = node.get("turn_id", "")
        if tid and tid not in turns_seen:
            turns_seen.add(tid)
            top_turns.append(tid)
        if len(top_turns) >= max_results:
            break

    # Step 4: Full reconstruction
    print(f"\n🧠 NIMA Graph Recall: '{query}'")
    print(f"   Scanned {len(visited)} nodes, found {len(ranked)} candidates")
    print(f"   Returning top {len(top_turns)} experiences\n")

    for turn_id in top_turns:
        turn = db.execute("SELECT * FROM memory_turns WHERE turn_id = ?", (turn_id,)).fetchone()
        if not turn:
            continue

        ts = datetime.fromtimestamp(turn["timestamp"] / 1000).strftime("%Y-%m-%d %H:%M")
        affect = json.loads(turn["affect_json"] or "{}")
        dominant = sorted(affect.items(), key=lambda x: -x[1])[:2] if affect else []
        affect_str = ", ".join(f"{k}({v:.2f})" for k, v in dominant) if dominant else "neutral"

        print(f"{'='*60}")
        print(f"📅 {ts}  |  🎭 {affect_str}")
        print(f"{'='*60}")

        for layer_name, col in [("input", "input_node_id"), ("contemplation", "contemplation_node_id"), ("output", "output_node_id")]:
            node_id = turn[col]
            if node_id:
                node = db.execute("SELECT * FROM memory_nodes WHERE id = ?", (node_id,)).fetchone()
                if node:
                    icon = "👂" if layer_name == "input" else "💭" if layer_name == "contemplation" else "💬"
                    who = f" ({node['who']})" if node["who"] and node["who"] != "self" else ""
                    text = node["text"] if full_text else node["summary"]
                    print(f"\n{icon} {layer_name.upper()}{who}:")
                    print(f"   {text}")

        print()

    # Stats
    total = db.execute("SELECT COUNT(*) FROM memory_nodes").fetchone()[0]
    print(f"📊 Graph: {total} total nodes | {db.execute('SELECT COUNT(*) FROM memory_edges').fetchone()[0]} edges | {db.execute('SELECT COUNT(*) FROM memory_turns').fetchone()[0]} turns")

    db.close()


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="NIMA Graph Memory Recall")
    parser.add_argument("query", help="Search query")
    parser.add_argument("--full", action="store_true", help="Show full text instead of summaries")
    parser.add_argument("--top", type=int, default=3, help="Max results (default 3)")
    parser.add_argument("--who", default="", help="Filter by speaker")
    parser.add_argument("--layer", default="", choices=["input", "contemplation", "output", ""], help="Filter by layer")
    args = parser.parse_args()

    recall(args.query, max_results=args.top, full_text=args.full, who_filter=args.who, layer_filter=args.layer)
