# Session State Tracker

A system for persistent session state in OpenClaw. Survive compaction and restarts.

## Quick Start

1. **Install the skill:**
   ```bash
   clawhub install qsmtco/session-state-tracker
   ```
   Or copy `skills/session-state-tracker/` into your workspace.

2. **Create initial state** (if none exists):
   ```bash
   session-state set project "my-project"
   session-state set task "Describe current task"
   session-state set status "active"
   session-state set next_steps '["Step 1","Step 2"]'
   ```

3. **Optional: Enable session transcript indexing** for automatic state refresh and discovery.
   - See "Optional Configuration" below. **Privacy note:** this indexes past conversations into the memory vector store.

4. **Optional: Customize pre-compaction flush** to automatically verify state before compaction.
   - See "Optional Configuration" below.

5. **Add maintenance rules** to your `AGENTS.md` (see full rules in SKILL.md).

## CLI Commands

```bash
session-state show              # Display current state (including Context)
session-state set <key> <value> # Update a field (e.g., "set task 'New task'")
session-state refresh           # Rediscover state from session transcripts (requires indexing)
session-state clear             # Reset to empty state
```

## How It Works

The skill provides three tools (`session_state_read`, `session_state_write`, `session_state_discover`) and a CLI. You control when to call them via `AGENTS.md` instructions or manual intervention.

For full automation (state refresh before compaction, re-anchor after), you should:
- Enable session transcript indexing (optional but recommended)
- Optionally customize the memory flush prompt
- Add the "Session State Maintenance" rules to `AGENTS.md`

See [SKILL.md](SKILL.md) for complete documentation, including security considerations and troubleshooting.

## Files

- `SESSION_STATE.md` – persistent state file (workspace root)
- `AGENTS.md` – agent behavior rules (you add the maintenance section)
- `memory/YYYY-MM-DD.md` – daily notes (complements state)

## Support

Issues: https://github.com/qsmtco/qrusher/issues
