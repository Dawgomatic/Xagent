#!/bin/bash
# Save a memory entry
# Usage: ./save.sh "topic" "content" [tags...]

set -e

TOPIC="${1:?Usage: $0 \"topic\" \"content\" [tags...]}"
CONTENT="${2:?Usage: $0 \"topic\" \"content\" [tags...]}"
shift 2
TAGS="$*"

MEMORY_DIR="${AGENT_MEMORY_DIR:-$HOME/.agent-memory}"
YEAR=$(date -u +%Y)
MONTH=$(date -u +%m)
DAY=$(date -u +%d)
TS=$(date -u +%s)000

# Ensure directory exists
mkdir -p "$MEMORY_DIR/$YEAR/$MONTH"

# Build tags array
TAGS_JSON="[]"
if [ -n "$TAGS" ]; then
    TAGS_JSON=$(echo "$TAGS" | tr ' ' '\n' | sed 's/.*/"&"/' | tr '\n' ',' | sed 's/,$//' | sed 's/^/[/' | sed 's/$/]/')
fi

# Escape content for JSON
CONTENT_ESC=$(echo "$CONTENT" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')

# Append entry
echo "{\"ts\":$TS,\"topic\":\"$TOPIC\",\"content\":\"$CONTENT_ESC\",\"tags\":$TAGS_JSON}" >> "$MEMORY_DIR/$YEAR/$MONTH/$DAY.jsonl"

echo "✓ Memory saved: [$TOPIC] $CONTENT_ESC"
