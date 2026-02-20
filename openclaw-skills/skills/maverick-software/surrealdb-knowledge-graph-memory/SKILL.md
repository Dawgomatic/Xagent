# SurrealDB Knowledge Graph Memory v2.1

A comprehensive knowledge graph memory system with semantic search, episodic memory, working memory, and automatic context injection.

## Description

Use this skill for:
- **Semantic Memory** — Store and retrieve facts with confidence-weighted vector search
- **Episodic Memory** — Record task histories and learn from past experiences
- **Working Memory** — Track active task state with crash recovery
- **Auto-Injection** — Automatically inject relevant context into agent prompts
- **Outcome Calibration** — Facts gain/lose confidence based on task outcomes

**Triggers:** "remember this", "store fact", "what do you know about", "memory search", "find similar tasks", "learn from history"

## Features (v2)

| Feature | Description |
|---------|-------------|
| **Semantic Facts** | Vector-indexed facts with confidence scoring |
| **Episodic Memory** | Task histories with decisions, problems, solutions, learnings |
| **Working Memory** | YAML-based task state that survives crashes |
| **Outcome Calibration** | Facts used in successful tasks gain confidence |
| **Auto-Injection** | Relevant facts/episodes injected into prompts automatically |
| **Entity Extraction** | Automatic entity linking and relationship discovery |
| **Confidence Decay** | Stale facts naturally decay over time |

## Dashboard UI

The Memory tab in the Control dashboard provides a two-column layout:

### Left Column: Dashboard
- **📊 Statistics** — Live counts of facts, entities, relations, and archived items
- **Confidence Bar** — Visual display of average confidence score
- **Sources Breakdown** — Facts grouped by source file
- **🏥 System Health** — Status of SurrealDB, schema, and Python dependencies
- **🔗 DB Studio** — Quick link to SurrealDB's web interface

### Right Column: Operations
- **📥 Knowledge Extraction**
  - *Extract Changes* — Incrementally extract facts from modified files
  - *Find Relations* — Discover semantic relationships between existing facts
  - *Full Sync* — Complete extraction + relation discovery
  - Progress bar with real-time status updates
  
- **🔧 Maintenance**
  - *Apply Decay* — Reduce confidence of stale facts
  - *Prune Stale* — Archive facts below threshold
  - *Full Sweep* — Complete maintenance cycle

- **💡 Tips** — Quick reference for operations

When the system needs setup, an **Installation** section appears with manual controls.

## Prerequisites

1. **SurrealDB** installed and running:
   ```bash
   # Install (one-time)
   ./scripts/install.sh
   
   # Start server
   surreal start --bind 127.0.0.1:8000 --user root --pass root file:~/.openclaw/memory/knowledge.db
   ```

2. **Python dependencies** (use the skill's venv):
   ```bash
   cd /path/to/surrealdb-memory
   python3 -m venv .venv
   source .venv/bin/activate
   pip install surrealdb openai pyyaml
   ```

3. **OpenAI API key** for embeddings (set in OpenClaw config or environment)

4. **mcporter** configured with this skill's MCP server

## MCP Server Setup

Add to your `config/mcporter.json`:

```json
{
  "servers": {
    "surrealdb-memory": {
      "command": ["python3", "/path/to/surrealdb-memory/scripts/mcp-server-v2.py"],
      "env": {
        "OPENAI_API_KEY": "${OPENAI_API_KEY}",
        "SURREAL_URL": "http://localhost:8000",
        "SURREAL_USER": "root",
        "SURREAL_PASS": "root"
      }
    }
  }
}
```

## MCP Tools (11 total)

### Core Tools
| Tool | Description |
|------|-------------|
| `knowledge_search` | Semantic search for facts |
| `knowledge_recall` | Get a fact with full context (relations, entities) |
| `knowledge_store` | Store a new fact |
| `knowledge_stats` | Get database statistics |

### v2 Tools
| Tool | Description |
|------|-------------|
| `knowledge_store_sync` | Store with importance routing (high importance = immediate write) |
| `episode_search` | Find similar past tasks |
| `episode_learnings` | Get actionable learnings from history |
| `episode_store` | Record a completed task episode |
| `working_memory_status` | Get current task state |
| `context_aware_search` | Search with task context boosting |
| `memory_inject` | **Intelligent context injection for prompts** |

### memory_inject Tool

The `memory_inject` tool returns formatted context ready for prompt injection:

```bash
mcporter call surrealdb-memory.memory_inject \
    query="user message" \
    max_facts:7 \
    max_episodes:3 \
    confidence_threshold:0.9 \
    include_relations:true
```

**Output:**
```markdown
## Semantic Memory (Relevant Facts)
📌 [60% relevant, 100% confidence] Relevant fact here...

## Related Entities
• Entity Name (type)

## Episodic Memory (Past Experiences)
✅ Task: Previous task goal [similarity]
   → Key learning from that task
```

## Auto-Injection (Enhanced Loop Integration)

When enabled, memory is automatically injected into every agent turn:

1. **Enable in Mode UI:**
   - Open Control dashboard → Mode tab
   - Scroll to "🧠 Memory & Knowledge Graph" section
   - Toggle "Auto-Inject Context"
   - Configure limits (max facts, max episodes, confidence threshold)

2. **How it works:**
   - On each user message, `memory_inject` is called automatically
   - Relevant facts are searched based on the user's query
   - If average fact confidence < threshold, episodic memories are included
   - Formatted context is injected into the agent's system prompt

3. **Configuration (in Mode settings):**
   | Setting | Default | Description |
   |---------|---------|-------------|
   | Auto-Inject Context | Off | Master toggle |
   | Max Facts | 7 | Maximum semantic facts to inject |
   | Max Episodes | 3 | Maximum episodic memories |
   | Confidence Threshold | 90% | Include episodes when below this |
   | Include Relations | On | Include entity relationships |

## Extraction with Progress Tracking

When you run extraction operations from the UI, you'll see:

1. **Progress Bar** — Visual indicator with percentage
2. **Current Step** — What's being processed (e.g., "Extracting facts from MEMORY.md")
3. **Counter** — Progress like "(3/7)" for file processing
4. **Detail Text** — Sub-step information

The progress updates in real-time via polling. When complete, statistics automatically refresh.

## CLI Commands

```bash
# Activate venv
source .venv/bin/activate

# Store a fact
python scripts/memory-cli.py store "Important fact" --confidence 0.9

# Search
python scripts/memory-cli.py search "query"

# Get stats
python scripts/knowledge-tool.py stats

# Run maintenance
python scripts/memory-cli.py maintain

# Extract from files
python scripts/extract-knowledge.py extract        # Changed files only
python scripts/extract-knowledge.py extract --full # All files
python scripts/extract-knowledge.py discover-relations
```

## Database Schema (v2)

### Tables
- `fact` — Semantic facts with embeddings and confidence
- `entity` — Extracted entities (people, places, concepts)
- `relates_to` — Relationships between facts
- `mentions` — Fact-to-entity links
- `episode` — Task histories with outcomes
- `working_memory` — Active task snapshots

### Key Fields (fact)
- `content` — The fact text
- `embedding` — Vector for semantic search
- `confidence` — Base confidence (0-1)
- `success_count` / `failure_count` — Outcome tracking
- `scope` — global, client, or agent

### Key Fields (episode)
- `goal` — What was attempted
- `outcome` — success, failure, abandoned
- `decisions` — Key decisions made
- `problems` — Problems encountered (structured)
- `solutions` — Solutions applied (structured)
- `key_learnings` — Extracted lessons

## Confidence Scoring

Effective confidence is calculated from:
- **Base confidence** (0.0–1.0)
- **+ Inherited boost** from supporting facts
- **+ Entity boost** from well-established entities
- **+ Outcome adjustment** based on success/failure history
- **- Contradiction drain** from conflicting facts
- **- Time decay** (configurable, ~5% per month)

## Maintenance

### Automated (Cron)
```bash
# Extract facts from memory files (every 6 hours)
0 */6 * * * cd ~/openclaw/skills/surrealdb-memory && source .venv/bin/activate && python scripts/extract-knowledge.py extract

# Discover relations (daily at 3 AM)
0 3 * * * cd ~/openclaw/skills/surrealdb-memory && source .venv/bin/activate && python scripts/extract-knowledge.py discover-relations
```

### Manual (UI)
Use the **Maintenance** section in the Memory tab:
- **Apply Decay** — Reduce confidence of stale facts
- **Prune Stale** — Archive facts below 0.3 confidence
- **Full Sweep** — Run complete maintenance cycle

## Files

### Scripts
| File | Purpose |
|------|---------|
| `mcp-server-v2.py` | MCP server with all 11 tools |
| `mcp-server.py` | Legacy v1 MCP server |
| `episodes.py` | Episodic memory module |
| `working_memory.py` | Working memory module |
| `memory-cli.py` | CLI for manual operations |
| `extract-knowledge.py` | Bulk extraction from files |
| `knowledge-tools.py` | Higher-level extraction |
| `schema-v2.sql` | v2 database schema |
| `migrate-v2.py` | Migration script |

### Integration
| File | Purpose |
|------|---------|
| `openclaw-integration/gateway/memory.ts` | Gateway server methods |
| `openclaw-integration/ui/memory-view.ts` | Memory dashboard UI |
| `openclaw-integration/ui/memory-controller.ts` | UI controller |

## Troubleshooting

**"Connection refused"**
→ Start SurrealDB: `surreal start --bind 127.0.0.1:8000 --user root --pass root file:~/.openclaw/memory/knowledge.db`

**"No MCP servers configured"**
→ Ensure mcporter is run from a directory containing `config/mcporter.json` with the surrealdb-memory server defined

**Memory injection returning null**
→ Check that `OPENAI_API_KEY` is set in the environment
→ Verify SurrealDB is running and schema is initialized

**Empty search results**
→ Run extraction from the UI or via CLI to populate facts from memory files

**Progress bar not updating**
→ Ensure the gateway has been restarted after UI updates
→ Check browser console for polling errors

## Migration from v1

```bash
# Apply v2 schema (additive, won't delete existing data)
./scripts/migrate-v2.sh

# Or manually:
source .venv/bin/activate
python scripts/migrate-v2.py
```

## Stats

Check your knowledge graph via UI (Dashboard section) or CLI:
```bash
mcporter call surrealdb-memory.knowledge_stats
```

Example output:
```json
{
  "facts": 379,
  "entities": 485,
  "relations": 106,
  "episodes": 3,
  "avg_confidence": 0.99
}
```
