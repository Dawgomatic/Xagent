---
name: shell-security-ultimate
version: 1.0.2
description: "Classify shell commands by risk level (SAFE to CRITICAL) before your OpenClaw agent executes them. Color-coded output, transparency enforcement, audit logging. Prevent dangerous operations, credential exposure, or unauthorized network access. Enforcement scripts and integration patterns included."
homepage: https://github.com/globalcaos/clawdbot-moltbot-openclaw
repository: https://github.com/globalcaos/clawdbot-moltbot-openclaw
---

# Shell Security Ultimate

Security-first command execution for AI agents. Classify, audit, and control every shell command.

---

## The Problem

AI agents with shell access can:
- Run destructive commands (`rm -rf /`)
- Leak sensitive data (`cat ~/.ssh/id_rsa`)
- Modify system state without oversight
- Execute commands without explaining why

**This skill solves it** by enforcing security classification, transparency, and auditability for every command.

---

## Coded vs Prompted Behaviors

There are two ways to control agent behavior:

| Approach | Enforcement | Reliability | Example |
|----------|-------------|-------------|---------|
| **Prompted** | Instructions in MD files | ~80% | "Don't run dangerous commands" in SOUL.md |
| **Coded** | Actual code/hooks | ~100% | Plugin that blocks `rm -rf` before execution |

### Why This Matters

- **Prompted behaviors decay** — Agents can forget instructions during long sessions
- **Coded behaviors persist** — Code doesn't forget, can't be talked out of rules
- **Defense in depth** — Use both: prompts for guidance, code for enforcement

### Current State of This Skill

| Component | Type | Status |
|-----------|------|--------|
| Classification guide | Prompted |  In SKILL.md |
| Display script | Coded |  `scripts/cmd_display.py` |
| SOUL.md integration | Prompted |  Template provided |
| OpenClaw plugin hook | Coded |  Not yet — requires `before_tool_call` hook |
| Blocklist enforcement | Coded |  Planned — would reject commands matching patterns |

**Where we are:** Mixed approach. The display script provides structure, but true enforcement (blocking dangerous commands before execution) requires an OpenClaw plugin. The current implementation relies on the agent *choosing* to use the wrapper.

**Where we're going:** Full coded enforcement via plugin that intercepts `exec` tool calls and applies security policy before execution.

---

## Security Levels

| Level | Emoji | Risk | Examples |
|-------|-------|------|----------|
|  SAFE | None | `ls`, `cat`, `git status`, `pwd` |
|  LOW | Reversible | `touch`, `mkdir`, `git commit` |
|  MEDIUM | Moderate | `npm install`, `git push`, config edits |
|  HIGH | Significant | `sudo`, service restarts, global installs |
|  CRITICAL | Destructive | `rm -rf`, database drops, credential access |

---

## Usage

### Basic Format

```bash
python3 scripts/cmd_display.py <level> "<command>" "<purpose>" "$(<command>)"
```

### Examples

** SAFE — Read-only:**
```bash
python3 scripts/cmd_display.py safe "git status" "Check repo state" "$(git status --short)"
```

** LOW — File changes:**
```bash
python3 scripts/cmd_display.py low "touch notes.md" "Create file" "$(touch notes.md && echo '✓')"
```

** MEDIUM — Dependencies:**
```bash
python3 scripts/cmd_display.py medium "npm install axios" "Add HTTP client" "$(npm install axios 2>&1 | tail -1)"
```

** HIGH — Show only, don't execute:**
```bash
python3 scripts/cmd_display.py high "sudo systemctl restart nginx" "Restart server" " Manual execution required"
```

** CRITICAL — Never auto-execute:**
```bash
python3 scripts/cmd_display.py critical "rm -rf node_modules" "Clean deps" " Blocked - requires human confirmation"
```

---

## Output Format

```
 SAFE ✓ git status --short │ Check repo state
   2 modified, 1 untracked

 HIGH  sudo systemctl restart nginx │ Restart server
    Manual execution required
```

---

## Agent Integration

Add to your `SOUL.md` or `AGENTS.md`:

```markdown
## Command Execution Protocol

1. **Classify** every command before running (SAFE/LOW/MEDIUM/HIGH/CRITICAL)
2. **Wrap** with: `python3 <skill>/scripts/cmd_display.py <level> "<cmd>" "<why>"`
3. **HIGH commands** — Show for manual execution, do not run
4. **CRITICAL commands** — NEVER execute, always ask human first
5. **Summarize** verbose output to one line
```

---

## Classification Quick Reference

** SAFE (auto-execute):**
`ls`, `cat`, `head`, `grep`, `find`, `git status`, `git log`, `pwd`, `whoami`, `date`

** LOW (execute, log):**
`touch`, `mkdir`, `cp`, `mv` (in project), `git add`, `git commit`

** MEDIUM (execute with caution):**
`npm/pip install`, `git push/pull`, config file edits

** HIGH (show, ask first):**
`sudo *`, service commands, global installs, network config

** CRITICAL (never auto-execute):**
`rm -rf`, `DROP DATABASE`, credential files, system directories

---

## Roadmap

- [x] Classification guidelines
- [x] Display wrapper script
- [x] Agent integration template
- [ ] OpenClaw plugin for `before_tool_call` enforcement
- [ ] Configurable blocklist patterns
- [ ] Audit log persistence

---

## Philosophy

> *"If you can enforce it with code, don't rely on documentation."*

Prompted behaviors are suggestions. Coded behaviors are laws. This skill provides both — use the prompts now, upgrade to coded enforcement when the plugin is ready.

---

## Credits

Created by **Oscar Serra** with the help of **Claude** (Anthropic).

*Security is not optional. Every command an agent runs should be classified, justified, and auditable.*
