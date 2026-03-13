# Memory Setup — Notes

## Initial Setup

Create directory structure on first use:

```bash
mkdir -p ~/notes/{meetings,decisions,projects,journal}
touch ~/notes/index.md
touch ~/notes/actions.md
```

---

## index.md Template

Copy to `~/notes/index.md`:

```markdown
# Notes Index

**Last updated:** YYYY-MM-DD

##  Structure

```
~/notes/
├── index.md        # This file
├── actions.md      # Active action items
├── meetings/       # Meeting notes
├── decisions/      # Decision log
├── projects/       # Project updates
└── journal/        # Daily notes
```

##  Tags Index

| Tag | Count | Recent |
|-----|-------|--------|
| #product | 5 | [[2026-02-19_roadmap]] |
| #engineering | 3 | [[2026-02-18_sprint]] |
| #1on1 | 8 | [[2026-02-17_alice-1on1]] |

##  People Index

| Person | Notes | Last |
|--------|-------|------|
| @alice | 12 | [[2026-02-19_product-sync]] |
| @bob | 8 | [[2026-02-15_design-review]] |

##  Recent Notes

### This Week
- [[2026-02-19_product-sync]] — meeting, #product
- [[2026-02-18_sprint-planning]] — meeting, #engineering
- [[2026-02-17_alice-1on1]] — 1on1, @alice

### Last Week
- [[2026-02-12_quarterly-review]] — meeting, #leadership
- [[2026-02-10_decision-pricing]] — decision, #product

##  Quick Search

Common queries:
- Meetings with @alice: `type:meeting attendees:alice`
- Product decisions: `type:decision tags:product`
- This month's journals: `type:journal date:2026-02`

---
*Update this index when adding notes with new tags or people.*
```

---

## actions.md Template

Copy to `~/notes/actions.md`:

```markdown
# Action Items Tracker

**Last updated:** YYYY-MM-DD HH:MM

##  Overdue

| # | Action | Owner | Due | Source | Days Late |
|---|--------|-------|-----|--------|-----------|
| — | *None* | — | — | — | — |

##  Due This Week

| # | Action | Owner | Due | Source |
|---|--------|-------|-----|--------|
| — | *None* | — | — | — |

##  Upcoming

| # | Action | Owner | Due | Source |
|---|--------|-------|-----|--------|
| — | *None* | — | — | — |

##  Recently Completed

| # | Action | Owner | Completed | Source |
|---|--------|-------|-----------|--------|
| — | *None* | — | — | — |

---

##  Stats

- **Total open:** 0
- **Overdue:** 0
- **Completion rate (7d):** —%

---
*Synced from all notes. Run "update actions" to refresh.*
```

---

## Sample Meeting Note

Example file `~/notes/meetings/2026-02-19_product-sync.md`:

```markdown
---
date: 2026-02-19
type: meeting
title: Product Sync
tags: [product, roadmap]
attendees: [alice, bob, carol]
duration: 30 min
---

# Meeting: Product Sync — 2026-02-19

**Time:** 10:00 - 10:30 | **Duration:** 30 min
**Facilitator:** Alice
**Attendees:** Alice, Bob, Carol

##  Meeting Goal
Align on Q1 priorities and blockers.

##  Key Discussion Points
- Feature X is behind schedule
- Customer feedback on Y is positive
- Need decision on Z approach

##  Decisions Made
- [DECISION] **Feature X scope:** Cut advanced mode for v1 — *Owner:* @alice | *Effective:* 2026-02-19
- [DECISION] **Launch date:** Move to Feb 28 — *Owner:* @bob | *Effective:* 2026-02-19

##  Action Items
| # | Task | Owner | Due | Status |
|---|------|-------|-----|--------|
| 1 | Update roadmap doc | @alice | 2026-02-20 |  |
| 2 | Notify stakeholders of date change | @bob | 2026-02-20 |  |
| 3 | Draft launch comms | @carol | 2026-02-25 |  |

##  Open Questions
- How to handle existing beta users? — *Needs input from:* @support

##  Meeting Effectiveness: 8/10
 Clear agenda beforehand
 Started/ended on time
 Decisions were made
 Actions have owners + deadlines
 Could NOT have been an email
```

---

## Sample Decision Entry

Example file `~/notes/decisions/2026-02-19_pricing-model.md`:

```markdown
---
date: 2026-02-19
type: decision
title: Pricing Model for v2
tags: [product, pricing, strategy]
status: active
---

# [DECISION] Pricing Model for v2 — 2026-02-19

## Context
Current flat pricing doesn't capture value from high-usage customers. Need to decide on v2 pricing before March launch.

## Options Considered

### Option A: Usage-Based
-  Aligns cost with value
-  Lower barrier to entry
-  Unpredictable revenue
-  Complex to communicate

### Option B: Tiered Plans
-  Predictable revenue
-  Easy to understand
-  May leave money on table
-  Upgrade friction

### Option C: Hybrid (Base + Usage)
-  Predictable base + upside
-  Fair for all segments
-  More complex billing
-  Harder to forecast

## Decision
**Chosen:** Option C — Hybrid model

## Rationale
Combines predictability of tiers with fairness of usage. Competitors moving this direction. Customer interviews showed preference for "pay for what you use" with a floor.

## Implementation
- **Owner:** @finance
- **Effective Date:** 2026-03-01
- **Review Date:** 2026-06-01

## Dependencies
- Requires: billing system upgrade [[2026-02-10_billing-update]]
- Blocks: launch communications

## Reversal
- [REVERSES] [[2025-06-15_flat-pricing]] — Original flat pricing decision
```
