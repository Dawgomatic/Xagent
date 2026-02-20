# Database Options

NIMA supports two memory backends: SQLite (default) and LadybugDB (graph database).

## SQLite (Default)

**Pros:**
- Zero setup - works out of the box
- No external dependencies
- Portable, single-file database
- Fast for most use cases

**Cons:**
- Limited graph traversal capabilities
- No native vector search
- Slower for complex relationship queries

**Location:** `~/.nima/memory/graph.sqlite`

**Schema:**
- `memory_nodes` - All memories (input, contemplation, output layers)
- `memory_edges` - Relationships between nodes
- `memory_turns` - Conversation turns
- `memory_fts` - Full-text search index

## LadybugDB (Graph Database)

**Pros:**
- Native graph traversal (Cypher queries)
- Better for relationship-heavy queries
- Built-in vector search support
- Concurrent writes with ACID transactions

**Cons:**
- Requires `real_ladybug` Python package
- Slightly larger storage footprint
- More complex setup

**Location:** `~/.nima/memory/ladybug.lbug`

**Installation:**

```bash
pip install real_ladybug

# Or with NIMA
pip install nima-core[vector]
```

**Schema:**
- `MemoryNode` - Nodes with vector embeddings
- `Turn` - Conversation turn nodes
- Relationships: `relates_to`, `has_input`, `has_contemplation`, `has_output`

## Dual-Write Mode

NIMA can write to both backends simultaneously for redundancy and migration.

**Enable in OpenClaw config:**

```json
{
  "plugins": {
    "entries": {
      "nima-memory": {
        "enabled": true,
        "dual_write": true
      }
    }
  }
}
```

Both databases stay in sync. If LadybugDB fails, SQLite continues to work.

## Choosing a Backend

**Use SQLite if:**
- You want zero-setup memory
- Single-agent use case
- Memory size < 10,000 turns
- Hosting on resource-constrained devices

**Use LadybugDB if:**
- You need graph analytics
- Multi-agent memory sharing
- Memory size > 10,000 turns
- Advanced query capabilities (Cypher)
- Vector similarity search

## Migration

To migrate from SQLite to LadybugDB:

```bash
python3 -m nima_core.bridges.migrate_to_ladybug
```

This copies all SQLite data to LadybugDB. Original SQLite database is preserved.

## Performance

**SQLite:**
- Text search: ~30ms (FTS5)
- Memory recall: ~50-100ms (10-20 results)
- Write: ~10ms per turn

**LadybugDB:**
- Graph traversal: ~9ms (3.4x faster than SQLite joins)
- Cypher queries: ~20-50ms
- Concurrent writes: 5 retries with exponential backoff

## Backup

**SQLite:**
```bash
cp ~/.nima/memory/graph.sqlite ~/backups/graph-$(date +%Y%m%d).sqlite
```

**LadybugDB:**
```bash
cp ~/.nima/memory/ladybug.lbug ~/backups/ladybug-$(date +%Y%m%d).lbug
```

## See Also

- [CONCURRENCY.md](../openclaw_hooks/nima-memory/CONCURRENCY.md) - Race condition handling
- [MIGRATION.md](../openclaw_hooks/nima-memory/MIGRATION.md) - SQLite → LadybugDB migration guide
