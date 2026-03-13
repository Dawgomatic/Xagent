# MISO — Mission Inline Skill Orchestration

> Simple ingredients. Rich flavor. 

## What is MISO?

The first agentic UI framework that lives inside your messaging app.

No React. No deploy. No code. Just a SKILL.md file.

## Origin

Created by Shunsuke Hayashi (@ShunsukeHayashi) and Miyabi 
Date: 2026-02-17
Location: Tokyo, Japan

## Core Concept

MISO transforms any messaging app into a multi-agent mission control using only native messaging features:

- **message edit** → Real-time dashboard updates
- **inline buttons** → Human in the Loop approval
- **reactions** → Lightweight feedback
- **emoji + Unicode** → Rich visual language

Zero external dependencies. Zero deployment. Zero coding.

## Why "MISO"?

味噌 (miso) = Simple ingredients (soybeans + koji) → Rich, deep flavor

MISO = Simple ingredients (SKILL.md + message edit) → Full Agentic UI

A Japanese-born concept for the global AI era.

## The Problem MISO Solves

Every AI interface today is a black box:

```
User: "Do this"
AI: ............
AI: "Done"
User: "That's not what I wanted"
```

MISO makes AI transparent:

```
User: "Do this"

 researcher — RUNNING
 "Analyzing competitor pricing..."
▓▓▓▓▓▓▓▓░░░░░░░░ 50%

 writer — WRITING
 "Leading with price advantage"
 560 / 3,000 words

[ Approve] [ Revise] [ Cancel]
```

## UI Paradigm Shift

```
CLI        → "Type command, see result"
GUI        → "Click button, see result"
Chat UI    → "Talk to AI, see result"
MISO       → "Delegate to agents, see process, intervene"
```

## 4+1 Layer UX Model

MISO uses a 4+1 layer visual hierarchy for instant status recognition:

| Layer | Element | Info Density | Speed |
|-------|---------|--------------|-------|
| Layer 0 |  Pin | Minimal (presence only) | Instant upon opening chat |
| Layer 0.5 |  ackReaction | Minimal (receipt confirmation) | Instant upon message receipt |
| Layer 1 | Reactions | Minimal (state only) | Instant in chat list |
| Layer 2 | Message body | Medium (progress, agents) | Seconds after opening |
| Layer 3 | Inline buttons | Action | Approval/intervention |

## Comparison

| | ChatGPT | Claude | Gemini | MISO |
|--|---------|--------|--------|------|
| Agent thinking visible |  | △ |  |  |
| Multi-agent |  |  |  |  |
| Mid-process intervention |  |  |  |  |
| Partial completion handling |  |  |  |  |
| Progress tracking |  |  |  |  |
| Approval gates |  |  |  |  |
| Cost tracking |  |  |  |  |
| Dedicated app required | Yes | Yes | Yes | No |
| Code required | N/A | N/A | N/A | Zero |

## Design System

See `DESIGN-SYSTEM.md` for Telegram-safe visual language.

## Phases

1. **INIT** — Agents spawning, task decomposition visible
2. **RUNNING** — Real-time progress, agent thinking, interim results
3. **PARTIAL** — Some agents complete, others still running
4. **AWAITING APPROVAL** — Human in the Loop gate with inline buttons
5. **COMPLETE** — Results summary, key findings, cost report
6. **ERROR** — Error handling, retry UI

## Tech Stack

- OpenClaw (any version)
- Any messaging app with message edit + inline buttons
- SKILL.md (this file)
- That's it.

## License

Open source. Free forever.

---

*Simple ingredients. Rich flavor.* 
*Born in Tokyo. Built for the world.* 
