#!/usr/bin/env python3
"""
session-guard: Update stored session ID in state file.

Usage:
  python3 update_session_id.py <session-id> [state-file]

Default state file: ~/clawd/memory/heartbeat-state.json
Override via GUARD_STATE_FILE env var or second argument.
"""
import json
import os
import sys

DEFAULT_STATE = os.path.expanduser("~/clawd/memory/heartbeat-state.json")
STATE_FILE = sys.argv[2] if len(sys.argv) > 2 else os.environ.get("GUARD_STATE_FILE", DEFAULT_STATE)

if len(sys.argv) < 2:
    print(f"Usage: update_session_id.py <session-id> [state-file]")
    sys.exit(1)

new_id = sys.argv[1].strip()
if not new_id:
    print("Error: session-id cannot be empty")
    sys.exit(1)

os.makedirs(os.path.dirname(STATE_FILE), exist_ok=True)

try:
    with open(STATE_FILE) as f:
        data = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    data = {}

data["lastSessionId"] = new_id

with open(STATE_FILE, "w") as f:
    json.dump(data, f, indent=2)

print(f"✓ lastSessionId updated: {new_id}")
