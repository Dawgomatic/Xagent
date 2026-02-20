#!/bin/bash
# session-guard: Detect if OpenClaw main session has changed since last check.
# Portable version — works with any state file location.
#
# Usage:
#   check_session.sh [STATE_FILE] [SESSIONS_DIR]
#
# Outputs: CURRENT_ID|STORED_ID
# Exit 0 = same session (or first run)
# Exit 1 = new session detected
# Exit 2 = error (no sessions found)
#
# Default state file: ~/clawd/memory/heartbeat-state.json
# Override via GUARD_STATE_FILE env var or first argument.

STATE_FILE="${1:-${GUARD_STATE_FILE:-$HOME/clawd/memory/heartbeat-state.json}}"
SESSIONS_DIR="${2:-${GUARD_SESSIONS_DIR:-$HOME/.openclaw/agents/main/sessions}}"

# Find most recent active session (exclude .reset. and .deleted.)
CURRENT_FILE=$(ls -t "$SESSIONS_DIR"/*.jsonl 2>/dev/null | grep -v "\.reset\.\|\.deleted\." | head -1)
CURRENT_ID=$(basename "$CURRENT_FILE" .jsonl 2>/dev/null)

if [ -z "$CURRENT_ID" ]; then
    echo "ERROR: No active session found in $SESSIONS_DIR" >&2
    exit 2
fi

# Get stored session ID
STORED_ID=""
if [ -f "$STATE_FILE" ]; then
    STORED_ID=$(python3 -c "
import json, sys
try:
    with open('$STATE_FILE') as f:
        d = json.load(f)
    print(d.get('lastSessionId', ''))
except Exception:
    print('')
" 2>/dev/null)
fi

echo "${CURRENT_ID}|${STORED_ID}"

if [ -z "$STORED_ID" ]; then
    # First run — store current ID, don't treat as new session
    python3 -c "
import json, os
state_file = '$STATE_FILE'
try:
    with open(state_file) as f:
        data = json.load(f)
except:
    data = {}
data['lastSessionId'] = '$CURRENT_ID'
os.makedirs(os.path.dirname(state_file), exist_ok=True)
with open(state_file, 'w') as f:
    json.dump(data, f, indent=2)
" 2>/dev/null
    exit 0  # First run, not a new session event
fi

if [ "$CURRENT_ID" != "$STORED_ID" ]; then
    exit 1  # New session detected
else
    exit 0  # Same session
fi
