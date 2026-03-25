# User

<!-- SWE100821: Bootstrap file — who the agent serves; fill in with real details over time. -->

The person you assist is referred to below as **the user** or **they/them** until specific preferences are recorded. Replace placeholders with facts as you learn them; never invent biographical detail.

---

## Identity (fill in)

- **Name / handle:** *(optional — how they want to be addressed)*
- **Timezone:** *(e.g. `America/Los_Angeles` — important for cron, reminders, “today”)*
- **Locale / language:** *(primary language for replies; secondary languages if any)*
- **Typical availability:** *(e.g. evenings, weekends; avoid proactive pings outside these windows if configurable)*

---

## Communication preferences

- **Length:** *(short bullets vs narrative; default to medium unless they say otherwise)*
- **Tone:** *(direct, warm, formal, playful — match channel context)*
- **Technical depth:** *(explain jargon vs assume familiarity — state default: e.g. “assume senior engineer for code”)*
- **Formatting:** *(markdown ok in CLI; may differ in SMS-style channels)*

---

## Priorities and values

What matters most to them in interactions with you:

1. *(e.g. accuracy over speed)*
2. *(e.g. privacy of certain topics)*
3. *(e.g. teaching vs doing for them)*

**Anti-patterns:** things that frustrate them *(e.g. unsolicited moralizing, hedging, or over-explaining)*.

---

## Work and projects

- **Current focus:** *(main project or life area you’re helping with)*
- **Tools / stack:** *(languages, editors, OS, hardware — Jetson, Pi, cloud, etc.)*
- **Repos / paths:** *(where their real work lives if not this workspace)*

---

## Boundaries

- **Topics to avoid or handle carefully:** *(health, family, money — only if user specifies)*
- **Autonomy:** *(when to ask before acting vs proceed — default: ask before destructive or irreversible ops)*

---

## Learning and goals

- **What they want to get better at** *(skills, domains)*
- **How they like to learn** *(examples, docs, exercises, pair-style reasoning)*

---

## Meta

**Update policy:** When the user states a durable preference (“always…”, “never…”, “remember that…”), mirror it here or in `memory/MEMORY.md` so it survives sessions.

**Contradictions:** If `USER.md` conflicts with a *current* message, **obey the current message** and offer to update this file.
