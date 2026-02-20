# Session State Tracker Skill

**Version:** 1.0.0  
**Author:** qsmtco  
**License:** MIT

---

## Overview

The Session State Tracker solves the problem of context loss during OpenClaw session compaction and restarts. It provides a persistent `SESSION_STATE.md` file that keeps your current project, task, status, and next steps readily available.

### Key Features

- **Manual or automatic state maintenance** – Agent can update `SESSION_STATE.md` after significant progress
- **Optional integration** with pre-compaction flush and session transcript indexing for seamless operation
- **CLI tools** for manual inspection and updates
- **No external network access** – operates entirely within workspace

---

## Problem Statement

OpenClaw automatically compacts long-running sessions to stay within model context windows. Compaction summarizes older messages and removes them from active context. This leads to loss of conversational continuity:

- The agent forgets the current task and next steps
- Users must repeatedly re-explain what they're working on
- Long-running projects become fragmented

Existing mechanisms (`MEMORY.md`, daily logs) are for curated long-term memory, not for *current working context*. We need a lightweight, always-present anchor that survives compaction.

---

## Solution Architecture

The solution is built around a simple file (`SESSION_STATE.md`) and optional OpenClaw features:

1. **`SESSION_STATE.md`** — A small Markdown file with YAML frontmatter storing current state
2. **Optional: Pre-compaction memory flush customization** – Instructs the agent to verify/update the state file before compaction occurs
3. **Optional: Session transcript indexing** (`memorySearch.experimental.sessionMemory = true`) – Enables `memory_search` to retrieve verbatim details from compacted messages

The skill itself provides **tools** to read, write, and discover state. How these tools are used (automatic vs manual) is up to the agent's instructions (e.g., `AGENTS.md`) and the user's configuration choices.

---

## File Format

`SESSION_STATE.md` lives in the workspace root:

```markdown
---
project: "session-state-tracker"
task: "Implement skill with read/write/discover tools"
status: "active"          # active | blocked | done
last_action: "Designed revised architecture"
next_steps:
  - "Create skill skeleton"
  - "Implement state.js"
  - "Write CLI"
updated: "2026-02-14T23:20:00Z"
---

## Context
- Brief freeform notes, constraints, links, etc.
```

All frontmatter fields are plain strings, arrays, booleans, or numbers. The `Context` section is for human notes and is preserved but not used programmatically.

---

## Implementation Walkthrough

This section documents how to set up the Session State Tracker system in your OpenClaw installation. **The skill itself is self-contained; the following steps are optional configuration that enhances automation.**

### 1. Install the Skill

Copy the `skills/session-state-tracker/` directory into your workspace, or install via ClawHub:

```bash
clawhub install qsmtco/session-state-tracker
```

The skill provides three tools:
- `session_state_read` – reads the state file
- `session_state_write` – updates fields (auto-timestamps)
- `session_state_discover` – uses `memory_search` to rebuild state from session transcripts

And a CLI: `session-state` (commands: `show`, `set`, `refresh`, `clear`).

**No gateway restart is required for the skill itself** (if `commands.nativeSkills = "auto"` it loads automatically; otherwise restart).

---

### 2. Optional: Enable Session Transcript Indexing

**Privacy impact:** This setting causes your conversation transcripts to be indexed into the memory vector store, making past messages searchable via `memory_search`. This increases the blast radius of your memory system. Only enable if you understand and accept this.

**Why it's useful:** The `session_state_discover` tool relies on `memory_search` to find recent task mentions. Without session indexing, discovery will return no results.

**How to enable:**

Patch your OpenClaw config (`~/.openclaw/openclaw.json` or `openclaw.json` in your profile):

```json5
{
  "agents": {
    "defaults": {
      "memorySearch": {
        "sources": ["memory", "sessions"],
        "experimental": { "sessionMemory": true },
        "sync": { "sessions": { "deltaBytes": 100000, "deltaMessages": 50 } }
      }
    }
  }
}
```

Then restart the gateway:

```bash
openclaw gateway restart
```

---

### 3. Optional: Customize Pre-Compaction Flush Prompt

**Why:** If you want the state file to be refreshed automatically before compaction, customize the `memoryFlush.prompt` to remind the agent to check `SESSION_STATE.md`.

**How:**

```json5
{
  "agents": {
    "defaults": {
      "compaction": {
        "memoryFlush": {
          "prompt": "Check SESSION_STATE.md. If your current task has changed, update the file. Then write any lasting notes to memory/YYYY-MM-DD.md. Reply with NO_REPLY if nothing to store."
        }
      }
    }
  }
}
```

Restart the gateway after applying.

---

### 4. Update `AGENTS.md` with Maintenance Rules

Add the following section to your workspace `AGENTS.md`. This is **not** part of the skill; it's agent behavior configuration.

```markdown
## Session State Maintenance

`SESSION_STATE.md` is your working memory across compaction and restarts.

**Format:**
```markdown
---
project: ""
task: ""
status: "active|blocked|done"
last_action: ""
next_steps: []
updated: "ISO timestamp"
---
## Context
- Notes, constraints, links
```

**Rules:**

1. **At session start** (after reading `SOUL.md`, `USER.md`, `memory/`):
   - If `SESSION_STATE.md` exists and `updated` is within the last 24 hours, read it and keep its contents in mind.
   - If missing or older than 24 hours, use `memory_search(sources=["sessions"], query="project|task|working on")` to discover current focus, then write a fresh state file.

2. **After significant progress or a change in focus**:
   - Call `session_state_write` (or write manually) to update `SESSION_STATE.md`.
   - Always update the `updated` timestamp.

3. **When you see a `compaction` entry in the conversation**:
   - Read `SESSION_STATE.md` to refresh your memory of the current task before responding.
   - If the file feels out of date, run discovery (`session_state_discover`) and update it.

4. **During the pre-compaction flush** (if you customized the prompt):
   - Verify `SESSION_STATE.md` matches the current focus; update if needed.
   - Then write any lasting notes to `memory/YYYY-MM-DD.md`.
   - Respond with `NO_REPLY`.

5. **Never delete** `SESSION_STATE.md`. If obsolete, set `status: "done"` and create a new file for new work.
```

---

### 5. Create Initial State

If `SESSION_STATE.md` doesn't exist, create it with your current project:

```bash
session-state set project "my-project"
session-state set task "Describe current task"
session-state set status "active"
session-state set last_action "Initialized state"
session-state set next_steps '["Step 1", "Step 2"]'
```

Or manually edit the file.

---

## How It Works

### Normal operation (with optional config)

1. You start a session. The agent reads `SESSION_STATE.md` if fresh; otherwise discovers from sessions.
2. You work on the task. The agent updates the state file after major steps (via `session_state_write` or manual edit).
3. Token usage rises. When near compaction threshold, if you customized the flush prompt, the agent receives a silent turn reminding it to check/update state.
4. Agent updates state if needed, writes daily notes, replies `NO_REPLY`.
5. Compaction executes: older messages summarized and removed from context.
6. Next user message arrives; the conversation includes a `compaction` entry.
7. By rule from `AGENTS.md`, the agent reads `SESSION_STATE.md` again to re‑anchor.
8. Conversation continues with the state file as the anchor.

### Without optional config (manual mode)

If you didn't enable session indexing or custom flush, you can still manually run `session-state refresh` when you suspect the state is stale, or edit the file directly. The agent should still follow the `AGENTS.md` rules to read the file at session start and after compaction (the rule about reading after a `compaction` entry works regardless of config; the agent just needs to have the habit to read the file).

---

## Tool Definitions

### `session_state_read`

Reads `SESSION_STATE.md` and returns **all frontmatter fields plus `body`** (the Context section).

**Input:** none  
**Output:**
```json
{
  "project": "session-state-tracker",
  "task": "Implement skill",
  "status": "active",
  "last_action": "Designed architecture",
  "next_steps": ["Create skeleton", "Implement state.js"],
  "updated": "2026-02-14T23:20:00Z",
  "body": "Notes about constraints, links, etc."
}
```

### `session_state_write`

Updates one or more fields in `SESSION_STATE.md`. Automatically sets `updated` to current ISO timestamp unless provided.

**Input:** partial object, e.g., `{ "task": "New task", "status": "active" }`  
**Output:** `{ "success": true, "fields": ["task","status"], "updated": "2026-02-14T23:25:00Z" }`

### `session_state_discover`

Uses `memory_search` (with `sources: ["sessions"]`) to synthesize a new state from recent conversation snippets. Automatically writes the discovered state to `SESSION_STATE.md`.

**Input:** optional `{ query, limit, minScore }`  
**Output:** state object (same shape as `read`) with a `_meta` field indicating snippet count and top source.

**Note:** Requires `memory_search` tool to be available and session transcript indexing to be enabled for meaningful results.

---

## CLI Commands

The skill installs a `session-state` binary:

```bash
# Show current state (including Context)
session-state show

# Update a single field (string; for arrays use JSON format)
session-state set task "Refine discovery algorithm"
session-state set next_steps '["Step A","Step B"]'

# Refresh state from session transcripts (calls discover)
session-state refresh

# Clear state (sets all fields empty)
session-state clear
```

**Note:** `session-state refresh` requires the `memory_search` tool to be available in the execution environment (when run via OpenClaw `exec`, it is injected; when run manually in a shell, it may fail unless you set up the environment).

---

## Dependencies & Requirements

- OpenClaw >= 2026.2.0 (for stable tool context and memory_search)
- Node.js 18+
- **Optional:** session transcript indexing (`memorySearch.experimental.sessionMemory = true`) for discovery
- **Optional:** custom `memoryFlush.prompt` for automatic pre-compaction refresh

Skill runtime dependencies:
- `js-yaml` (for robust YAML parsing)

---

## Security & Privacy Considerations

- **File scope:** The skill reads and writes only `SESSION_STATE.md` in the workspace root. It does not access files outside the workspace.
- **Network:** No network calls are made by the skill itself. `session_state_discover` uses the `memory_search` tool, which may use remote embedding APIs depending on your OpenClaw config. That is governed by your existing memory provider settings.
- **Privilege:** The skill runs with the agent's standard file I/O permissions. It does **not** request elevated `exec`, `node`, or sandbox escape.
- **Persistence:** State is persisted in a plain Markdown file, which you can back up via Git or other means.
- **Config changes:** The recommended configuration (session indexing, flush prompt) are **optional** and affect **all agents** on the gateway. Review the privacy implications before applying them:
  - Session indexing increases the amount of data stored in the memory vector database (full conversation snippets).
  - The flush prompt customization is harmless but does add a pre-compaction agent turn.
- **User control:** You can choose to use the skill in a purely manual mode without any config changes. Simply edit `SESSION_STATE.md` yourself and optionally use the CLI tools.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `session-state refresh` fails with "memory_search not available" | Session indexing not enabled or running outside OpenClaw exec | Enable `memorySearch.experimental.sessionMemory` in config, or run via `openclaw exec` |
| State file not read after compaction | `AGENTS.md` rule missing or outdated | Ensure the "Session State Maintenance" section is present and includes rule #3 |
| YAML parse errors | Malformed frontmatter (e.g., missing colon, bad indentation) | Use a YAML validator; keep values simple (scalars, JSON arrays) |
| `readState` returns null | File missing or empty | Create initial `SESSION_STATE.md` with at least the frontmatter fields |

---

## Design Rationale

- **Plain file** instead of database: human-readable, Git-friendly, no extra infrastructure.
- **Minimal dependencies**: only `js-yaml` for robust parsing.
- **Agent compliance** via `AGENTS.md` rules rather than forced injection: respects OpenClaw's architecture where the agent decides when to read/write.
- **Optional integration** with existing features: doesn't require changing gateway defaults; works even without session indexing (just less automation).

---

## Future Enhancements

- Validate state schema on write (ensure required fields present)
- Add `session_state validate` CLI command
- Support multiple state files per project (subdirectories)
- Heartbeat integration to flag stale states

---

**That's the complete skill.** All configuration steps are clearly marked as optional, with privacy warnings. The skill itself is low‑privilege and only operates on the state file.
