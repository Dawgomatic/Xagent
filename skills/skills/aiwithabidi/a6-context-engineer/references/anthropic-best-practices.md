# Anthropic Prompt Engineering Best Practices

> Source: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/
> Retrieved: 2026-02-19

## Overview
Anthropic recommends these techniques in order of broad effectiveness:
1. Be clear and direct
2. Use examples (multishot)
3. Let Claude think (chain of thought)
4. Use XML tags
5. Give Claude a role (system prompts)
6. Chain complex prompts
7. Long context tips

---

## 1. Be Clear, Direct, and Detailed
**Source:** Anthropic Docs — "Be clear and direct"

**Core principle:** Think of Claude as a brilliant new employee with amnesia who needs explicit instructions.

**Golden rule:** Show your prompt to a colleague with minimal context. If they're confused, Claude will be too.

### How to be clear:
- **Give contextual information:** What the results will be used for, who the audience is, what workflow this is part of, what success looks like
- **Be specific:** If you want only code output, say so explicitly
- **Use sequential steps:** Numbered lists/bullet points ensure exact execution order

### Example — Unclear vs. Clear:
- ❌ "Remove PII from this feedback"
- ✅ "Your task is to anonymize customer feedback for our quarterly review. Instructions: 1. Replace names with CUSTOMER_[ID]. 2. Replace emails with EMAIL_[ID]@example.com. 3. Redact phone numbers as PHONE_[ID]. 4. Leave product names intact. 5. Output only processed messages separated by ---."

---

## 2. Use Examples (Multishot Prompting)
**Source:** Anthropic Docs — "Multishot prompting"

**Core principle:** 3-5 diverse, relevant examples dramatically improve accuracy, consistency, and quality.

### Why examples work:
- Reduce misinterpretation of instructions
- Enforce uniform structure and style
- Boost performance on complex tasks

### Crafting effective examples:
- **Relevant:** Mirror your actual use case
- **Diverse:** Cover edge cases; vary enough to avoid unintended pattern matching
- **Clear:** Wrap in `<example>` tags (nested in `<examples>` for multiple)

### Pro tip:
Ask Claude to evaluate your examples for relevance, diversity, or clarity. Or have Claude generate more examples from your initial set.

---

## 3. Chain of Thought (CoT) Prompting
**Source:** Anthropic Docs — "Let Claude think"

**Core principle:** Giving Claude space to think step-by-step dramatically improves performance on complex tasks.

### Benefits:
- **Accuracy:** Reduces errors in math, logic, analysis
- **Coherence:** More organized responses
- **Debugging:** See where prompts are unclear

### Three levels (least → most complex):
1. **Basic:** Add "Think step-by-step" — lacks guidance on *how* to think
2. **Guided:** Outline specific thinking steps — lacks structure for separating answer
3. **Structured:** Use `<thinking>` and `<answer>` XML tags to separate reasoning from output

### Critical rule:
Always have Claude OUTPUT its thinking. Without outputting the thought process, no thinking occurs.

---

## 4. XML Tags for Structure
**Source:** Anthropic Docs — "Use XML tags"

**Core principle:** XML tags help Claude parse multi-component prompts accurately.

### Benefits:
- **Clarity:** Separate instructions, context, examples, formatting
- **Accuracy:** Prevent mixing up prompt components
- **Flexibility:** Easy to modify sections independently
- **Parseability:** Extract specific response parts via post-processing

### Best practices:
- Use consistent tag names throughout prompts
- Reference tags by name: "Using the contract in `<contract>` tags..."
- Nest tags for hierarchical content: `<outer><inner></inner></outer>`
- No canonical "best" tag names — just make them semantic

### Power combo:
Combine XML tags with multishot (`<examples>`) and CoT (`<thinking>`, `<answer>`) for maximum performance.

---

## 5. Role Prompting via System Prompts
**Source:** Anthropic Docs — "System prompts"

**Core principle:** Use the `system` parameter to set Claude's role. Put task instructions in the `user` turn.

### Why roles work:
- **Enhanced accuracy:** Domain-specific roles (legal, financial) significantly boost performance
- **Tailored tone:** CFO brevity vs. copywriter flair
- **Improved focus:** Role context keeps Claude within task bounds

### Pro tip:
Experiment with specificity: "data scientist" vs. "data scientist specializing in customer insight analysis for Fortune 500 companies" yield different results.

---

## 6. Prompt Chaining
**Source:** Anthropic Docs — "Chain complex prompts"

**Core principle:** Break complex tasks into smaller subtasks, each getting Claude's full attention.

### When to chain:
- Multi-step analysis, document processing, iterative content creation
- Multiple transformations, citations, or complex instructions

### How to chain:
1. Identify subtasks with distinct objectives
2. Use XML tags for clear handoffs between steps
3. Each subtask = single clear objective
4. Iterate based on performance

### Common patterns:
- Content pipeline: Research → Outline → Draft → Edit → Format
- Data processing: Extract → Transform → Analyze → Visualize
- Decision-making: Gather → List options → Analyze → Recommend
- Self-correction: Generate → Review → Refine → Re-review

### Advanced — Self-correction chains:
Chain prompts to have Claude review its own work. Catches errors for high-stakes tasks.

---

## 7. Extended Thinking
**Source:** Anthropic Docs — "Extended thinking tips"

**Core principle:** For very complex problems, enable extended thinking mode. Give high-level instructions rather than prescriptive steps.

### Key tips:
- Start with minimum thinking budget (1024 tokens), increase as needed
- Use general instructions first: "Think thoroughly" > step-by-step prescriptions
- The model's creativity in problem-solving may exceed your ability to prescribe
- For >32K thinking tokens, use batch processing
- Don't pass thinking output back as user input — it degrades results
- Multishot examples with `<thinking>` tags work well in extended thinking mode

### When to use extended thinking vs. manual CoT:
- Extended thinking: Complex STEM, multi-step reasoning, long-form analysis
- Manual CoT with XML: When thinking budget should be below 1024 tokens

---

## 8. Long Context Tips
**Source:** Anthropic Docs — "Long context tips"

### Key principles:
- Put critical instructions at the TOP and BOTTOM of prompts (primacy/recency effect)
- Reference documents by their XML tag names
- For very long contexts, use structured retrieval: ask Claude to find relevant sections first, then analyze
- Consider chunking: split large docs and process in parallel
