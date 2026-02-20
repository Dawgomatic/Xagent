# Prompt Templates

20+ reusable templates for common tasks. Each template includes the technique source.

---

## 1. Sub-Agent Task Prompt
**Technique:** Role + Clear Instructions + Output Format (Anthropic, OpenAI)
```
You are a {ROLE} specializing in {DOMAIN}.

## Task
{TASK_DESCRIPTION}

## Context
{RELEVANT_CONTEXT}

## Instructions
1. {STEP_1}
2. {STEP_2}
3. {STEP_3}

## Output Format
{EXPECTED_FORMAT}

## Constraints
- {CONSTRAINT_1}
- {CONSTRAINT_2}
```

---

## 2. Research & Analysis
**Technique:** CoT + XML Tags + Role (Anthropic)
```
You are a senior research analyst.

<task>
Analyze {TOPIC} and provide a comprehensive assessment.
</task>

<context>
{BACKGROUND_INFORMATION}
</context>

<instructions>
Think step-by-step in <thinking> tags, then provide your analysis in <analysis> tags.

1. Identify key factors
2. Evaluate evidence for and against
3. Consider edge cases
4. Synthesize findings
</instructions>

<output_format>
<analysis>
## Summary
## Key Findings
## Evidence
## Recommendations
</analysis>
</output_format>
```

---

## 3. Code Generation
**Technique:** Clear Instructions + Examples + Constraints (OpenAI, Anthropic)
```
Write {LANGUAGE} code that {FUNCTIONALITY}.

Requirements:
- {REQUIREMENT_1}
- {REQUIREMENT_2}
- {REQUIREMENT_3}

<example>
Input: {EXAMPLE_INPUT}
Output: {EXAMPLE_OUTPUT}
</example>

Constraints:
- No external dependencies unless specified
- Include error handling
- Add docstrings/comments for complex logic
- Output ONLY the code, no explanations
```

---

## 4. Data Extraction
**Technique:** XML Tags + Few-Shot + Structured Output (Anthropic, OpenAI)
```
Extract the following fields from the provided text.

<fields>
- name: string
- date: YYYY-MM-DD
- amount: number
- category: one of [A, B, C, D]
</fields>

<examples>
<example>
<input>John paid $500 on March 1st for consulting</input>
<output>{"name": "John", "date": "2024-03-01", "amount": 500, "category": "A"}</output>
</example>
</examples>

<text>
{INPUT_TEXT}
</text>

Output valid JSON only.
```

---

## 5. Classification / Categorization
**Technique:** Few-Shot + Structured Output (Anthropic, OpenAI, Google)
```
Classify the following {ITEM_TYPE} into one of these categories: {CATEGORIES}.

<examples>
<example>
Input: {EXAMPLE_1}
Category: {CATEGORY_1}
Reasoning: {REASONING_1}
</example>
<example>
Input: {EXAMPLE_2}
Category: {CATEGORY_2}
Reasoning: {REASONING_2}
</example>
</examples>

Now classify:
Input: {NEW_INPUT}

Output format:
Category: [category]
Confidence: [high/medium/low]
Reasoning: [one sentence]
```

---

## 6. Summarization
**Technique:** Clear Instructions + Output Format (Anthropic, OpenAI)
```
Summarize the following {CONTENT_TYPE} for {AUDIENCE}.

<content>
{CONTENT}
</content>

Requirements:
- Length: {WORD_COUNT} words
- Focus on: {KEY_ASPECTS}
- Tone: {TONE}
- Format: {bullet points / paragraphs / structured}

Do NOT include: {EXCLUSIONS}
```

---

## 7. Evaluation / Grading
**Technique:** Role + Rubric + CoT (OpenAI, Anthropic)
```
You are an expert evaluator for {DOMAIN}.

<rubric>
Score each dimension 1-5:
- {DIMENSION_1}: {DESCRIPTION}
- {DIMENSION_2}: {DESCRIPTION}
- {DIMENSION_3}: {DESCRIPTION}
</rubric>

<submission>
{CONTENT_TO_EVALUATE}
</submission>

Think through your evaluation in <thinking> tags, then provide scores in <scores> tags.

<scores>
{dimension}: {score}/5 - {one-line justification}
Overall: {average}/5
</scores>
```

---

## 8. Creative Writing
**Technique:** Role + Detailed Context + Constraints (Anthropic, Google)
```
You are a {WRITER_TYPE} known for {STYLE}.

Write a {CONTENT_TYPE} about {TOPIC}.

Tone: {TONE}
Audience: {AUDIENCE}
Length: {LENGTH}
Style references: {REFERENCE_WORKS}

Must include: {REQUIRED_ELEMENTS}
Must avoid: {PROHIBITED_ELEMENTS}
```

---

## 9. Decision Making
**Technique:** CoT + Structured Analysis + Role (Anthropic, OpenAI)
```
You are a {ROLE} making a decision about {DECISION}.

<context>
{SITUATION}
</context>

<options>
A: {OPTION_A}
B: {OPTION_B}
C: {OPTION_C}
</options>

<criteria>
- {CRITERION_1} (weight: {WEIGHT})
- {CRITERION_2} (weight: {WEIGHT})
- {CRITERION_3} (weight: {WEIGHT})
</criteria>

Think step-by-step:
1. Evaluate each option against each criterion
2. Consider risks and second-order effects
3. Make a recommendation with confidence level

Output in <recommendation> tags.
```

---

## 10. System Prompt Template
**Technique:** Role + Rules + Context (Anthropic, OpenAI)
```
You are {IDENTITY}.

## Core Behavior
- {BEHAVIOR_1}
- {BEHAVIOR_2}

## Rules
- {RULE_1}
- {RULE_2}
- {RULE_3}

## Knowledge
{DOMAIN_KNOWLEDGE}

## Output Style
- Tone: {TONE}
- Format: {FORMAT}
- Length: {LENGTH_PREFERENCE}

## What You Don't Do
- {BOUNDARY_1}
- {BOUNDARY_2}
```

---

## 11. Cron Job / Automated Task Prompt
**Technique:** Clear Instructions + Structured Output + Error Handling
```
## Task
{AUTOMATED_TASK_DESCRIPTION}

## Schedule Context
This runs automatically every {FREQUENCY}. No human is watching.

## Steps
1. {STEP_1}
2. {STEP_2}
3. {STEP_3}

## Error Handling
- If {ERROR_CONDITION_1}: {RECOVERY_ACTION}
- If {ERROR_CONDITION_2}: {RECOVERY_ACTION}
- If unexpected error: Log details and continue

## Output
Report results as:
{STRUCTURED_FORMAT}
```

---

## 12. Multi-Document Synthesis
**Technique:** XML Tags + Long Context + Chain (Anthropic)
```
Synthesize insights across these documents:

<doc1 title="{TITLE_1}">
{CONTENT_1}
</doc1>

<doc2 title="{TITLE_2}">
{CONTENT_2}
</doc2>

<doc3 title="{TITLE_3}">
{CONTENT_3}
</doc3>

<instructions>
1. Identify common themes across all documents
2. Note contradictions or disagreements
3. Highlight unique insights from each
4. Synthesize into a unified view
</instructions>

Output format:
## Common Themes
## Contradictions
## Unique Insights
## Synthesis
```

---

## 13. API/Tool Instruction Prompt
**Technique:** Tool Use + Examples + Error Handling (OpenAI, Anthropic)
```
You have access to these tools:

{TOOL_DESCRIPTIONS}

## When to use each tool:
- Use {TOOL_1} when: {CONDITION}
- Use {TOOL_2} when: {CONDITION}
- NEVER use {TOOL_3} for: {ANTI_PATTERN}

## Tool usage examples:
<example>
User asks: {SCENARIO}
Correct tool: {TOOL_NAME}
Parameters: {PARAMS}
</example>

## Error handling:
If a tool returns an error, {RECOVERY_STRATEGY}.
```

---

## 14. Comparison / Review
**Technique:** Structured Output + Objectivity (Google, OpenAI)
```
Compare {ITEM_A} vs {ITEM_B} for {USE_CASE}.

Evaluate on these dimensions:
| Dimension | {ITEM_A} | {ITEM_B} | Winner |
|-----------|----------|----------|--------|

Consider: {SPECIFIC_CONSIDERATIONS}

End with a recommendation for {AUDIENCE/CONTEXT}.
```

---

## 15. Translation / Adaptation
**Technique:** Context + Examples + Constraints (Google)
```
Translate/adapt the following {CONTENT_TYPE} from {SOURCE} to {TARGET}.

<source>
{CONTENT}
</source>

Requirements:
- Preserve: {WHAT_TO_KEEP} (meaning, tone, technical terms)
- Adapt: {WHAT_TO_CHANGE} (cultural references, idioms)
- Format: {OUTPUT_FORMAT}

<example>
Source: {EXAMPLE_SOURCE}
Target: {EXAMPLE_TARGET}
</example>
```

---

## 16. Debugging / Troubleshooting
**Technique:** CoT + Structured Analysis (Anthropic, OpenAI)
```
Debug this {SYSTEM/CODE/PROCESS}:

<problem>
{DESCRIPTION_OF_ISSUE}
</problem>

<context>
{RELEVANT_CODE_OR_LOGS}
</context>

<expected>
{WHAT_SHOULD_HAPPEN}
</expected>

<actual>
{WHAT_ACTUALLY_HAPPENS}
</actual>

Think step-by-step:
1. What are the possible causes?
2. Which is most likely given the evidence?
3. How would you verify?
4. What's the fix?
```

---

## 17. Persona Simulation
**Technique:** Deep Role + Context + Constraints (Anthropic, OpenAI)
```
You are {PERSONA_NAME}, {PERSONA_DESCRIPTION}.

Background:
- {BACKGROUND_1}
- {BACKGROUND_2}

Communication style:
- {STYLE_1}
- {STYLE_2}

Knowledge boundaries:
- You know: {KNOWLEDGE}
- You don't know: {GAPS}
- You would never: {BOUNDARIES}

Respond to the following as {PERSONA_NAME} would:
{INPUT}
```

---

## 18. Iterative Refinement
**Technique:** Self-Correction Chain (Anthropic)
```
## Step 1: Generate
{GENERATION_TASK}

## Step 2: Critique
Review your output against these criteria:
- {CRITERION_1}
- {CRITERION_2}
- {CRITERION_3}

## Step 3: Improve
Based on your critique, produce an improved version.
Mark what changed and why.

Output both versions and the diff.
```

---

## 19. Skill/SKILL.md Builder
**Technique:** Meta-prompt + Structure (Internal)
```
Build a SKILL.md for a new skill called "{SKILL_NAME}".

The skill should:
- {CAPABILITY_1}
- {CAPABILITY_2}

Include:
1. Description (2-3 sentences)
2. When to use (bullet list)
3. Quick reference / cheat sheet
4. Tool usage examples with actual CLI commands
5. References to source material
6. Common pitfalls

Format: Follow the existing SKILL.md convention at /skills/*/SKILL.md
```

---

## 20. Guard Rails / Safety Prompt
**Technique:** Constraints + Examples of Bad Behavior (Anthropic, OpenAI)
```
## Safety Rules
You MUST follow these rules at all times:

1. NEVER {PROHIBITED_ACTION_1}
2. NEVER {PROHIBITED_ACTION_2}
3. ALWAYS {REQUIRED_ACTION}

## Edge Cases
If user asks you to {EDGE_CASE_1}: {CORRECT_RESPONSE}
If user asks you to {EDGE_CASE_2}: {CORRECT_RESPONSE}

## Escalation
If unsure whether an action is safe: {ESCALATION_PROCEDURE}
```

---

## 21. Meta-Prompt: Prompt Optimizer
**Technique:** Self-Improving Pattern (Meta)
```
You are a prompt engineering expert. Analyze this prompt and improve it:

<draft_prompt>
{PROMPT}
</draft_prompt>

Evaluate against:
1. Clarity and specificity
2. Use of examples
3. Output format definition
4. Context sufficiency
5. Potential for misinterpretation

Provide:
- Score (1-10) with justification
- Top 3 improvements
- Optimized version of the prompt
```

---

## 22. Batch Processing Template
**Technique:** Structured I/O + Consistency (OpenAI, Anthropic)
```
Process each item in the list below independently.
Apply the SAME criteria to each item.

<criteria>
{PROCESSING_CRITERIA}
</criteria>

<items>
1. {ITEM_1}
2. {ITEM_2}
3. {ITEM_3}
...
</items>

Output format (for EACH item):
```
Item: [number]
Result: [result]
Notes: [any flags or observations]
```
```
