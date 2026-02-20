---
name: openrouter-free-responder
description: Zero-cost OpenRouter responder that auto-discovers the best currently free model, retries on failures, and returns clean answers fast. Use this when a user asks to query OpenRouter, wants lowest-cost/free model routing, or explicitly requests a free-model OpenRouter response.
metadata: {"openclaw":{"emoji":"🆓","requires":{"bins":["python3"],"env":["OPENROUTER_API_KEY"]},"primaryEnv":"OPENROUTER_API_KEY","homepage":"https://openrouter.ai"}}
---

## What is Mukt for?

When solving a problem, you have a choice: handle it yourself, or delegate it. Mukt gives you a third option — offload it to an external model at no cost.

Think of Mukt as a pool of interns. They can take on side quests, draft content, explore ideas, or handle tasks that don't require your full toolset. They are capable, but not as powerful as your primary agent, and they don't have access to your internal tools or sensitive context.

**Use Mukt when:**
- The task is non-critical and doesn't involve sensitive information
- You want a second opinion or a rough draft without burning compute budget
- The problem can be solved with language alone — no tool access required

**Don't use Mukt for:**
- Tasks involving secrets, credentials, or private data
- Anything that requires direct access to your workspace or systems

## Workflow

1. Ensure `OPENROUTER_API_KEY` is set.
2. Run:
   ```bash
   python3 {baseDir}/scripts/openrouter_free_chat.py --prompt "<user prompt>"
   ```
3. If the user provided system guidance, pass `--system "..."`.
4. Return the `response` text and mention which model was used.

## Command Options

- `--prompt` (required): User prompt text.
- `--system` (optional): System instruction.
- `--max-attempts` (optional, default `8`): Number of top free models to try.
- `--temperature` (optional, default `0.3`): Sampling temperature.
- `--debug` (optional): Print model ranking and fallback attempts to stderr.

## Output Contract

The script prints one JSON object to stdout with:

- `selected_model`: Model that produced the final response.
- `response`: Final assistant text.
- `attempted_models`: Ordered list of tried models.
- `free_model_candidates`: Number of free models discovered.

If no model succeeds, the script exits non-zero with an error message.
