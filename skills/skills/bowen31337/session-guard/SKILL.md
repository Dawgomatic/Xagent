---
name: session-guard
description: "Prevents and recovers from OpenClaw session bloat and context loss. Use when: (1) diagnosing why main session context was lost or truncated, (2) setting up isolated heartbeat/monitoring to avoid session pollution, (3) detecting OpenClaw restarts and rehydrating context automatically, (4) auditing whether built-in heartbeat is causing session file bloat, (5) configuring safe compaction settings, (6) any mention of session reset, context lost, output cut off, or heartbeat pollution."
---

# Session Guard

Fixes the OpenClaw heartbeat-in-main-session architectural gap that causes session files to bloat, corrupt, and reset — losing all agent context.

## The Problem

OpenClaw's built-in `heartbeat` runs exclusively in the main session. Every heartbeat turn accumulates as conversation history, inflating the session `.jsonl` file indefinitely. With hourly heartbeats over 2+ days, this reaches 10–15MB, corrupting the file header and triggering an automatic session reset — silently wiping all context.

**Secondary bug**: When heartbeat returns `HEARTBEAT_OK`, OpenClaw strips it but still tries to forward an empty string to messaging platforms → `sendMessage error: message text is empty` spam. Unfixable from agent side.

## Quick Audit

Run to detect issues:

```bash
python3 skills/session-guard/scripts/audit.py
```

Output: lists config antipatterns (heartbeat enabled, safeguard compaction) and session file sizes.

## Fix: Disable Built-in Heartbeat

If audit finds `heartbeat.every` is set (non-zero), patch the config:

```python
# Via gateway tool:
gateway(action="config.patch", raw=json.dumps({
    "agents": {
        "defaults": {
            "heartbeat": {"every": "0m"},
            "compaction": {"mode": "default"}
        }
    }
}), note="Disabled main-session heartbeat to prevent bloat")
```

## Fix: Create Isolated Heartbeat Cron

Replace the disabled built-in heartbeat with an isolated cron job. Use a cheap model. The isolated session reads HEARTBEAT.md and sends Telegram alerts directly via `message` tool (isolated sessions don't auto-deliver to channels).

```python
cron(action="add", job={
    "name": "Isolated Heartbeat",
    "schedule": {"kind": "every", "everyMs": 3600000},  # 1h
    "payload": {
        "kind": "agentTurn",
        "model": "anthropic-proxy-4/glm-4.7",  # cheap model
        "message": "Read HEARTBEAT.md and follow it. Send Telegram alerts via message tool for anything urgent. Do NOT reply HEARTBEAT_OK — isolated sessions must use message tool to notify.",
        "timeoutSeconds": 120
    },
    "sessionTarget": "isolated"
})
```

## Fix: Session Wake Detection

To detect when OpenClaw resets the session and re-inject context automatically:

**Step 1**: Set up the wake monitor cron (runs every 5 min on cheapest model):

```python
cron(action="add", job={
    "name": "Session Wake Monitor",
    "schedule": {"kind": "every", "everyMs": 300000},  # 5min
    "payload": {
        "kind": "agentTurn",
        "model": "nvidia-nim/qwen/qwen2.5-7b-instruct",
        "message": """Check if main session has reset:
1. Run: bash skills/session-guard/scripts/check_session.sh
   Output: CURRENT_ID|STORED_ID. Exit 0=same, 1=new, 2=error.
2. If exit 1 (new session):
   a. Update ID: python3 skills/session-guard/scripts/update_session_id.py <CURRENT_ID>
   b. Notify main session via sessions_send to trigger hydration.
3. If exit 0: do nothing, reply DONE.""",
        "timeoutSeconds": 60
    },
    "sessionTarget": "isolated"
})
```

**Step 2**: Add session wake detection to HEARTBEAT.md so every heartbeat also checks:

```markdown
## Session Wake Detection (run first on every heartbeat)
1. bash memory/scripts/check_new_session.sh
2. If exit 1: hydrate context (read today's daily notes, search tiered memory), update ID
```

**Step 3**: Initialize stored session ID (first time only):

```bash
# Get current session ID
ls -t ~/.openclaw/agents/main/sessions/*.jsonl | grep -v '\.reset\.' | head -1 | xargs basename | sed 's/\.jsonl//'
# Then store it:
python3 skills/session-guard/scripts/update_session_id.py <ID>
```

## Monitoring Session Size

Check if current sessions are bloating:

```bash
python3 skills/session-guard/scripts/audit.py --warn-mb 3
```

Thresholds: warn at 5MB, critical at 10MB. A healthy active session stays under 2MB with `compaction: "default"`.

## Hydration: Re-inject Context After Session Reset

When a session reset is detected, run hydration to rebuild context:

```bash
python3 skills/session-guard/scripts/hydrate.py
```

This loads and concatenates:
1. **Daily notes** — last 2 days from `memory/YYYY-MM-DD.md`
2. **Tiered memory** — top-3 relevant nodes via tree search
3. **MEMORY.md** — first 2000 chars of long-term memory

Output is a structured markdown summary. Read it, synthesize the key context, then notify the user that the session was reset and you've reloaded context.

**Options:**
```bash
python3 hydrate.py --days 3              # load 3 days of notes (default: 2)
python3 hydrate.py --memory-limit 5     # fetch 5 tiered memory results (default: 3)
python3 hydrate.py --workspace /path    # explicit workspace (default: auto-detect ~/clawd)
```

**Full wake detection + hydration flow (used in Session Wake Monitor cron):**

```
1. bash skills/session-guard/scripts/check_session.sh
   → exit 0: same session, skip
   → exit 1: NEW SESSION — proceed with hydration

2. python3 skills/session-guard/scripts/hydrate.py > /tmp/hydration.txt
   cat /tmp/hydration.txt  # read and synthesize

3. python3 skills/session-guard/scripts/update_session_id.py <CURRENT_ID>

4. Notify user (via message tool in isolated sessions):
   "🔄 Session reset detected — context reloaded. [brief summary of key projects/state]"
```

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/audit.py` | Audit config + session sizes. Args: `--config`, `--sessions-dir`, `--warn-mb`, `--json` |
| `scripts/check_session.sh` | Detect session ID change. Exit 0=same, 1=new, 2=error. Args: [state_file] [sessions_dir] |
| `scripts/update_session_id.py` | Store new session ID. Args: `<id>` [state_file] |
| `scripts/hydrate.py` | Load recent daily notes + tiered memory + MEMORY.md into a summary. Args: `--days`, `--memory-limit`, `--workspace` |

State file default: `~/clawd/memory/heartbeat-state.json` (key: `lastSessionId`).
Override via `GUARD_STATE_FILE` env var or script argument.

## Known OpenClaw Bugs (cannot fix from agent side)

1. `heartbeat.session` config only supports `"main"` — no isolated option
2. `HEARTBEAT_OK` stripped → empty string forwarded to messaging → sendMessage error
3. `compaction.mode: "safeguard"` default too conservative for long-running agents
