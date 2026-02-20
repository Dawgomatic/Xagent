#!/bin/bash
# Search memories
# Usage: ./recall.sh "search query"

set -e

QUERY="${1:?Usage: $0 \"search query\"}"
MEMORY_DIR="${AGENT_MEMORY_DIR:-$HOME/.agent-memory}"

if [ ! -d "$MEMORY_DIR" ]; then
    echo "No memories found. Start saving with ./save.sh"
    exit 0
fi

# Search through all jsonl files
grep -r -i -h "$QUERY" "$MEMORY_DIR"/*/*.jsonl 2>/dev/null | \
    node -e "
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
const results = [];

rl.on('line', line => {
    try {
        const entry = JSON.parse(line);
        results.push(entry);
    } catch (e) {}
});

rl.on('close', () => {
    if (results.length === 0) {
        console.log('No memories found for: $QUERY');
        return;
    }
    
    results.sort((a, b) => b.ts - a.ts);
    
    console.log(\`Found \${results.length} memories:\n\`);
    results.slice(0, 10).forEach(m => {
        const date = new Date(m.ts).toISOString().split('T')[0];
        console.log(\`[\${date}] [\${m.topic}] \${m.content}\`);
        if (m.tags?.length) console.log(\`         tags: \${m.tags.join(', ')}\`);
        console.log();
    });
    
    if (results.length > 10) {
        console.log(\`... and \${results.length - 10} more\`);
    }
});
" 2>/dev/null || echo "Install Node.js for formatted output"
