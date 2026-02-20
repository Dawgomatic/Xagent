# OpenAI Prompt Engineering Best Practices

> Source: https://platform.openai.com/docs/guides/prompt-engineering
> Note: OpenAI site blocked direct fetch; content based on official published documentation

## Six Strategies for Better Results

### 1. Write Clear Instructions
**Source:** OpenAI Docs — "Write clear instructions"

- Include details in queries to get more relevant answers
- Ask the model to adopt a persona
- Use delimiters to clearly indicate distinct parts of the input
- Specify the steps required to complete a task
- Provide examples (few-shot)
- Specify the desired length of the output

**Key insight:** Models can't read your mind. If outputs are too long, ask for brief replies. If too simple, ask for expert-level writing. If the format is wrong, demonstrate the format you'd like.

### 2. Provide Reference Text
**Source:** OpenAI Docs — "Provide reference text"

- Instruct the model to answer using a reference text
- Instruct the model to answer with citations from a reference text

**Key insight:** Models can confidently generate plausible-sounding but fabricated answers, especially about niche topics or citations. Providing reference text reduces fabrication.

### 3. Split Complex Tasks into Simpler Subtasks
**Source:** OpenAI Docs — "Split complex tasks"

- Use intent classification to identify the most relevant instructions
- For dialogue that requires very long conversations, summarize or filter previous dialogue
- Summarize long documents piecewise and construct a full summary recursively

**Key insight:** Just as software engineering decomposes complex systems into modules, the same principle applies to LLM tasks.

### 4. Give the Model Time to Think
**Source:** OpenAI Docs — "Give time to think"

- Instruct the model to work out its own solution before rushing to a conclusion
- Use inner monologue or structured queries to hide reasoning process
- Ask the model if it missed anything on previous passes

**Key insight:** A model makes more reasoning errors when answering immediately vs. taking time to reason. "Chain of thought" prompting can help.

### 5. Use External Tools
**Source:** OpenAI Docs — "Use external tools"

- Use embeddings-based search for efficient knowledge retrieval
- Use code execution for accurate calculations or API calls
- Give the model access to specific functions/tools

**Key insight:** Compensate for model weaknesses by feeding outputs from other tools. E.g., a text retrieval system can tell the model about relevant documents.

### 6. Test Changes Systematically
**Source:** OpenAI Docs — "Test changes systematically"

- Evaluate model outputs with reference to gold-standard answers
- Use automated evals where possible

**Key insight:** Sometimes a change that improves one example degrades others. Define a comprehensive test suite ("eval") that represents the full range of expected inputs.

---

## OpenAI-Specific Techniques

### Structured Outputs
- Use JSON mode or function calling to force structured output
- Provide JSON schema in system prompt for consistent format
- Use `response_format: { type: "json_object" }` for guaranteed JSON

### System Messages
- Set behavior, personality, and constraints in the system message
- System messages are treated with higher priority than user messages
- Keep system messages concise but comprehensive

### Temperature & Parameters
- Temperature 0: Deterministic, best for factual/analytical tasks
- Temperature 0.7-1.0: Creative tasks, brainstorming
- Top-p: Alternative to temperature for controlling randomness
- Max tokens: Control output length explicitly

### Tool/Function Calling
- Define tools with clear descriptions and parameter schemas
- The description field is the "prompt" for when to use the tool
- Provide examples of when each tool should and shouldn't be used
