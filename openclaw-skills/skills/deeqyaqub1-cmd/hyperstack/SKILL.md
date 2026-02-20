---
name: hyperstack
description: "Typed graph memory for multi-agent coordination. Replace GOALS.md + DECISIONS.md with queryable cards and relations. Ask 'what blocks task X?' and get exact answers. Ask 'what depends on this card?' and trace full impact. Ask 'what's related to this?' and get scored recommendations. Ask anything in plain English — smart search picks the right mode automatically. Pin cards to protect them. Prune stale memory. Commit agent outcomes as decisions. Zero LLM cost."
user-invocable: true
homepage: https://cascadeai.dev/hyperstack
metadata:
  openclaw:
    emoji: "🃏"
    requires:
      env:
        - HYPERSTACK_API_KEY
        - HYPERSTACK_WORKSPACE
    primaryEnv: HYPERSTACK_API_KEY
---

# HyperStack — Typed Graph Memory for Multi-Agent Coordination

## What this does

Replaces markdown-file coordination (GOALS.md, DECISIONS.md, WORKING.md) with a typed knowledge graph that any agent can query. Five graph traversal modes — forward, impact, recommend, smart search, and time-travel — cover every relational question an agent needs to ask. Memory pruning removes stale cards. TTL scratchpad handles working memory. Feedback commit turns successful outcomes into permanent decisions.

**Before** (current multi-agent):
```
# DECISIONS.md (append-only)
- 2026-02-15: Use Clerk for auth (coder-agent)
- 2026-02-16: Migration blocks production deploy (ops-agent)
```
"What breaks if we change auth?" → grep through every file → manual, fragile, slow

**After** (HyperStack):
```
"What breaks if we change auth?"    → hs_impact use-clerk → [auth-api, deploy-prod, billing-v2]
"What blocks deploy?"               → hs_blockers deploy-prod → [migration-23]
"What's related to stripe?"         → hs_recommend use-stripe → scored list of related cards
"What depends on the API?"          → hs_smart_search "what depends on the API?" → auto-routed
"Clean up old memory"               → hs_prune dry=true → preview, then prune
"Agent completed a task"            → hs_commit → outcome stored as decided card
```

Typed relations. Exact answers. Zero LLM cost.

---

## Tools

### hs_search
Search the shared knowledge graph. Hybrid semantic + keyword matching.
```
hs_search({ query: "authentication setup" })
```

### hs_smart_search ✨ NEW in v1.0.18
Agentic RAG — automatically routes to the best retrieval mode (search, graph traversal, or impact analysis) based on the query. Use this when you're unsure which mode to use, or for natural language queries. Returns results plus the mode that was used.
```
hs_smart_search({ query: "what depends on the auth system?" })
→ routed to: impact
→ [auth-api] API Service — via: triggers
→ [billing-v2] Billing v2 — via: depends-on
→ [deploy-prod] Production Deploy — via: blocks

hs_smart_search({ query: "authentication setup" })
→ routed to: search
→ Found 3 memories: ...

# Optional: hint a starting card slug
hs_smart_search({ query: "what breaks if this changes?", slug: "use-clerk" })
```

When to use:
- When you're not sure whether to use hs_search, hs_graph, or hs_impact
- Natural language queries about dependencies, relationships, or context
- Multi-agent flows where the querying agent doesn't know the graph shape
- The 4-agent demo pattern: Agent 4 calls hs_smart_search and gets the full chain automatically

### hs_store
Store a card in the graph. Auto-tags with your agent ID. Supports pinning to protect from pruning. Supports TTL for temporary scratchpad cards.
```
hs_store({
  slug: "use-clerk",
  title: "Use Clerk for auth",
  body: "Better DX, lower cost, native Next.js support",
  type: "decision",
  links: "auth-api:triggers,alice:decided"
})

# Pin a card so it's never pruned
hs_store({
  slug: "core-auth-decision",
  title: "Core Auth Decision",
  body: "...",
  type: "decision",
  pinned: true
})

# Scratchpad card with TTL — auto-deletes after expiry
hs_store({
  slug: "task-scratch-001",
  title: "Working memory for current task",
  body: "Intermediate reasoning step...",
  type: "scratchpad",
  ttl: "2026-02-21T10:00:00Z"
})
```

Valid cardTypes: `general`, `person`, `project`, `decision`, `preference`, `workflow`, `event`, `account`, `signal`, `scratchpad`

### hs_decide
Record a decision with full provenance — who decided, what it affects, what it blocks.
```
hs_decide({
  slug: "use-clerk",
  title: "Use Clerk for auth",
  rationale: "Better DX, lower cost vs Auth0",
  affects: "auth-api,user-service",
  blocks: ""
})
```

### hs_commit ✨ NEW in v1.0.19
Commit a successful agent outcome as a permanent decision card, auto-linked to the source task via `decided` relation. Teaches agents to learn from what worked. Full version history, embeddings, and webhooks fire on commit.
```
hs_commit({
  taskSlug: "task-auth-refactor",
  outcome: "Successfully migrated all auth middleware to Clerk. Zero regressions.",
  title: "Auth Refactor Completed",
  keywords: ["clerk", "auth", "completed"]
})
→ {
    committed: true,
    slug: "commit-task-auth-refactor-1234567890",
    linkedTo: "task-auth-refactor",
    relation: "decided",
    cardType: "decision"
  }
```

When to use:
- After a task completes successfully — commit the outcome so future agents learn from it
- To build procedural memory — what worked, not just what was planned
- Before closing a session — commit any decisions made so they persist

### hs_prune ✨ NEW in v1.0.19
Remove stale cards that haven't been updated in N days and are not referenced by any other card. Pinned cards and TTL scratchpad cards are never pruned. Always run with `dry=true` first to preview.
```
# Preview what would be pruned — safe, no deletions
hs_prune({ days: 30, dry: true })
→ {
    dryRun: true,
    wouldPrune: 3,
    skipped: 2,
    cards: [{ slug: "old-task", title: "...", lastUpdated: "..." }],
    protected: [{ slug: "still-linked", title: "..." }]
  }

# Execute the prune
hs_prune({ days: 30 })
→ { pruned: 3, skipped: 2, message: "Pruned 3 cards..." }

# Aggressive cleanup
hs_prune({ days: 7, dry: true })
```

Safety guarantees:
- Cards linked by other cards are NEVER pruned (graph integrity preserved)
- Pinned cards (`pinned: true`) are NEVER pruned
- TTL scratchpad cards are handled by their own expiry — not touched by prune
- `dry=true` always safe — preview only, zero deletions

When to use:
- Periodic maintenance: remove abandoned tasks and outdated notes
- Before a major refactor: clean stale context that might confuse retrieval
- Always dry-run first, inspect the list, then execute

### hs_blockers
Check what blocks a task/card. Returns exact typed blockers, not fuzzy search results.
```
hs_blockers({ slug: "deploy-prod" })
→ "1 blocker: [migration-23] Auth migration to Clerk"
```

### hs_graph
Traverse the knowledge graph forward from a starting card. See connections, ownership, dependencies. Supports time-travel: pass a timestamp to reconstruct the graph at any point in time.
```
hs_graph({ from: "auth-api", depth: 2 })
→ nodes: [auth-api, use-clerk, migration-23, alice]
→ edges: [auth-api→triggers→use-clerk, migration-23→blocks→deploy-prod]

# Time-travel: see the graph at a specific moment
hs_graph({ from: "auth-api", depth: 2, at: "2026-02-15T03:00:00Z" })
```

### hs_impact ✨ NEW in v1.0.16
Reverse traversal — find everything that depends on or is affected by a given card.
```
hs_impact({ slug: "use-clerk" })
→ "Impact of [use-clerk]: 3 cards depend on this
   [auth-api] API Service (project) — via: triggers
   [billing-v2] Billing v2 (project) — via: depends-on
   [deploy-prod] Production Deploy (workflow) — via: blocks"

# Filter by relation type
hs_impact({ slug: "use-clerk", relation: "depends-on" })
```

### hs_recommend ✨ NEW in v1.0.16
Find cards most related to a given card by shared graph neighbourhood (co-citation scoring).
```
hs_recommend({ slug: "use-stripe" })
→ "Related cards for [use-stripe] — 4 found:
   [billing-v2] Billing v2 (project) — score: 4
   [use-clerk] Use Clerk for auth (decision) — score: 3"
```

### hs_my_cards
List all cards created by this agent.
```
hs_my_cards()
→ "3 cards by agent researcher: [finding-clerk-pricing] [finding-auth0-limits]"
```

### hs_ingest
Auto-extract cards from raw text. Zero LLM cost (regex-based).
```
hs_ingest({ text: "We're using Next.js 14 and PostgreSQL. Alice decided to use Clerk for auth." })
→ "✅ Created 3 cards from 78 chars"
```

### hs_inbox
Check for cards directed at this agent by other agents.
```
hs_inbox({})
→ "Inbox for cursor-mcp: 1 card(s)"
```

### hs_stats (Pro+)
Get token savings stats and memory usage for this workspace.
```
hs_stats()
→ "Cards: 24 | Tokens stored: 246 | Saving: 94% — $2.07/mo"
```

### hs_webhook (Team+)
Register a webhook for real-time agent notifications.
```
hs_webhook({ url: "https://your-server.com/webhook", events: "card.created,signal.received" })
```

### hs_agent_tokens (Team+)
Create, list, and revoke scoped per-agent tokens.
```
hs_agent_tokens({ action: "create", name: "Researcher Agent", agentId: "researcher", canRead: ["*"], canWrite: ["general", "signal"] })
```

---

## The Six Graph Modes

HyperStack's graph API covers every relational question an agent needs to ask:

| Mode | Tool | Question answered |
|------|------|-------------------|
| Smart | `hs_smart_search` | Ask anything — auto-routes to the right mode |
| Forward | `hs_graph` | What does this card connect to? |
| Impact | `hs_impact` | What depends on this card? What breaks if it changes? |
| Recommend | `hs_recommend` | What's topically related, even without direct links? |
| Time-travel | `hs_graph` with `at=` | What did the graph look like at a given moment? |
| Prune | `hs_prune` | What stale memory is safe to remove? |

---

## Memory Model

HyperStack now covers the full agent memory lifecycle:

| Memory type | Tool | Behaviour |
|-------------|------|-----------|
| Long-term facts | `hs_store` | Permanent, searchable, graph-linked |
| Working memory | `hs_store` with `ttl=` + `type=scratchpad` | Auto-deletes after TTL |
| Outcomes / learning | `hs_commit` | Commits what worked as a decided card |
| Stale cleanup | `hs_prune` | Removes unused cards, preserves graph integrity |
| Protected facts | `hs_store` with `pinned=true` | Never pruned, always kept |

---

## Multi-Agent Setup

Each agent gets its own ID. Cards are auto-tagged so you can see who created what.

Recommended roles:
- **coordinator**: Routes tasks, monitors blockers (`hs_blockers`, `hs_impact`, `hs_graph`, `hs_decide`)
- **researcher**: Investigates, stores findings (`hs_search`, `hs_recommend`, `hs_store`, `hs_ingest`)
- **builder**: Implements, records tech decisions (`hs_store`, `hs_decide`, `hs_commit`, `hs_blockers`)
- **memory-agent**: Maintains graph health (`hs_prune`, `hs_stats`, `hs_smart_search`)

---

## Setup

### Option A: MCP (Claude Desktop / Cursor / VS Code / Windsurf)
```json
{
  "mcpServers": {
    "hyperstack": {
      "command": "npx",
      "args": ["-y", "hyperstack-mcp"],
      "env": { "HYPERSTACK_API_KEY": "hs_your_key" }
    }
  }
}
```

### Option B: OpenClaw environment variable
1. Get free API key: https://cascadeai.dev/hyperstack
2. Set `HYPERSTACK_API_KEY=hs_your_key` in your OpenClaw env
3. Tools are available immediately

### Option C: Programmatic (Node.js adapter)
```js
import { createOpenClawAdapter } from "hyperstack-core/adapters/openclaw";
const adapter = createOpenClawAdapter({ agentId: "builder" });
await adapter.onSessionStart({ agentName: "Builder", agentRole: "Implementation" });
await adapter.onSessionEnd({ summary: "Completed auth migration" });
```

---

## When to use each tool

| Moment | Tool |
|--------|------|
| Start of session | `hs_search` + `hs_recommend` for context |
| Not sure which mode | `hs_smart_search` — auto-routes |
| New project / onboarding | `hs_ingest` to auto-populate from existing docs |
| Decision made | `hs_decide` with rationale and links |
| Task completed | `hs_commit` — commit outcome as decided card |
| Task blocked | `hs_store` with `blocks` relation |
| Before starting work | `hs_blockers` to check dependencies |
| Before changing a card | `hs_impact` to check blast radius |
| Discovery | `hs_recommend` to find related context |
| Working memory | `hs_store` with `ttl=` + `type=scratchpad` |
| Periodic cleanup | `hs_prune dry=true` → inspect → `hs_prune` |
| Debug a bad decision | `hs_graph` with `at` timestamp |
| Cross-agent signal | `hs_store` with `targetAgent` → other agent checks `hs_inbox` |
| Check efficiency | `hs_stats` to see token savings |
| Lock down agents | `hs_agent_tokens` per agent |

---

## Data safety

NEVER store passwords, API keys, tokens, PII, or credentials. Cards should be safe in a data breach. Always confirm with user before storing.

---

## Pricing

Free: 10 cards, keyword search, REST API + MCP
Pro ($29/mo): 100 cards, graph traversal (all modes), semantic search, analytics
Team ($59/mo): 500 cards, 5 team API keys, webhooks, scoped agent tokens
Business ($149/mo): 2,000 cards, 20 members, SSO

---

## Changelog

### v1.0.19 (Feb 20, 2026)

#### ✨ `hs_prune` — Memory Pruning with Dry-Run
Removes stale cards not updated in N days that are not referenced by any other card. Safety-first: linked cards, pinned cards, and TTL scratchpad cards are never touched. Always use `dry=true` first to preview what would be pruned before executing. Returns full list of what was pruned and what was protected with reasons.

#### ✨ `hs_commit` — Feedback-Driven Memory (Agent Learning)
Commits a successful task outcome as a permanent `decision` card auto-linked to the source task via `decided` relation. Builds procedural memory — agents accumulate what worked, not just what was planned. Full version history, embeddings, and webhooks fire on every commit.

#### ✨ `pinned` field on cards
Set `pinned: true` on any card to protect it from pruning permanently. Use for core architecture decisions, critical constraints, or any card that must never be deleted regardless of age. Exposed in all GET responses.

#### ✨ `scratchpad` cardType
New `scratchpad` cardType for temporary working memory. Combine with `ttl=` for auto-expiring cards. TTL scratchpad cards are excluded from pruning — they manage their own lifecycle.

#### ✨ TTL lazy expiry
Cards with a `ttl` datetime auto-delete on next GET request after expiry. No cron job needed. Expiry count returned in list responses as `expired: N`.

### v1.0.18 (Feb 20, 2026)
- Added `hs_smart_search` — Agentic RAG routing (mode=auto)
- Synced with MCP v1.6.0, hyperstack-py v1.1.0, hyperstack-langgraph v1.3.0

### v1.0.17 (Feb 19, 2026)
- Metadata fix: declared HYPERSTACK_WORKSPACE env var requirement

### v1.0.16 (Feb 19, 2026)
- Added `hs_impact` — reverse graph traversal
- Added `hs_recommend` — co-citation scoring

### v1.0.15 (Feb 17, 2026)
- Added `hs_stats` — token savings & memory health (Pro+)
- Added `hs_agent_tokens` — scoped per-agent permissions (Team+)

### v1.0.14 (Feb 17, 2026)
- Added `hs_ingest`, `hs_inbox`, `hs_webhook` / `hs_webhooks`

### v1.0.13 and earlier — Core Foundation
- `hs_search`, `hs_store`, `hs_decide`, `hs_blockers`, `hs_graph`, `hs_my_cards`
