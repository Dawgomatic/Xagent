/**
 * NIMA Graph-Augmented Recall
 * ===========================
 * Three-layer lazy reconstruction:
 *   Layer 1: Graph index (always loaded, node IDs + edges)
 *   Layer 2: Compressed summaries (~10-20 tokens each)
 *   Layer 3: Full text (on-demand, only top 2-3)
 *
 * Flow: Query → FTS match → graph traverse → summaries → rank → reconstruct top N
 *
 * Author: NIMA Core Team
 * Date: 2026-02-13
 * Security fixes: 2026-02-13
 */

import { execFileSync } from "node:child_process";
import { execPython } from "../utils/async-python.js"; // Async wrapper
import { join } from "node:path";
import { writeFileSync, unlinkSync, mkdtempSync } from "node:fs";
import os from "node:os";

const GRAPH_DB = join(os.homedir(), ".nima", "memory", "graph.sqlite");

/**
 * Recall memories related to a query using graph-augmented search.
 *
 * @param {string} query - Search query
 * @param {object} options - Recall options
 * @param {number} options.maxResults - Max full memories to return (default 3)
 * @param {number} options.maxSummaries - Max summaries to scan (default 15)
 * @param {number} options.traverseDepth - Graph traversal depth (default 2)
 * @param {string} options.who - Filter by speaker (optional)
 * @param {string} options.layer - Filter by layer: input|contemplation|output (optional)
 * @returns {object} { memories: [...], summaryCount, nodesScanned }
 */
export async function recall(query, options = {}) {
  const maxResults = options.maxResults || 3;
  const maxSummaries = options.maxSummaries || 15;
  const traverseDepth = options.traverseDepth || 2;
  const whoFilter = options.who || "";
  const layerFilter = options.layer || "";

  // Create temp directory and write params as JSON file
  const tmpDir = mkdtempSync(join(os.tmpdir(), "nima-recall-"));
  const paramsFile = join(tmpDir, "params.json");
  
  try {
    const params = {
      graphDb: GRAPH_DB,
      query,
      maxResults,
      maxSummaries,
      traverseDepth,
      whoFilter,
      layerFilter,
    };
    
    writeFileSync(paramsFile, JSON.stringify(params), "utf-8");

    const recallScript = `
import sqlite3, json, sys, os
from collections import deque

# ─── Constants ───
DECAY_FACTOR = 0.7
MAX_SUMMARIES = 15
MAX_VISITED = 500
MAX_FULL_TEXT = 3

# ─── Load params from JSON file ───
params_file = sys.argv[1]
with open(params_file, 'r') as f:
    params = json.load(f)

GRAPH_DB = params['graphDb']
query = params['query']
max_summaries = min(params['maxSummaries'], MAX_SUMMARIES)
max_results = min(params['maxResults'], MAX_FULL_TEXT)
traverse_depth = params['traverseDepth']
who_filter = params['whoFilter']
layer_filter = params['layerFilter']

if not os.path.exists(GRAPH_DB):
    print(json.dumps({"error": "no database", "memories": []}))
    sys.exit(0)

db = sqlite3.connect(GRAPH_DB)
db.row_factory = sqlite3.Row

# ─── Helper: Escape LIKE wildcards ───
def escape_like(text):
    """Escape special LIKE wildcards: %, _, \\"""
    return text.replace('\\\\', '\\\\\\\\').replace('%', '\\\\%').replace('_', '\\\\_')

# ─── Helper: Sanitize FTS5 query ───
def sanitize_fts5(text):
    """Remove FTS5 operators from user input"""
    # Remove/escape: AND, OR, NOT, *, ", NEAR, etc.
    forbidden = ['AND', 'OR', 'NOT', 'NEAR']
    sanitized = text
    for word in forbidden:
        sanitized = sanitized.replace(word, '')
    # Remove special chars that have FTS5 meaning
    sanitized = sanitized.replace('*', '').replace('"', '').replace('(', '').replace(')', '')
    return sanitized.strip()

# ─── Helper: Safe JSON parse ───
def safe_json_parse(json_str):
    """Parse JSON with fallback to empty dict"""
    try:
        return json.loads(json_str or '{}')
    except (json.JSONDecodeError, TypeError):
        return {}

# ─── Step 1: Text search for entry points ───
search_terms = [escape_like(term.lower()) for term in query.lower().split()]
if not search_terms:
    print(json.dumps({"error": "empty query", "memories": []}))
    sys.exit(0)

# Build WHERE clause matching any search term in text or summary
conditions = []
params_list = []
for term in search_terms[:10]:  # Limit to 10 terms
    conditions.append("(LOWER(text) LIKE ? ESCAPE '\\\\' OR LOWER(summary) LIKE ? ESCAPE '\\\\')")
    params_list.extend([f"%{term}%", f"%{term}%"])

where = " OR ".join(conditions)

if who_filter:
    where = f"({where}) AND LOWER(who) LIKE ? ESCAPE '\\\\'"
    params_list.append(f"%{escape_like(who_filter.lower())}%")

if layer_filter:
    # Layer filter is controlled input, but still validate
    if layer_filter in ['input', 'contemplation', 'output']:
        where = f"({where}) AND layer = ?"
        params_list.append(layer_filter)

# Score by number of matching terms (simple relevance)
score_conditions = [f"(CASE WHEN LOWER(text) LIKE ? ESCAPE '\\\\' OR LOWER(summary) LIKE ? ESCAPE '\\\\' THEN 1 ELSE 0 END)" for _ in search_terms[:10]]
score_params = [p for t in search_terms[:10] for p in (f"%{t}%", f"%{t}%")]

entry_points = db.execute(f"""
    SELECT id, layer, summary, who, timestamp, turn_id, affect_json,
           ({' + '.join(score_conditions)}) as score
    FROM memory_nodes 
    WHERE {where}
    ORDER BY score DESC, timestamp DESC
    LIMIT ?
""", params_list + score_params + [max_summaries]).fetchall()

if not entry_points:
    print(json.dumps({"memories": [], "summaryCount": 0, "nodesScanned": 0}))
    sys.exit(0)

# ─── Step 2: Graph traversal (iterative BFS with node limit) ───
visited = set()
related_nodes = {}  # node_id -> {data, depth, score}
queue = deque()  # (node_id, depth, base_score)

# Add entry points
for ep in entry_points:
    node_id = ep['id']
    score = ep['score'] or 1
    related_nodes[node_id] = {
        'id': node_id,
        'layer': ep['layer'],
        'summary': ep['summary'],
        'who': ep['who'],
        'timestamp': ep['timestamp'],
        'turn_id': ep['turn_id'],
        'affect': safe_json_parse(ep['affect_json']),
        'score': score,
        'depth': 0,
        'relation': 'direct_match'
    }
    visited.add(node_id)
    queue.append((node_id, 1, score))

# BFS traversal with hard limit
while queue and len(visited) < MAX_VISITED:
    node_id, depth, base_score = queue.popleft()
    
    if depth > traverse_depth:
        continue
    
    # Get all connected nodes via edges (both directions)
    edges = db.execute("""
        SELECT target_id as connected_id, relation FROM memory_edges WHERE source_id = ?
        UNION
        SELECT source_id as connected_id, relation FROM memory_edges WHERE target_id = ?
    """, (node_id, node_id)).fetchall()
    
    # Collect IDs to batch fetch
    new_ids = [edge['connected_id'] for edge in edges if edge['connected_id'] not in visited]
    
    if not new_ids:
        continue
    
    # Hard cap enforcement
    if len(visited) >= MAX_VISITED:
        break
    
    # Batch fetch connected nodes (fix N+1 problem)
    placeholders = ','.join('?' * len(new_ids))
    connected_nodes = db.execute(f"""
        SELECT id, layer, summary, who, timestamp, turn_id, affect_json 
        FROM memory_nodes 
        WHERE id IN ({placeholders})
    """, new_ids).fetchall()
    
    # Build lookup for edge relations
    edge_relations = {edge['connected_id']: edge['relation'] for edge in edges}
    
    for node_data in connected_nodes:
        connected_id = node_data['id']
        
        if connected_id in visited:
            continue
        
        visited.add(connected_id)
        
        # Hard cap check
        if len(visited) >= MAX_VISITED:
            break
        
        # Score decays with depth
        decay = DECAY_FACTOR ** depth
        related_nodes[connected_id] = {
            'id': connected_id,
            'layer': node_data['layer'],
            'summary': node_data['summary'],
            'who': node_data['who'],
            'timestamp': node_data['timestamp'],
            'turn_id': node_data['turn_id'],
            'affect': safe_json_parse(node_data['affect_json']),
            'score': base_score * decay,
            'depth': depth,
            'relation': edge_relations.get(connected_id, 'related')
        }
        
        # Add to queue for further traversal
        if depth + 1 <= traverse_depth:
            queue.append((connected_id, depth + 1, base_score * DECAY_FACTOR))

# ─── Step 3: Rank and select top N for full reconstruction ───
ranked = sorted(related_nodes.values(), key=lambda x: (-x['score'], -x['timestamp']))

# Group by turn_id to return complete experiences
turns_seen = set()
top_turns = []
for node in ranked:
    turn_id = node.get('turn_id', '')
    if turn_id and turn_id not in turns_seen:
        turns_seen.add(turn_id)
        top_turns.append(turn_id)
    if len(top_turns) >= max_results:
        break

# ─── Step 4: Full reconstruction of top turns ───
memories = []
for turn_id in top_turns:
    turn = db.execute(
        "SELECT * FROM memory_turns WHERE turn_id = ?", (turn_id,)
    ).fetchone()
    
    if not turn:
        continue
    
    # Get full text for all three layers
    layers = {}
    for layer_name, node_col in [('input', 'input_node_id'), ('contemplation', 'contemplation_node_id'), ('output', 'output_node_id')]:
        node_id = turn[node_col]
        if node_id:
            node = db.execute("SELECT * FROM memory_nodes WHERE id = ?", (node_id,)).fetchone()
            if node:
                layers[layer_name] = {
                    'text': node['text'],
                    'summary': node['summary'],
                    'who': node['who'],
                }
    
    memories.append({
        'turn_id': turn_id,
        'timestamp': turn['timestamp'],
        'affect': safe_json_parse(turn['affect_json']),
        'layers': layers,
        'score': related_nodes.get(turn.get('input_node_id', 0), {}).get('score', 0)
    })

# ─── Step 5: Format output ───
result = {
    'memories': memories,
    'summaryCount': len(ranked),
    'nodesScanned': len(visited),
    'query': query,
}

print(json.dumps(result, default=str))
db.close()
`;

    const result = await execPython("python3", ["-c", recallScript, paramsFile], {
      timeout: 15000,
      encoding: "utf-8",
      breakerId: "recall-cli"
    });
    return JSON.parse(result.trim());
  } catch (err) {
    return { error: err.message, memories: [], summaryCount: 0, nodesScanned: 0 };
  } finally {
    // Clean up temp file
    try {
      unlinkSync(paramsFile);
      // Note: tmpDir cleanup would require recursive removal, leave for OS
    } catch (cleanupErr) {
      // Ignore cleanup errors
    }
  }
}

/**
 * Format recalled memories for injection into agent context.
 * Uses compressed summaries for context efficiency.
 *
 * @param {object} recallResult - Result from recall()
 * @param {boolean} fullText - Include full text (default false = summaries only)
 * @returns {string} Formatted memory block for context injection
 */
export function formatForContext(recallResult, fullText = false) {
  if (!recallResult.memories || recallResult.memories.length === 0) {
    return "";
  }

  const lines = [`[NIMA Memory Recall: ${recallResult.memories.length} experiences found]`];

  for (const memory of recallResult.memories) {
    const ts = new Date(memory.timestamp).toISOString().split("T")[0];
    const affect = memory.affect || {};
    const dominant = Object.entries(affect).sort((a, b) => b[1] - a[1])[0];
    const affectTag = dominant ? `${dominant[0]}(${dominant[1].toFixed(2)})` : "neutral";

    lines.push(`\n--- Memory (${ts}) [affect: ${affectTag}] ---`);

    for (const [layerName, layer] of Object.entries(memory.layers || {})) {
      const icon = layerName === "input" ? "👂" : layerName === "contemplation" ? "💭" : "💬";
      const text = fullText ? layer.text : layer.summary;
      if (text) {
        const who = layer.who && layer.who !== "self" ? ` (${layer.who})` : "";
        lines.push(`${icon} ${layerName.toUpperCase()}${who}: ${text}`);
      }
    }
  }

  lines.push(`\n[Scanned ${recallResult.nodesScanned} nodes, ${recallResult.summaryCount} candidates]`);

  return lines.join("\n");
}

/**
 * Quick recall — returns formatted string ready for context injection.
 */
export function quickRecall(query, options = {}) {
  const result = recall(query, options);
  return formatForContext(result, options.fullText || false);
}
