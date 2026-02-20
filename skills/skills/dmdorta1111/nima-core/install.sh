#!/bin/bash
# NIMA Core Installation Script
# Usage: ./install.sh [--with-ladybug] [--with-local-embedder]

set -e

echo "🧠 NIMA Core Installer"
echo "======================"

# Defaults
INSTALL_LADYBUG=false
LOCAL_EMBEDDER=false

# Parse arguments
for arg in "$@"; do
    case $arg in
        --with-ladybug)
            INSTALL_LADYBUG=true
            shift
            ;;
        --with-local-embedder)
            LOCAL_EMBEDDER=true
            shift
            ;;
    esac
done

# Check prerequisites
echo ""
echo "📋 Checking prerequisites..."

if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "❌ Node.js is required"
    exit 1
fi

echo "✅ Prerequisites OK"

# Create directories
echo ""
echo "📁 Creating directories..."
mkdir -p ~/.nima/memory
mkdir -p ~/.nima/affect
mkdir -p ~/.openclaw/extensions

echo "✅ Directories created"

# Install Python dependencies
echo ""
echo "📦 Installing Python dependencies..."
pip install -q numpy pandas

if [ "$INSTALL_LADYBUG" = true ]; then
    echo "📦 Installing LadybugDB..."
    pip install -q real-ladybug
fi

if [ "$LOCAL_EMBEDDER" = true ]; then
    echo "📦 Installing sentence-transformers..."
    pip install -q sentence-transformers
fi

echo "✅ Python dependencies installed"

# Install hooks
echo ""
echo "🔌 Installing OpenClaw hooks..."

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Copy hooks to extensions
cp -r "$SCRIPT_DIR/openclaw_hooks/nima-memory" ~/.openclaw/extensions/
cp -r "$SCRIPT_DIR/openclaw_hooks/nima-recall-live" ~/.openclaw/extensions/
cp -r "$SCRIPT_DIR/openclaw_hooks/nima-affect" ~/.openclaw/extensions/

echo "✅ Hooks installed"

# Configure OpenClaw
echo ""
echo "⚙️ Configuring OpenClaw..."

CONFIG_FILE="$HOME/.openclaw/openclaw.json"

if [ -f "$CONFIG_FILE" ]; then
    echo "⚠️ Config file exists, please add manually:"
    echo ""
    echo 'Add to plugins section:'
    echo '  "plugins": {'
    echo '    "slots": {'
    echo '      "memory": "nima-memory"'
    echo '    }'
    echo '  }'
else
    echo "⚠️ Config file not found, skipping auto-config"
fi

# Initialize database
echo ""
echo "🗄️ Initializing database..."
python3 -c "
import sqlite3
import os

db_path = os.path.expanduser('~/.nima/memory/graph.sqlite')
conn = sqlite3.connect(db_path)

# Create tables
conn.executescript('''
CREATE TABLE IF NOT EXISTS memory_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    layer TEXT NOT NULL,
    text TEXT NOT NULL,
    summary TEXT NOT NULL,
    who TEXT DEFAULT '',
    affect_json TEXT DEFAULT '{}',
    session_key TEXT DEFAULT '',
    conversation_id TEXT DEFAULT '',
    turn_id TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    embedding BLOB DEFAULT NULL,
    fe_score REAL DEFAULT 0.5
);

CREATE TABLE IF NOT EXISTS memory_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    relation TEXT NOT NULL,
    weight REAL DEFAULT 1.0,
    FOREIGN KEY (source_id) REFERENCES memory_nodes(id),
    FOREIGN KEY (target_id) REFERENCES memory_nodes(id)
);

CREATE TABLE IF NOT EXISTS memory_turns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    turn_id TEXT UNIQUE NOT NULL,
    input_node_id INTEGER,
    contemplation_node_id INTEGER,
    output_node_id INTEGER,
    timestamp INTEGER NOT NULL,
    affect_json TEXT DEFAULT '{}',
    FOREIGN KEY (input_node_id) REFERENCES memory_nodes(id),
    FOREIGN KEY (contemplation_node_id) REFERENCES memory_nodes(id),
    FOREIGN KEY (output_node_id) REFERENCES memory_nodes(id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
    text, summary, who, layer,
    content=memory_nodes,
    content_rowid=id
);

-- FTS5 sync triggers to keep search index in sync with memory_nodes
CREATE TRIGGER IF NOT EXISTS memory_fts_insert AFTER INSERT ON memory_nodes BEGIN
    INSERT INTO memory_fts(rowid, text, summary, who, layer)
    VALUES (NEW.id, NEW.text, NEW.summary, NEW.who, NEW.layer);
END;

CREATE TRIGGER IF NOT EXISTS memory_fts_update AFTER UPDATE ON memory_nodes BEGIN
    UPDATE memory_fts SET text=NEW.text, summary=NEW.summary, who=NEW.who, layer=NEW.layer
    WHERE rowid=NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS memory_fts_delete AFTER DELETE ON memory_nodes BEGIN
    INSERT INTO memory_fts(memory_fts, rowid, text, summary, who, layer)
    VALUES ('delete', OLD.id, OLD.text, OLD.summary, OLD.who, OLD.layer);
END;

CREATE INDEX IF NOT EXISTS idx_nodes_timestamp ON memory_nodes(timestamp);
CREATE INDEX IF NOT EXISTS idx_nodes_layer ON memory_nodes(layer);
CREATE INDEX IF NOT EXISTS idx_nodes_who ON memory_nodes(who);
CREATE INDEX IF NOT EXISTS idx_edges_source ON memory_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON memory_edges(target_id);
''')

conn.commit()
conn.close()
print('✅ Database initialized')
"

# Migrate to LadybugDB if requested
if [ "$INSTALL_LADYBUG" = true ]; then
    echo ""
    echo "🔄 Migrating to LadybugDB..."
    if [ -f "$SCRIPT_DIR/scripts/ladybug_parallel.py" ]; then
        python3 "$SCRIPT_DIR/scripts/ladybug_parallel.py" --migrate || echo "⚠️ Migration had issues, SQLite will be used as fallback"
    else
        echo "⚠️ Migration script not found, skipping. Run manually: python scripts/ladybug_parallel.py --migrate"
    fi
fi

# Summary
echo ""
echo "🎉 Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Set your embedding provider:"
echo "     export NIMA_EMBEDDER=voyage"
echo "     export VOYAGE_API_KEY=your-api-key"
echo ""
echo "  2. Restart OpenClaw:"
echo "     openclaw restart"
echo ""
echo "  3. Verify installation:"
echo "     python3 -c \"from nima_core.db import get_stats; print(get_stats())\""
echo ""
echo "📚 Documentation: docs/"
echo "🐛 Issues: https://github.com/your-org/nima-core/issues"