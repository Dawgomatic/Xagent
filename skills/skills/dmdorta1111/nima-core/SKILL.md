---
name: nima-core
description: Neural Integrated Memory Architecture — Graph-based memory with LadybugDB, semantic search, dynamic affect, lazy recall. Production-ready for AI agents. Learn more at nima-core.ai
version: 2.0.12
metadata: {"clawdbot":{"emoji":"","requires":{"bins":["python3","node"],"env":["NIMA_DATA_DIR"]},"optional_env":{"NIMA_EMBEDDER":"voyage|openai|local (default: local)","VOYAGE_API_KEY":"Required when NIMA_EMBEDDER=voyage","OPENAI_API_KEY":"Required when NIMA_EMBEDDER=openai"},"permissions":{"reads":["~/.openclaw/agents/*/sessions/*.jsonl"],"writes":["~/.nima/"],"network":["voyage.ai (conditional)","openai.com (conditional)"]}}}
---

# NIMA Core 2.0

**Neural Integrated Memory Architecture** — A complete memory system for AI agents with emotional intelligence.

**Website:** https://nima-core.ai
**GitHub:** https://github.com/lilubot/nima-core

##  Quick Start

```bash
# Install
pip install nima-core

# Or with LadybugDB (recommended for production)
pip install nima-core[vector]

# Set embedding provider
export NIMA_EMBEDDER=voyage
export VOYAGE_API_KEY=your-key

# Install hooks
./install.sh --with-ladybug

# Restart OpenClaw
openclaw restart
```

##  Privacy & Permissions

**Data Access:**
-  Reads session transcripts from `~/.openclaw/agents/*/sessions/*.jsonl`
-  Writes to local storage at `~/.nima/` (databases, affect history, embeddings)

**Network Calls (conditional on embedder choice):**
-  **Voyage API** — Only when `NIMA_EMBEDDER=voyage` (sends text for embeddings)
-  **OpenAI API** — Only when `NIMA_EMBEDDER=openai` (sends text for embeddings)
-  **Local embeddings** — Default (`NIMA_EMBEDDER=local`), no external API calls

**Opt-in Controls:**
```json
// openclaw.json
{
  "plugins": {
    "entries": {
      "nima-memory": {
        "enabled": true,
        "skip_subagents": true,      // Exclude subagent sessions (default)
        "skip_heartbeats": true,      // Exclude heartbeat checks (default)
        "noise_filtering": {
          "filter_heartbeat_mechanics": true,
          "filter_system_noise": true
        }
      }
    }
  }
}
```

**Privacy Defaults:**
- Subagent sessions excluded
- Heartbeat/system noise filtered  
- Local embeddings (no external calls)
- All data stored locally

**To disable:** Remove `nima-memory` from `plugins.allow` in `openclaw.json`

## What's New in 2.0

### LadybugDB Backend
- **3.4x faster** text search (9ms vs 31ms)
- **Native vector search** with HNSW (18ms)
- **44% smaller** database (50MB vs 91MB)
- **Graph traversal** with Cypher queries

### Security Hardened
- Query sanitization (FTS5, SQL injection prevention)
- Path traversal protection
- Temp file cleanup
- Error handling throughout

### Thread Safe
- Singleton pattern with double-checked locking
- API timeouts (30s Voyage, 10s LadybugDB)
- Connection pooling ready

### 348 Tests
- Full unit test coverage
- Thread safety verified
- Edge cases covered

## Architecture

```text
OPENCLAW HOOKS
├── nima-memory      — Three-layer capture (input/contemplation/output)
├── nima-recall-live — Lazy recall injection (before_agent_start)
└── nima-affect      — Real-time emotion detection

PYTHON CORE
├── nima_core/cognition/
│   ├── dynamic_affect.py     — Panksepp 7-affect system
│   ├── personality_profiles.py — JSON personality configs
│   ├── emotion_detection.py  — Lexicon-based emotion→affect mapping
│   └── archetypes.py         — Baseline affect profiles
└── scripts/
    ├── nima_ladybug_backend.py — LadybugDB CLI
    └── ladybug_parallel.py    — Parallel migration

DATABASE (SQLite or LadybugDB)
├── memory_nodes   — Messages with embeddings
├── memory_edges   — Graph relationships
└── memory_turns   — Conversation turns
```

## Performance

| Metric | SQLite | LadybugDB |
|--------|--------|-----------|
| Text Search | 31ms | **9ms** (3.4x) |
| Vector Search | External | **18ms** (native) |
| Database Size | 91MB | **50MB** (44% smaller) |
| Context Tokens | ~180 | **~30** (6x smaller) |

## API

```python
from nima_core import DynamicAffectSystem, get_affect_system

# Get singleton instance (thread-safe)
affect = get_affect_system(identity_name="lilu")

# Process input and get affect state
state = affect.process_input("I'm so excited about this project!")
print(state.current)  # {"SEEKING": 0.72, "PLAY": 0.65, ...}

# Recall memories (via hooks - automatic)
# Or manually via CLI:
# nima-query who_search "David" --limit 5
# nima-query text_search "project" --limit 5
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `NIMA_DATA_DIR` | `~/.nima` | Memory storage path |
| `NIMA_EMBEDDER` | `voyage` | `voyage`, `openai`, or `local` |
| `VOYAGE_API_KEY` | — | Required for Voyage |
| `NIMA_LADYBUG` | `0` | Set `1` for LadybugDB backend |

## Hooks

### nima-memory (Capture)
- Captures input, contemplation, output on every turn
- Stores to SQLite or LadybugDB
- Computes and stores embeddings

### nima-recall-live (Recall)
- Injects relevant memories before agent starts
- Lazy loading — only top N results
- Deduplicates with injected context

### nima-affect (Emotion)
- Real-time emotion detection from text
- Maintains Panksepp 7-affect state
- Modulates response style

## Installation Options

### SQLite (Development)
```bash
pip install nima-core
./install.sh
```

### LadybugDB (Production)
```bash
pip install nima-core[vector]
./install.sh --with-ladybug
```

## Documentation

| Guide | Description |
|-------|-------------|
| [README.md](./README.md) | Full system overview |
| [SETUP_GUIDE.md](./SETUP_GUIDE.md) | Step-by-step installation |
| [docs/DATABASE_OPTIONS.md](./docs/DATABASE_OPTIONS.md) | SQLite vs LadybugDB |
| [docs/EMBEDDING_PROVIDERS.md](./docs/EMBEDDING_PROVIDERS.md) | Voyage, OpenAI, Local |
| [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) | Migrate from old versions |

## Security & Privacy

### Data Access
This plugin accesses:
- `~/.openclaw/agents/.../*.jsonl` — Session transcripts (for memory capture)
- `~/.nima/` — Local memory database (SQLite or LadybugDB)
- `~/.openclaw/extensions/` — Hook installation

### Network Calls
Embeddings are sent to external APIs:
- **Voyage AI** (`api.voyageai.com`) — Default embedding provider
- **OpenAI** (`api.openai.com`) — Optional embedding provider
- **Local** — No external calls when using sentence-transformers

### Required Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `NIMA_EMBEDDER` | `voyage`, `openai`, or `local` | No (default: voyage) |
| `VOYAGE_API_KEY` | Voyage AI authentication | If using Voyage |
| `OPENAI_API_KEY` | OpenAI authentication | If using OpenAI |
| `NIMA_DATA_DIR` | Memory storage path | No (default: ~/.nima) |
| `NIMA_LADYBUG` | Use LadybugDB backend | No (default: 0) |

### Installation Script
The `install.sh` script:
1. Checks for Python 3 and Node.js
2. Creates `~/.nima/` directories
3. Installs Python packages via pip
4. Copies hooks to `~/.openclaw/extensions/`

**No external downloads.** All packages come from PyPI.

---

## Changelog

### v2.0.3 — Security Hardening (Feb 15, 2026)
- **Security:** Fixed path traversal vulnerability in affect_history.py (CRITICAL)
- **Security:** Fixed temp file resource leaks in 3 files (HIGH)
- **Fixed:** Corrected non-existent json.JSONEncodeError → TypeError/ValueError
- **Improved:** Exception handling - replaced 5 generic catches with specific types
- **Quality:** Better error visibility and debugging throughout

### v2.0.1 — Thread Safety + Metadata
- **Fixed:** Thread-safe singleton with double-checked locking
- **Security:** Clarified metadata requirements (Node.js, env vars)
- **Docs:** Added security disclosure for API key usage

### v2.0.0 — LadybugDB + Security
- **Added:** LadybugDB backend with HNSW vector search
- **Added:** Native graph traversal with Cypher
- **Added:** nima-query CLI for unified queries
- **Security:** SQL/FTS5 injection prevention
- **Security:** Path traversal protection
- **Security:** Temp file cleanup
- **Fixed:** Thread-safe singleton initialization
- **Fixed:** API timeouts (Voyage 30s, LadybugDB 10s)
- **Tests:** 348 tests passing
- **Performance:** 3.4x faster text search, 44% smaller DB

### v1.2.1 — Consciousness Architecture
- Added: 8 consciousness systems (Φ, Global Workspace, self-awareness)
- Added: Sparse Block VSA memory
- Added: ConsciousnessCore unified interface

### v1.1.9 — Hook Efficiency Fix
- Fixed: nima-recall hook spawning new Python process every bootstrap
- Performance: ~50-250x faster hook recall

### v1.2.0 — Affective Response Engines
- Added: 4 Layer-2 composite affect engines
- Added: Async affective processing
- Added: Voyage AI embedding support