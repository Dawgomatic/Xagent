---
name: "Skill Finder"
description: "Find the right skill for any need. Search, evaluate, and compare skills intelligently."
version: "1.0.4"
changelog: "Add Security Note about npx for VirusTotal compliance"
---

## Find the Right Skill

Search by need, not just name. Evaluate quality before recommending.

**References:**
- `search.md` — search strategies and commands
- `evaluate.md` — quality evaluation criteria
- `criteria.md` — when to update preference memory

**Related skills:**
- `skill-manager` — manages installed, suggests proactively
- `skill-builder` — creates new skills

---

## Scope

This skill ONLY:
- Searches ClawHub via `npx clawhub search` command
- Evaluates skills from search results
- Stores user preferences in `~/skill-finder/memory.md`

This skill NEVER:
- Reads files outside `~/skill-finder/`
- Observes or infers preferences from user behavior
- Installs skills automatically

---

## Security Note

This skill uses `npx clawhub search` which queries the ClawHub registry. This is a read-only operation that does not download or execute skill code. Skill installation requires separate user consent.

---

## When to Use

User explicitly asks to find a skill:
- "Is there a skill for X?"
- "Find me something that does Y"
- "What skills exist for Z?"

## Workflow

1. **Search** — `npx clawhub search "query"`
2. **Evaluate** — Apply criteria from `evaluate.md`
3. **Compare** — If multiple match, rank by fit
4. **Recommend** — Present top 1-3 with reasoning

---

## Data Storage

Preferences stored in `~/skill-finder/memory.md`.

**First use:** Create folder with `mkdir -p ~/skill-finder`

**What is stored (ONLY from explicit user statements):**
- Preferences user explicitly stated ("I prefer X")
- Skills user said they liked with reason
- Skills user declined with stated reason

**What is NEVER stored:**
- Inferred preferences from behavior
- Installation history without user comment
- Any data from outside this skill's scope

**Format:**
```markdown
## Preferences
- "value" — user's exact words

## Liked
- slug — "reason user gave"

## Passed
- slug — "reason user gave"
```

**Limit:** ≤50 lines. Archive old entries when exceeded.
