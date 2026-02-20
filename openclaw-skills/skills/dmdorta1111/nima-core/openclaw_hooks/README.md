# NIMA OpenClaw Hooks

Unified memory and affect system for OpenClaw-powered agents.

## Architecture

```
Message in → nima-affect (emotion detection)
           → agent processes
           → agent_end → nima-memory (three-layer capture to SQLite graph)

Session start → nima-recall (FTS5 + graph traversal → context injection)
```

## Components

### `nima-memory/` — Unified Capture & Recall Plugin
**Type:** OpenClaw extension (`~/.openclaw/extensions/nima-memory/`)  
**Event:** `agent_end`

Captures every conversation turn as a three-layer experience:
- **Input** — What was received (user message + sender info)
- **Contemplation** — What was thought (thinking/reasoning blocks)
- **Output** — What was said (response text)

Each layer is stored as a node in an SQLite graph with edges connecting them.
Affect state is bound to each turn.

**Files:**
| File | Purpose |
|------|---------|
| `index.js` | Capture plugin (agent_end hook) |
| `recall.js` | Graph-augmented recall engine (FTS5 + traversal) |
| `cli_recall.py` | CLI tool for querying memories |
| `backfill.py` | Ingest historical session transcripts |
| `openclaw.plugin.json` | Plugin manifest |
| `package.json` | Node.js package metadata |

### `nima-recall` (separate hook)
**Type:** OpenClaw internal hook (`~/.openclaw/hooks/nima-recall/`)  
**Event:** `agent.bootstrap`

Queries the memory graph at session startup and injects relevant memories
into agent context as `NIMA_RECALL.md`.

### `nima-affect` (separate extension)
**Type:** OpenClaw extension (`~/.openclaw/extensions/nima-affect/`)  
**Events:** `message_received`, `before_agent_start`, `agent_end`

Detects emotions via keyword matching against Panksepp's 7 affects,
updates dynamic affect state with EMA blending, and injects affect
context into agent prompts.

## Database

SQLite graph at `~/.nima/memory/graph.sqlite`:
- **Tables:** `memory_nodes`, `memory_edges`, `memory_turns`
- **FTS5:** `memory_fts` (full-text search with auto-sync triggers)
- **Mode:** WAL (concurrent reads, serialized writes)

## Security (Hardened Feb 13, 2026)

- ✅ No SQL injection — data passed via temp JSON files
- ✅ Bounded traversal — 500-node cap, iterative BFS
- ✅ Transaction wrapping — atomic multi-insert
- ✅ LIKE wildcard escaping
- ✅ FTS5 query sanitization
- ✅ Content length limits (3000 chars text, 300 chars summary)
- ✅ Error logging (no silent failures)
- ✅ Init retry on failure

---

## Fresh Install

### 1. Install nima-memory plugin

```bash
# Create extension directory
mkdir -p ~/.openclaw/extensions/nima-memory

# Copy plugin files
cp openclaw_hooks/nima-memory/index.js ~/.openclaw/extensions/nima-memory/
cp openclaw_hooks/nima-memory/recall.js ~/.openclaw/extensions/nima-memory/
cp openclaw_hooks/nima-memory/cli_recall.py ~/.openclaw/extensions/nima-memory/
cp openclaw_hooks/nima-memory/backfill.py ~/.openclaw/extensions/nima-memory/

# Create plugin manifest
cat > ~/.openclaw/extensions/nima-memory/openclaw.plugin.json << 'EOF'
{
  "name": "nima-memory",
  "version": "1.0.0",
  "description": "NIMA Unified Memory — Three-layer experiential capture and graph-augmented recall",
  "main": "index.js",
  "type": "module"
}
EOF

# Create package.json
echo '{"name":"nima-memory","version":"1.0.0","type":"module","main":"index.js"}' \
  > ~/.openclaw/extensions/nima-memory/package.json
```

### 2. Install nima-recall hook

```bash
mkdir -p ~/.openclaw/hooks/nima-recall

# Copy handler (TypeScript — OpenClaw compiles it automatically)
cp openclaw_hooks/nima-memory/recall.js ~/.openclaw/hooks/nima-recall/
# OR use the dedicated handler.ts if available in your install
```

### 3. Register in OpenClaw config

Add to `~/.openclaw/openclaw.json`:

```json
{
  "plugins": {
    "allow": ["nima-memory"],
    "entries": {
      "nima-memory": {
        "enabled": true
      }
    }
  },
  "hooks": {
    "internal": {
      "entries": {
        "nima-recall": {
          "enabled": true
        }
      }
    }
  }
}
```

### 4. Backfill existing conversations (optional)

```bash
python3 ~/.openclaw/extensions/nima-memory/backfill.py
```

### 5. Restart gateway

```bash
openclaw gateway restart
```

---

## Upgrading from Legacy Hooks

If you previously installed `nima-capture`, `nima-bootstrap`, `nima-recall` as separate hooks, or the old `openclaw_plugin/` directory, follow these steps:

### What to REMOVE

These are all replaced by the unified `nima-memory` plugin:

```bash
# Old capture hook (replaced by nima-memory plugin)
rm -rf ~/.openclaw/hooks/nima-capture

# Old bootstrap hook (merged into nima-recall)
rm -rf ~/.openclaw/hooks/nima-bootstrap

# Old plugin directory (pre-extensions system)
rm -rf /path/to/nima-core/openclaw_plugin
```

#### ⚠️ CRITICAL: Remove old heartbeat capture service

If you previously used `lilu_core/services/heartbeat.py` (or any standalone
heartbeat capture script), **you MUST stop and remove it**. The old heartbeat
service captures ALL session messages WITHOUT filtering, causing:

- **Massive duplication** (thousands of HEARTBEAT_OK entries)
- **System noise stored as memories** (affect states, NIMA recalls, cron outputs)
- **Database bloat** (80%+ of entries may be spam)

The `nima-memory` plugin handles ALL memory capture with proper filtering
(Free Energy scoring, noise detection, deduplication). Running both systems
simultaneously will fill your database with unfiltered garbage.

```bash
# 1. Kill the old heartbeat process
pkill -f "heartbeat.py" || true

# 2. Remove the LaunchAgent (macOS) if it exists
launchctl unload ~/Library/LaunchAgents/ai.lilu.memory-heartbeat.plist 2>/dev/null
rm -f ~/Library/LaunchAgents/ai.lilu.memory-heartbeat.plist

# 3. Remove systemd service (Linux) if it exists
systemctl --user stop lilu-heartbeat 2>/dev/null
systemctl --user disable lilu-heartbeat 2>/dev/null
rm -f ~/.config/systemd/user/lilu-heartbeat.service

# 4. Remove or rename the old script
mv lilu_core/services/heartbeat.py lilu_core/services/heartbeat.py.deprecated
```

**Verify it's gone:**
```bash
# Should return nothing:
ps aux | grep "heartbeat.py" | grep -v grep
ls ~/Library/LaunchAgents/*heartbeat* 2>/dev/null
```

Remove from `openclaw.json` hooks config:
```json
// DELETE these entries if they exist:
"nima-capture": { "enabled": true }
"nima-bootstrap": { "enabled": true }
```

### What to KEEP

```
~/.openclaw/hooks/nima-recall/         — UPDATE to new handler.ts
~/.openclaw/extensions/nima-affect/    — No changes needed
~/.nima/memory/graph.sqlite            — Your data. Never delete.
~/.nima/affect/                        — Affect state. Keep.
```

### What to ADD

```
~/.openclaw/extensions/nima-memory/    — NEW unified plugin (see Fresh Install above)
```

### Config changes

**Before** (legacy):
```json
{
  "hooks": {
    "internal": {
      "entries": {
        "nima-capture": { "enabled": true },
        "nima-bootstrap": { "enabled": true },
        "nima-recall": { "enabled": true }
      }
    }
  }
}
```

**After** (unified):
```json
{
  "plugins": {
    "allow": ["nima-memory"],
    "entries": {
      "nima-memory": { "enabled": true }
    }
  },
  "hooks": {
    "internal": {
      "entries": {
        "nima-recall": { "enabled": true }
      }
    }
  }
}
```

### Migration checklist

- [ ] **Kill old heartbeat.py process** (`pkill -f heartbeat.py`)
- [ ] **Remove heartbeat LaunchAgent/systemd service** (see above)
- [ ] **Remove or rename heartbeat.py** (`.deprecated`)
- [ ] Install `nima-memory` plugin (see Fresh Install)
- [ ] Update `nima-recall` hook handler
- [ ] Remove `nima-capture` hook directory
- [ ] Remove `nima-bootstrap` hook directory (if exists)
- [ ] Remove `openclaw_plugin/` directory (if exists)
- [ ] Update `openclaw.json` (remove old entries, add plugin)
- [ ] Run `openclaw doctor --non-interactive`
- [ ] Restart gateway: `openclaw gateway restart`
- [ ] Verify: Send a test message, check `~/.nima/memory/graph.sqlite` for new entries
- [ ] **Verify no old capture running:** `ps aux | grep heartbeat.py`

### Verify cleanup

```bash
# Should NOT exist:
ls ~/.openclaw/hooks/nima-capture      # Should fail
ls ~/.openclaw/hooks/nima-bootstrap    # Should fail

# SHOULD exist:
ls ~/.openclaw/extensions/nima-memory/index.js       # Capture plugin
ls ~/.openclaw/extensions/nima-memory/recall.js       # Recall engine
ls ~/.openclaw/extensions/nima-affect/index.js        # Affect plugin
ls ~/.openclaw/extensions/nima-recall-live/index.js   # Recall hook
ls ~/.nima/memory/graph.sqlite                         # Memory database
```

---

## CLI Usage

### Query memories
```bash
python3 ~/.openclaw/extensions/nima-memory/cli_recall.py "search terms"
python3 ~/.openclaw/extensions/nima-memory/cli_recall.py "topic" --top 5 --full
python3 ~/.openclaw/extensions/nima-memory/cli_recall.py "David" --who David --layer input
```

### Backfill sessions
```bash
python3 ~/.openclaw/extensions/nima-memory/backfill.py
```

### Check database stats
```bash
python3 -c "
import sqlite3, os
db = sqlite3.connect(os.path.expanduser('~/.nima/memory/graph.sqlite'))
print(f'Nodes: {db.execute(\"SELECT COUNT(*) FROM memory_nodes\").fetchone()[0]}')
print(f'Turns: {db.execute(\"SELECT COUNT(*) FROM memory_turns\").fetchone()[0]}')
print(f'Edges: {db.execute(\"SELECT COUNT(*) FROM memory_edges\").fetchone()[0]}')
print(f'FTS5:  {db.execute(\"SELECT COUNT(*) FROM memory_fts\").fetchone()[0]}')
db.close()
"
```
