---
name: LLM Evaluator
version: 1.0.0
description: LLM-as-a-Judge evaluation system with Langfuse integration
author: aiwithabidi
---

# LLM Evaluator 

LLM-as-a-Judge evaluation system using Langfuse. Score AI outputs on relevance, accuracy, hallucination, and helpfulness. Backfill scoring on historical traces. Uses GPT-5-nano for cost-efficient judging.

## Usage

```bash
# Test with sample cases
python3 scripts/evaluator.py test

# Score a specific Langfuse trace
python3 scripts/evaluator.py score <trace_id>

# Score with a single evaluator
python3 scripts/evaluator.py score <trace_id> --evaluators relevance

# Backfill scores on recent unscored traces
python3 scripts/evaluator.py backfill --limit 20
```

### Evaluators

- **relevance** (0-1) — How relevant is the response to the query?
- **accuracy** (0-1) — Is the response factually correct?
- **hallucination** (0-1) — Does the response contain fabricated information?
- **helpfulness** (0-1) — How useful is the response?

## Requirements

- `OPENROUTER_API_KEY` environment variable (for GPT-5-nano judge)
- `LANGFUSE_PUBLIC_KEY` and `LANGFUSE_SECRET_KEY` environment variables
- `LANGFUSE_HOST` — your Langfuse instance URL
- Python 3.10+
- `langfuse`, `requests` packages

## Credits

Built by **AgxntSix** — AI ops agent by [M. Abidi](https://www.linkedin.com/in/mohammad-ali-abidi)
 [agxntsix.ai](https://www.agxntsix.ai) | Part of the **AgxntSix Skill Suite** for OpenClaw agents
