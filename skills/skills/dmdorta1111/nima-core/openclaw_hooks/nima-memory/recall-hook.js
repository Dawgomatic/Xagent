/**
 * NIMA Live Recall Hook
 * ======================
 * Queries experiential memory on EVERY message via before_agent_start.
 * Lightweight: FTS5 search only (no full graph traversal), ~50-100ms.
 * 
 * Hooks:
 *   before_agent_start → query recent user message against graph → prepend memories
 * 
 * Author: NIMA Core Team
 * Date: Feb 14, 2026
 */

import { execFileSync } from "node:child_process";
import { execPython } from "../utils/async-python.js"; // Async wrapper
import { existsSync, writeFileSync, readFileSync, unlinkSync, mkdtempSync } from "node:fs";
import { join } from "node:path";
import os from "node:os";

const GRAPH_DB = join(os.homedir(), ".nima", "memory", "graph.sqlite");
const MAX_RESULTS = 3;
const QUERY_TIMEOUT = 5000; // 5s max — must be fast
const MIN_QUERY_LENGTH = 15; // Skip very short messages
const COOLDOWN_MS = 30000; // Don't re-query within 30s if same topic
const MAX_QUERY_CHARS = 300;

// Simple cache to avoid redundant queries
let lastQuery = "";
let lastQueryTime = 0;
let lastResult = "";

/**
 * Sanitize FTS5 query — escape operators
 */
function sanitizeFTS5(query) {
  return query
    .replace(/["():*^~{}[\]]/g, " ")
    .replace(/\b(AND|OR|NOT|NEAR)\b/gi, " ")
    .trim()
    .split(/\s+/)
    .filter(w => w.length > 2)
    .slice(0, 8)
    .map(w => `"${w.replace(/"/g, "")}"`)
    .join(" OR ");
}

/**
 * Strip channel prefix from message (e.g., [Telegram UserName id:123 ...] → actual message)
 */
function stripChannelPrefix(text) {
  return text.replace(/^\[(?:Telegram|Discord|Signal|SMS|Slack|Matrix|WhatsApp|iMessage|Email)\s+[^\]]*\]\s*/i, "").trim();
}

/**
 * Extract the user's actual message text from the prompt
 */
function extractUserMessage(prompt) {
  if (!prompt || typeof prompt !== "string") return "";
  
  // Get the last meaningful user content — skip system prefixes, affect blocks
  const lines = prompt.split("\n");
  const userLines = [];
  
  for (let i = lines.length - 1; i >= 0 && userLines.length < 5; i--) {
    const line = lines[i].trim();
    if (!line) continue;
    if (line.includes("AFFECT STATE") || line.includes("[Dynamic affect")) continue;
    if (line.startsWith("🎭")) continue;
    // Skip heartbeat prompts
    if (line.includes("HEARTBEAT") || line.includes("heartbeat")) return "";
    // Skip gateway restart messages
    if (line.includes("GatewayRestart")) return "";
    // Skip very short noise
    if (line.length < 10) continue;
    // Skip system metadata lines
    if (line.startsWith("[message_id:")) continue;
    if (line.startsWith("<media:")) continue;
    
    // Strip channel prefix to get actual message content
    const cleaned = stripChannelPrefix(line);
    if (cleaned.length > 3) {
      userLines.unshift(cleaned);
    }
  }
  
  return userLines.join(" ").substring(0, MAX_QUERY_CHARS);
}

/**
 * Run lightweight FTS5-only recall (no graph traversal)
 */
async function quickRecall(query) {
  if (!existsSync(GRAPH_DB)) return null;
  
  const ftsQuery = sanitizeFTS5(query);
  if (!ftsQuery) return null;
  
  // Build Python script for fast FTS5 search
  const tmpDir = mkdtempSync(join(os.tmpdir(), "nima-recall-live-"));
  const queryFile = join(tmpDir, "query.json");
  
  try {
    writeFileSync(queryFile, JSON.stringify({
      db_path: GRAPH_DB,
      fts_query: ftsQuery,
      max_results: MAX_RESULTS
    }));
    
    const script = `
import json, sqlite3, sys, os

with open(sys.argv[1]) as f:
    params = json.load(f)

db = sqlite3.connect(params["db_path"])
db.row_factory = sqlite3.Row

try:
    rows = db.execute("""
        SELECT n.turn_id, n.layer, n.summary, n.who, n.timestamp,
               fts.rank as score
        FROM memory_fts fts
        JOIN memory_nodes n ON fts.rowid = n.id
        WHERE memory_fts MATCH ?
        ORDER BY fts.rank
        LIMIT ?
    """, (params["fts_query"], params["max_results"] * 3)).fetchall()
except Exception:
    rows = []

# Group by turn_id, pick best turns
turns = {}
for r in rows:
    tid = r["turn_id"]
    if tid not in turns:
        turns[tid] = {"layers": {}, "score": 0, "timestamp": r["timestamp"], "who": r["who"]}
    turns[tid]["layers"][r["layer"]] = r["summary"] or ""
    turns[tid]["score"] += abs(r["score"] or 0)

# Sort by score, take top results
ranked = sorted(turns.values(), key=lambda t: t["score"], reverse=True)[:params["max_results"]]

output = []
for t in ranked:
    parts = []
    if t["layers"].get("input"):
        parts.append(f"In: {t['layers']['input'][:150]}")
    if t["layers"].get("output"):
        parts.append(f"Out: {t['layers']['output'][:150]}")
    if parts:
        who = t.get("who", "unknown")
        output.append(f"[{who}] " + " | ".join(parts))

print(json.dumps(output))
db.close()
`;
    
    const result = await execPython("python3", ["-c", script, queryFile], {
      timeout: QUERY_TIMEOUT,
      encoding: "utf-8",
      breakerId: "recall-hook-fts"
    });
    
    return JSON.parse(result);
  } catch (err) {
    console.error(`[nima-recall-live] Query failed: ${err.message}`);
    return null;
  } finally {
    try { unlinkSync(queryFile); } catch {}
    try { require("fs").rmdirSync(tmpDir); } catch {}
  }
}

/**
 * Format memories for context injection
 */
function formatMemories(memories) {
  if (!memories || memories.length === 0) return "";
  
  const lines = memories.map((m, i) => `  ${i + 1}. ${m}`);
  return `\n[NIMA RECALL — relevant memories from past conversations]\n${lines.join("\n")}\n[End recall — use naturally, don't announce]\n`;
}

export default function nimaRecallLivePlugin(api, config) {
  const log = api.log || console;
  const skipSubagents = config?.skipSubagents !== false;
  
  log.info?.("[nima-recall-live] Live recall hook loaded");
  
  api.on("before_agent_start", async (event, ctx) => {
    try {
      // Debug: log that we fired
      console.error(`[nima-recall-live] FIRED. event keys: ${Object.keys(event || {}).join(",")}, ctx keys: ${Object.keys(ctx || {}).join(",")}`);
      console.error(`[nima-recall-live] event.prompt type: ${typeof event?.prompt}, length: ${event?.prompt?.length || 0}`);
      console.error(`[nima-recall-live] event.prompt first 200: ${String(event?.prompt || "").substring(0, 200)}`);
      
      // Skip subagents and heartbeats
      if (skipSubagents && ctx.sessionKey?.includes(":subagent:")) return;
      if (ctx.sessionKey?.includes("heartbeat")) return;
      
      const userMessage = extractUserMessage(event.prompt);
      console.error(`[nima-recall-live] extracted userMessage (${userMessage.length}): ${userMessage.substring(0, 100)}`);
      if (!userMessage || userMessage.length < MIN_QUERY_LENGTH) {
        console.error(`[nima-recall-live] SKIP: too short (${userMessage.length} < ${MIN_QUERY_LENGTH})`);
        return;
      }
      
      // Cooldown — don't query if same topic recently
      const now = Date.now();
      const queryKey = userMessage.substring(0, 100);
      if (queryKey === lastQuery && (now - lastQueryTime) < COOLDOWN_MS) {
        // Return cached result
        if (lastResult) return { prependContext: lastResult };
        return;
      }
      
      const memories = await quickRecall(userMessage);
      const formatted = formatMemories(memories);
      
      // Cache
      lastQuery = queryKey;
      lastQueryTime = now;
      lastResult = formatted;
      
      if (formatted) {
        log.info?.(`[nima-recall-live] Injected ${memories.length} memories`);
        return { prependContext: formatted };
      }
    } catch (err) {
      console.error(`[nima-recall-live] Error: ${err.message}`);
      return undefined;
    }
  }, { priority: 15 }); // After nima-affect (priority 10)
}
