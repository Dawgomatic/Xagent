# ContextKeeper Implementation Guide

## Architecture

ContextKeeper is **conceptual** — it works through:

1. **Memory files** (already exists via SPIRIT/OpenClaw)
2. **Structured summaries** (written by agent)
3. **Trigger phrases** (detected in user input)

## No Binary Required

Unlike SPIRIT, ContextKeeper is **skill-only** — it uses existing infrastructure:

- `memory/YYYY-MM-DD.md` — daily logs
- `PROJECTS.md` — project registry
- Session history — via OpenClaw

## Phase 1: Manual (Current)

Agent writes checkpoint manually:

```markdown
## ContextKeeper Checkpoint: P002
**Time:** 2026-02-18 19:30 UTC
**Summary:** Working on BotCall PWA deployment
**Active:** `pwa/app.js`, `cmd/bot-cli/main.go`
**Decisions:** Use localtunnel for testing
**Blockers:** Discovery server not externally accessible
**Next:** Fix localtunnel discovery exposure
```

## Phase 2: Semi-Auto

Cron or trigger writes:

```bash
# On session end
echo "## ContextKeeper Checkpoint" >> memory/$(date +%Y-%m-%d).md
```

## Phase 3: Full Auto (Future)

Hook into OpenClaw:

- Session start → load checkpoint
- Message N → auto-summarize
- Session end → save checkpoint

## Integration Points

### With SPIRIT

```yaml
# SPIRIT syncs ContextKeeper state
tracked_files:
  - .memory/contextkeeper/
  - PROJECTS.md
```

### With HEARTBEAT.md

```yaml
# Add ContextKeeper check
- Read last checkpoint
- If stale >1h, create new summary
- Show dashboard if user asks
```

## Query Patterns

| Query | How ContextKeeper Responds |
|-------|------------------------------|
| "What were we doing?" | Read `current-state.json` |
| "Continue P002" | Load P002 timeline + last checkpoint |
| "Why?" | Search decisions for current project |
| "When did X?" | Search checkpoint summaries |
| "Finish it" | Check projects with status=in_progress |

## File Organization

```
.memory/
├── YYYY-MM-DD.md           # Daily log (existing)
├── contextkeeper/
│   ├── projects/
│   │   ├── P002-botcall/
│   │   │   ├── checkpoints/
│   │   │   ├── decisions.md
│   │   │   └── timeline.md
│   │   └── P003-spirit/
│   └── intents.json        # "it" → P002 mappings
└── PROJECTS.md             # Registry (existing)
```

## Decision Log Format

```markdown
## P002 Decisions

| Date | Decision | Rationale | Reversible |
|------|----------|-----------|------------|
| 2026-02-18 | Use localtunnel over VPS direct | Faster testing, no firewall | Yes |
| 2026-02-18 | GitHub Pages for PWA | Free, auto-deploy | Yes |
```

## Blocker Tracking

```markdown
## P002 Blockers

| Date | Blocker | Status | Resolution |
|------|---------|--------|------------|
| 2026-02-18 | Discovery not externally reachable | Active | Need localtunnel |
```

## Trigger Detection

```javascript
const TRIGGERS = {
  'continue_from_yesterday': /continue.*yesterday|resume.*yesterday|pick up/i,
  'what_were_doing': /what.*we.*doing|where.*left.*off|remind me/i,
  'ambiguous_thing': /(finish|check|look at|done with)\s+(it|that|this)/i,
  'checkpoint': /checkpoint|save state|mark this/i,
  'status': /status|dashboard|whats.*going on|update/i
};
```

## Sample Dashboard

```
 ContextKeeper
═══════════════

Active: P002 (BotCall) — 2h ago
├─ Last: Localtunnel fix in progress
├─ Files: pwa/app.js, cmd/bot-cli/main.go  
├─ Blocker: Discovery external access
└─ Next: Test end-to-end

On Deck: P003 (SPIRIT) — completed
├─ Status: v0.1.6 published
└─ Action: Update PR #391

 Suggest: Continue P002 or switch to open PR?
```

## Implementation Priority

1. **Manual checkpoints** — Agent writes to PROJECTS.md ✓
2. **Git-aware helper script** — Auto-detect repo, branch, recent files (NEW)
3. **Trigger detection** — Parse user intent
4. **Auto-summaries** — Brief summaries every N messages
5. **Dashboard** — Status command
6. **Full auto** — Session hooks

## New: Git-Aware Checkpoint Script

**Location:** `scripts/ck-checkpoint.sh`

**Features:**
- Auto-detects git repo, branch, modified file count
- Finds recently changed files (last 24h)
- Links to previous checkpoint
- Creates JSON checkpoint + updates `current-state.json`

**Usage:**
```bash
# Create checkpoint with context
./scripts/ck-checkpoint.sh create "Fixed localtunnel routing"

# Quick status (git + files + last checkpoint)
./scripts/ck-checkpoint.sh status

# View last checkpoint
./scripts/ck-checkpoint.sh last
```

**Output format:**
```json
{
    "timestamp": "2026-02-18T20:08:00Z",
    "checkpoint_id": "abc123de",
    "type": "auto",
    "git": {
        "repo": "botcall",
        "branch": "dev/pwa",
        "modified_count": "3"
    },
    "files_changed": ["pwa/app.js", "cmd/bot-cli/main.go"],
    "context": {
        "previous": "...",
        "message": "Fixed localtunnel routing"
    }
}
```

**Next step:** Hook this into session end for Phase 2.5 auto-checkpointing.

## No Code Required

This skill is **procedural** — it guides the agent (me) to:
- Write structured summaries
- Check PROJECTS.md before responding
- Ask clarifying questions on ambiguous references

The "implementation" is behavioral, not a binary.
