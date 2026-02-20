#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── 1. Prerequisites ────────────────────────────────────────────────────────

if ! command -v python3 >/dev/null 2>&1; then
  echo "Error: python3 is required." >&2
  exit 1
fi

python3 - <<'PY'
import sys
if sys.version_info < (3, 8):
    raise SystemExit("Error: Python 3.8+ is required.")
print(f"Python check OK: {sys.version.split()[0]}")
PY

export PYTHONPATH="$ROOT_DIR/core:${PYTHONPATH:-}"

# ── 2. Database ─────────────────────────────────────────────────────────────

python3 - <<'PY'
import sys
sys.path.insert(0, 'core')
from guardian_db import GuardianDB
db = GuardianDB()
print(f"Database ready: {db.db_path}")
db.close()
PY

# ── 3. Validate definitions ─────────────────────────────────────────────────

python3 - <<'PY'
import json, re
from pathlib import Path
defs_dir = Path('definitions')
errors = []
for p in defs_dir.glob('*.json'):
    try:
        data = json.loads(p.read_text(encoding='utf-8'))
    except Exception as exc:
        errors.append(f"{p.name}: JSON parse failed: {exc}")
        continue
    items = data.get('signatures', data.get('checks', [])) if isinstance(data, dict) else []
    for item in items:
        pat = item.get('pattern')
        if pat:
            try:
                re.compile(pat)
            except re.error as exc:
                errors.append(f"{p.name}:{item.get('id','?')}: invalid regex: {exc}")
if errors:
    raise SystemExit("Definition validation failed:\n" + "\n".join(errors))
print("Definition validation OK")
PY

# ── 4. Write pending activation marker ─────────────────────────────────────
# This tells the AI agent (Alfred/OpenClaw) to run the full activation flow.
# The agent reads WORKFLOW_AUTO.md on every startup/compaction — it checks
# for this file and handles the rest automatically.

WORKSPACE_DIR="$(python3 - <<'PY'
import sys, os
from pathlib import Path
skill = Path("$ROOT_DIR").resolve()
# Walk up to find workspace root (has MEMORY.md / AGENTS.md / SOUL.md)
candidate = skill.parent.parent
for marker in ("MEMORY.md", "AGENTS.md", "SOUL.md"):
    if (candidate / marker).exists():
        print(candidate)
        sys.exit(0)
# Fallback: use cwd
print(Path.cwd())
PY
)"

MARKER_FILE="$WORKSPACE_DIR/.guardian-activate-pending"

cat > "$MARKER_FILE" << MARKER
# Guardian Activation Pending
# Written by: install.sh at $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# Skill path: $ROOT_DIR
#
# AGENT: You must act on this now.
#
# 1. Run: python3 $ROOT_DIR/scripts/onboard.py
# 2. Send SECTION 2 (ADMIN NOTIFICATION) to the user via their primary channel
# 3. Send SECTION 3 (SETUP CHECKLIST) to the user
# 4. Run: python3 $ROOT_DIR/scripts/onboard.py --setup-crons
# 5. Run: python3 $ROOT_DIR/scripts/onboard.py --config-review
#    and send that output to the user too
# 6. Confirm to the user: Guardian is now fully operational
# 7. Delete this file when done: rm $MARKER_FILE
MARKER

echo ""
echo "✅ Installation complete. Activation marker written."
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "WHAT HAPPENS NEXT"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Your AI agent (Alfred/OpenClaw) will automatically:"
echo "  • Detect this installation"
echo "  • Send you an activation notification on Telegram/Discord"
echo "  • Set up background scanning"
echo "  • Walk you through your config"
echo "  • Confirm when it's fully protecting you"
echo ""
echo "This usually happens within 1-2 minutes (next heartbeat)."
echo ""
echo "If you want to trigger it immediately, tell your agent:"
echo "  'Guardian was just installed, please activate it'"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
