# NIMA Core

![NIMA Core Banner](assets/banner.png)

**Neural Integrated Memory Architecture** — A complete memory system for AI agents with emotional intelligence.

> "Memory isn't just storage — it's context, emotion, and connection."

NIMA provides:
- **Graph-based memory** — SQLite or LadybugDB
- **Semantic search** — Voyage, OpenAI, or local embeddings
- **Dynamic affect** — 7-dimensional emotional state (Panksepp)
- **Lazy recall** — Efficient memory injection with dedup
- **Session optimization** — Token budgeting and compression

## 🚀 Quick Start

```bash
# Install
pip install nima-core

# Or with LadybugDB (recommended)
pip install nima-core[vector]

# Set embedding provider
export NIMA_EMBEDDER=voyage
export VOYAGE_API_KEY=your-key
```

```bash
# Install hooks
./install.sh --with-ladybug

# Restart OpenClaw
openclaw restart
```

## 🔒 Privacy & Permissions

**What NIMA accesses:**
- ✅ Session transcripts: `~/.openclaw/agents/*/sessions/*.jsonl` (captures conversation turns)
- ✅ Local storage: `~/.nima/` (SQLite/LadybugDB databases, affect history, embeddings)

**Network calls (conditional):**
- 🌐 **Voyage API** — Only when `NIMA_EMBEDDER=voyage` (sends conversation text for embeddings)
- 🌐 **OpenAI API** — Only when `NIMA_EMBEDDER=openai` (sends conversation text for embeddings)
- 🔒 **Local embeddings** — Default (`NIMA_EMBEDDER=local`), no external network calls

**Opt-in controls:**
```json
// openclaw.json - Fine-grained control
{
  "plugins": {
    "entries": {
      "nima-memory": {
        "enabled": true,
        "skip_subagents": true,      // Don't capture subagent sessions
        "skip_heartbeats": true,      // Don't capture heartbeat checks
        "noise_filtering": {
          "filter_heartbeat_mechanics": true,
          "filter_system_noise": true
        }
      }
    }
  }
}
```

**Privacy-first defaults:**
- Subagent sessions excluded by default
- Heartbeat/system noise filtered
- Local embeddings (no external API calls)
- All data stored locally in `~/.nima/`

**To disable entirely:** Remove `nima-memory` from `plugins.allow` in `openclaw.json`

## 📚 Documentation

| Guide | Description |
|-------|-------------|
| [SETUP_GUIDE.md](./SETUP_GUIDE.md) | Step-by-step installation |
| [AFFECTIVE_CORE_PROFILES_GUIDE.md](./docs/AFFECTIVE_CORE_PROFILES_GUIDE.md) | Archetypes & personality profiles |
| [DATABASE_OPTIONS.md](./docs/DATABASE_OPTIONS.md) | SQLite vs LadybugDB |
| [EMBEDDING_PROVIDERS.md](./docs/EMBEDDING_PROVIDERS.md) | Voyage, OpenAI, Local |
| [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) | Migrate from old versions |
| [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) | Common commands |

## 🧠 Memory System

### Database Options

| Feature | SQLite | LadybugDB |
|---------|--------|-----------|
| **Setup** | Zero config | `pip install real-ladybug` |
| **Size** | ~91 MB | ~50 MB (44% smaller) |
| **Text Search** | 31ms | 9ms (3.4x faster) |
| **Vector Search** | External | Native HNSW (18ms) |
| **Graph Traversal** | JOINs | Native Cypher |

**Recommendation:** Use LadybugDB for production, SQLite for development.

### Embedding Providers

| Provider | Dimensions | Cost | Quality |
|----------|------------|------|---------|
| **Voyage** | 1024 | $0.12/1M | Excellent |
| **OpenAI** | 1536 | $0.13/1M | Excellent |
| **Local** | 384 | Free | Good |

**Recommendation:** Use Voyage for best quality/cost ratio.

### Memory Operations

```bash
# Recall memories
python -m nima_core recall "what did we discuss yesterday" --top 5

# Backfill from session logs  
python -m nima_core backfill --session recent

# Benchmark database
python -m nima_core benchmark

# Migrate to LadybugDB
python scripts/ladybug_parallel.py --migrate
```

## 🎭 Dynamic Affect System

NIMA includes a biologically-inspired emotional intelligence system based on the **Panksepp 7-Affect Model**:

- **SEEKING** — Curiosity, exploration, anticipation
- **RAGE** — Frustration, assertion, boundary-setting
- **FEAR** — Vigilance, caution, protection
- **LUST** — Desire, attraction, motivation
- **CARE** — Nurturing, connection, empathy
- **PANIC** — Separation distress, sensitivity
- **PLAY** — Joy, humor, social bonding

```python
from nima_core import DynamicAffectSystem

# Create agent with "Guardian" personality
affect = DynamicAffectSystem(identity_name="my_bot", baseline="guardian")

# Process incoming message
affect.process_input({"CARE": 0.8, "PLAY": 0.2}, intensity=0.7)

# Get current emotional state
current = affect.current
print(f"Dominant: {current.dominant()}")  # ('CARE', 0.82)
```

### Archetype Presets

| Archetype | Baseline |
|-----------|----------|
| **Guardian** | High CARE, Low PLAY |
| **Explorer** | High SEEKING, Low FEAR |
| **Trickster** | High PLAY, High SEEKING |
| **Empath** | High CARE, High PANIC |
| **Sage** | High SEEKING, Balanced |
| See [DYNAMIC_AFFECT.md](./DYNAMIC_AFFECT.md) for full list |

## 🔌 OpenClaw Hooks

NIMA provides three hooks for OpenClaw:

### nima-memory

Captures memories during conversations.

```json
{
  "events": ["capture"],
  "description": "NIMA Memory Capture",
  "priority": 10
}
```

### nima-recall-live

Injects relevant memories before each response.

```javascript
// Config options
const MAX_RESULTS = 7;           // Max memories per query
const SESSION_TOKEN_BUDGET = 500; // Token limit per session
const USE_LADYBUG = true;         // Use LadybugDB backend
const USE_COMPRESSED_FORMAT = true; // 80% smaller output
```

### nima-affect

Tracks emotional state across conversations.

```json
{
  "events": ["before_agent_start"],
  "description": "NIMA Affect System",
  "priority": 10
}
```

## 📊 Performance

| Operation | SQLite | LadybugDB |
|-----------|--------|-----------|
| Text search | 31.7ms | 9.4ms |
| Vector search | N/A | 18ms |
| Recall overhead | ~180 tokens | ~30 tokens |
| Database size | 91 MB | 50 MB |

## 🔄 Migration from v1.x

```bash
# Backup
cp ~/.nima/memory/graph.sqlite ~/.nima/memory/graph.sqlite.backup

# Update hooks
rm -rf ~/.openclaw/hooks/nima-*
cp -r openclaw_hooks/* ~/.openclaw/extensions/

# (Optional) Migrate to LadybugDB
pip install real-ladybug
python scripts/ladybug_parallel.py --migrate

# Restart
openclaw restart
```

See [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) for details.

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     OpenClaw Gateway                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    nima-recall-live                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Text Search │  │Vector Search│  │   Dedup     │         │
│  │   (9ms)     │  │   (18ms)    │  │ (session)   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 LadybugDB / SQLite                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │Memory Nodes │  │   Edges     │  │ HNSW Index  │         │
│  │   9,428     │  │   4,515     │  │  8,893 emb  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    nima-affect                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  SEEKING │ RAGE │ FEAR │ LUST │ CARE │ PANIC │ PLAY │   │
│  │   0.49   │ 0.12 │ 0.08 │ 0.05 │ 0.48 │ 0.03  │ 0.38 │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT License. Free to use for any AI agent.