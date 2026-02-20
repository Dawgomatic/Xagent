# Context Engineering Principles

## Context Engineering vs. Prompt Engineering

### The Shift (2025)
In mid-2025, Andrej Karpathy coined the term **"context engineering"** to describe the evolution beyond simple prompt engineering. The key insight:

> "It's not about the prompt anymore. It's about the context. The art of filling the context window with exactly the right information at the right time." — Andrej Karpathy, 2025

**Prompt engineering** = How you phrase the question
**Context engineering** = What information you put in the context window, how you structure it, and what you leave out

### Why Context > Prompt
- A mediocre prompt with perfect context outperforms a perfect prompt with mediocre context
- Most LLM failures are context failures, not prompt failures
- The context window is your entire "workspace" — system prompt, user message, retrieved docs, tool outputs, conversation history, examples

---

## The 7 Principles of Context Engineering

### 1. Right Information, Right Time
**Principle:** Only include information the model needs for THIS specific step.

- Don't dump everything into every prompt
- Use retrieval (RAG) to pull relevant context dynamically
- Filter conversation history to relevant turns
- Remove stale or contradictory information

**Anti-pattern:** Stuffing the entire codebase into every coding prompt
**Pattern:** Retrieve only the files/functions relevant to the current task

### 2. Structure Determines Understanding
**Principle:** How you organize context matters as much as what you include.

- Use clear sections with headers or XML tags
- Put instructions before data (the model reads sequentially)
- Group related information together
- Use consistent formatting throughout

**Anti-pattern:** A wall of unstructured text mixing instructions, data, and examples
**Pattern:** `<instructions>` → `<context>` → `<examples>` → `<data>` → `<output_format>`

### 3. Recency and Primacy Effects
**Principle:** Models pay most attention to the beginning and end of context.

- Put critical instructions at the TOP (system prompt)
- Put the specific task/question at the BOTTOM (user message)
- Middle sections are for reference material
- Repeat critical constraints at the end if context is very long

**Source:** Anthropic long-context tips, confirmed by research on "lost in the middle" phenomenon (Liu et al., 2023)

### 4. Context Window Budget Management
**Principle:** Treat your context window like a scarce resource with a budget.

| Priority | Content | Allocation |
|----------|---------|------------|
| Critical | System prompt + core instructions | 5-10% |
| High | Current task + immediate context | 20-30% |
| Medium | Examples + reference docs | 20-30% |
| Low | Conversation history | 10-20% |
| Reserve | Model output space | 20-30% |

**Rule:** Always reserve at least 20% of context window for the model's response.

### 5. Compression Without Loss
**Principle:** Summarize, filter, and compress context while preserving decision-relevant information.

Techniques:
- **Summarize history:** Replace old conversation turns with a summary
- **Extract key facts:** Pull only relevant data points from large documents
- **Use references:** "As described in the project brief above" instead of repeating
- **Hierarchical context:** Overview first, details available on request

### 6. Context Determines Behavior More Than Instructions
**Principle:** What the model "sees" shapes its behavior more than what you "tell" it.

- If you include aggressive text in context, the model may mirror it
- If examples show a specific format, the model follows that format
- The model infers unstated patterns from provided context
- Context is implicit instruction

**Implication for agents:** The AGENTS.md, SOUL.md, and conversation history ARE the primary instructions, not just the system prompt.

### 7. Iterative Context Refinement
**Principle:** Build context iteratively — start minimal, add based on failures.

Process:
1. Start with minimal context (instruction + task)
2. Run and evaluate output
3. Identify failure mode (wrong format? missing info? hallucination?)
4. Add the minimum context needed to fix that failure
5. Repeat until quality threshold is met
6. Remove any context that doesn't contribute to quality

---

## Context Engineering for Agents

### Agent Context Architecture
```
┌─────────────────────────────────────┐
│ System Prompt (Identity + Rules)    │  ← Persistent, loaded every turn
├─────────────────────────────────────┤
│ Workspace Files (AGENTS.md, etc.)   │  ← Semi-persistent, updated rarely
├─────────────────────────────────────┤
│ Retrieved Context (RAG / Memory)    │  ← Dynamic, per-task
├─────────────────────────────────────┤
│ Conversation History (compressed)   │  ← Rolling, summarized
├─────────────────────────────────────┤
│ Current Task + User Message         │  ← Fresh each turn
├─────────────────────────────────────┤
│ Tool Results                        │  ← Ephemeral, current turn only
└─────────────────────────────────────┘
```

### Sub-Agent Context Design
When spawning sub-agents, context engineering is critical:

1. **Minimal viable context:** Only what the sub-agent needs for its task
2. **Clear boundaries:** What it should and shouldn't do
3. **Output format:** Exactly how to report results
4. **No leakage:** Don't include parent agent's full context
5. **Task-complete framing:** The sub-agent should have everything it needs in its initial prompt

### Memory as Context
- Short-term: Current conversation (context window)
- Medium-term: Session summaries, recent decisions (RAG retrieval)
- Long-term: Persistent facts, preferences, relationships (vector DB + knowledge graph)
- The art is knowing which memory layer to query for each task

---

## Key Research References

- Liu et al. (2023). "Lost in the Middle: How Language Models Use Long Contexts" — Models struggle with information in the middle of long contexts
- Karpathy, A. (2025). "Context Engineering" — Reframing prompt engineering as context engineering
- Anthropic (2025). "Long Context Tips" — Official guidance on managing large context windows
- OpenAI (2024). "Prompt Engineering Guide" — Six strategies for better results
- Google (2025). "Gemini Prompting Strategies" — Multimodal context engineering
