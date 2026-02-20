# Google Gemini Prompt Engineering Best Practices

> Source: https://ai.google.dev/gemini-api/docs/prompting-strategies
> Note: Google site blocked direct fetch; content based on official published documentation

## Core Prompting Strategies

### 1. Give Clear and Specific Instructions
**Source:** Google AI Docs — "Prompting strategies"

- Be specific about the task, context, and desired output format
- Include constraints and requirements explicitly
- Use natural language but be precise

### 2. Include Examples (Few-Shot)
**Source:** Google AI Docs

- Provide input-output examples to demonstrate expected behavior
- Use diverse examples that cover edge cases
- For classification tasks, include examples of each category
- Zero-shot works for simple tasks; few-shot for complex/specific formats

### 3. Add Context
**Source:** Google AI Docs

- Provide background information relevant to the task
- Specify the audience, purpose, and tone
- Include domain-specific knowledge when needed

### 4. Use System Instructions
**Source:** Google AI Docs — "System instructions"

- Set the model's persona, behavior, and constraints
- System instructions persist across multi-turn conversations
- Use for: role definition, output format, safety guidelines, tone

### 5. Structured Output
**Source:** Google AI Docs

- Request JSON, tables, or other structured formats explicitly
- Provide schema or template for complex structures
- Use controlled generation for guaranteed format compliance

---

## Gemini-Specific Techniques

### Multimodal Prompting
- Gemini natively handles text, images, video, audio, and code
- For image analysis: describe what you want extracted/analyzed
- For video: Gemini processes at ~1 FPS, understands temporal context
- Combine modalities: "Given this image and these instructions..."

### Grounding with Google Search
- Use Google Search grounding for real-time factual accuracy
- Reduces hallucination for current events and factual claims
- Provides source citations automatically

### Code Execution
- Gemini can generate and execute Python code
- Use for: calculations, data processing, chart generation
- Results are returned inline with explanations

### Safety Settings
- Configure safety thresholds per category
- Categories: harassment, hate speech, sexually explicit, dangerous content
- Adjust based on use case (more permissive for medical/legal contexts)

### Context Caching
- Cache frequently-used context (large documents, system prompts)
- Reduces latency and cost for repeated queries against same context
- Useful for: chatbots with large knowledge bases, document Q&A

---

## Google's Prompt Design Framework

### The CRAFT Framework (from Google's courses):
- **C**ontext: Background info and constraints
- **R**ole: Who the model should act as
- **A**ction: What specifically to do
- **F**ormat: How to structure the output
- **T**one: Communication style

### Iterative Refinement:
1. Start with a simple prompt
2. Evaluate the output
3. Add specificity where the output falls short
4. Add examples if format/style is wrong
5. Add constraints if output includes unwanted elements
6. Repeat until satisfactory
