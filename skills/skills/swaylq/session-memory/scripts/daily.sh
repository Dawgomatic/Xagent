#!/bin/bash
# Get memories from a specific day
# Usage: ./daily.sh [YYYY-MM-DD]

DATE="${1:-$(date -u +%Y-%m-%d)}"
MEMORY_DIR="${AGENT_MEMORY_DIR:-$HOME/.agent-memory}"

YEAR=$(echo "$DATE" | cut -d- -f1)
MONTH=$(echo "$DATE" | cut -d- -f2)
DAY=$(echo "$DATE" | cut -d- -f3)

FILE="$MEMORY_DIR/$YEAR/$MONTH/$DAY.jsonl"

if [ ! -f "$FILE" ]; then
    echo "No memories for $DATE"
    exit 0
fi

echo "Memories for $DATE:"
echo

cat "$FILE" | node -e "
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });

rl.on('line', line => {
    try {
        const m = JSON.parse(line);
        const time = new Date(m.ts).toISOString().split('T')[1].slice(0, 5);
        console.log(\`[\${time}] [\${m.topic}] \${m.content}\`);
    } catch (e) {}
});
" 2>/dev/null || cat "$FILE"
