#!/bin/bash
# List all memory topics
# Usage: ./topics.sh

MEMORY_DIR="${AGENT_MEMORY_DIR:-$HOME/.agent-memory}"

if [ ! -d "$MEMORY_DIR" ]; then
    echo "No memories found."
    exit 0
fi

cat "$MEMORY_DIR"/*/*.jsonl 2>/dev/null | \
    node -e "
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
const topics = {};

rl.on('line', line => {
    try {
        const entry = JSON.parse(line);
        topics[entry.topic] = (topics[entry.topic] || 0) + 1;
    } catch (e) {}
});

rl.on('close', () => {
    const sorted = Object.entries(topics).sort((a, b) => b[1] - a[1]);
    console.log('Memory Topics:\n');
    sorted.forEach(([topic, count]) => {
        console.log(\`  \${topic}: \${count} entries\`);
    });
    console.log(\`\nTotal: \${sorted.reduce((a, b) => a + b[1], 0)} memories\`);
});
" 2>/dev/null || grep -roh '"topic":"[^"]*"' "$MEMORY_DIR" | sort | uniq -c | sort -rn
