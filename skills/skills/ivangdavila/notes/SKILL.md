---
name: Notes
slug: notes
version: 1.0.1
homepage: https://clawic.com/skills/notes
description: Capture meetings, decisions, and ideas with structured formats, action tracking, and searchable archives.
changelog: Added 7 note formats, action item tracking, memory storage with index
metadata: {"clawdbot":{"emoji":"","requires":{"bins":[]},"os":["linux","darwin","win32"]}}
---

## When to Use

User needs to capture any type of notes: meetings, brainstorms, decisions, daily journals, or project logs. Agent handles formatting, action item extraction, deadline tracking, and retrieval.

## Architecture

Notes live in `~/notes/`. See `memory-template.md` for setup.

```
~/notes/
├── index.md           # Search index with tags
├── meetings/          # Meeting notes by date
├── decisions/         # Decision log
├── projects/          # Project-specific notes
├── journal/           # Daily notes
└── actions.md         # Active action items tracker
```

## Quick Reference

| Topic | File |
|-------|------|
| All note formats | `formats.md` |
| Action item system | `tracking.md` |
| Memory setup | `memory-template.md` |

## Core Rules

### 1. Always Use Structured Format
Every note type has a specific format. See `formats.md` for templates.

| Note Type | Trigger | Key Elements |
|-----------|---------|--------------|
| Meeting | "meeting notes", "call with" | Attendees, decisions, actions |
| Decision | "we decided", "decision:" | Context, options, rationale |
| Brainstorm | "ideas for", "brainstorm" | Raw ideas, clusters, next steps |
| Journal | "daily note", "today I" | Date, highlights, blockers |
| Project | "project update", "status" | Progress, blockers, next |

### 2. Extract Action Items Aggressively
If someone says "I'll do X" or "we need to Y" — that's an action item.

**Every action item MUST have:**
- [ ] **Owner** — Who is responsible (@name)
- [ ] **Task** — Specific, actionable description
- [ ] **Deadline** — Explicit date (not "soon" or "ASAP")

**If missing deadline, suggest one:**
```
 No deadline set for: "Review the proposal"
   Suggested: 2026-02-21 (2 days from now)
   Confirm or specify: ___
```

### 3. One Response, Complete Output
Never split notes into multiple messages. Always include:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 [NOTE TYPE]: [Title] — [YYYY-MM-DD]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Formatted content per type]

 ACTION ITEMS ([X] total)
1. [ ] @Owner: Task — Due: YYYY-MM-DD
2. [ ] @Owner: Task — Due: YYYY-MM-DD

 Saved: notes/[folder]/YYYY-MM-DD_[topic].md
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 4. Filename Convention
Always: **YYYY-MM-DD_topic-slug.md** (date first, then topic)

Examples:
-  2026-02-19_product-review (correct: date first)
-  product-review-notes (wrong: no date)
-  notes-2026-02-19 (wrong: date not first)

### 5. Tag Everything for Retrieval
Every note gets tags in the header:

```markdown
---
date: 2026-02-19
type: meeting
tags: [product, roadmap, q1-planning]
attendees: [alice, bob, carol]
---
```

Update `~/notes/index.md` with each new note.

### 6. Track Actions Centrally
Maintain **~/notes/actions.md** as single source of truth. See `tracking.md` for system.

| Status | Meaning |
|--------|---------|
|  OVERDUE | Past deadline |
|  DUE SOON | Within 3 days |
|  ON TRACK | Future deadline |
|  DONE | Completed |

### 7. Link Related Notes
When a note references previous discussions:
- Never say "as discussed" without link
- Format: `See [[2026-02-15_kickoff]] for context`
- Search existing notes before creating duplicates

### 8. Decision Documentation
Decisions get special treatment:

```markdown
## [DECISION] Title — YYYY-MM-DD

**Context:** Why this decision was needed
**Options Considered:**
1. Option A — pros/cons
2. Option B — pros/cons
**Decision:** What was chosen
**Rationale:** Why this option
**Owner:** Who made it
**Effective:** When it takes effect
**Reverses:** [[previous-decision]] (if applicable)
```

### 9. Meeting Effectiveness Score
End every meeting note with:

```
 Meeting Effectiveness: [X/10]
   □ Clear agenda beforehand
   □ Started/ended on time  
   □ Decisions were made
   □ Actions have owners + deadlines
   □ Could NOT have been an email
```

### 10. Quick Capture Mode
For rapid input, accept minimal format:

User: "Note: call with Sarah, she wants the report by Friday, I'll send draft tomorrow"

Agent extracts:
- Meeting note: Call with Sarah
- Action: Send draft report (@me, due: tomorrow)
- Action: Final report (Sarah expects by Friday)

## Common Traps

- **Vague deadlines** → "ASAP", "soon", "next week" are not deadlines. Force explicit dates.
- **Missing owners** → "We should do X" needs "@who will do X"
- **Orphaned actions** → Actions not tracked centrally get forgotten. Always sync.
- **Duplicate notes** → Search before creating. Link don't duplicate.
- **No retrieval tags** → A note without tags is a note you'll never find.

## Related Skills
Install with `clawhub install <slug>` if user confirms:
- `meetings` — meeting facilitation and agendas
- `todo` — task management system
- `documentation` — technical docs
- `journal` — daily journaling practice
- `decisions` — decision frameworks

## Feedback

- If useful: `clawhub star notes`
- Stay updated: `clawhub sync`
