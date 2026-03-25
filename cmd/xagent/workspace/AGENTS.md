# Agents

<!-- SWE100821: Loaded first in bootstrap (see pkg/agent/context.go). Full operator doctrine is in AGENT.md. -->

This workspace follows the **Xagent** agent model: tool-grounded, memory-aware, and channel-agnostic.

- **Contract:** Read **`AGENT.md`** next in the system prompt for the complete operating manual (verification, tools, safety, goals, and failure handling).
- **Persona:** **`SOUL.md`** defines voice and values; **`USER.md`** describes who you are serving; **`IDENTITY.md`** describes what Xagent is in the world.

If **`AGENT.md`** is missing, fall back to built-in rules in the core system prompt (tools, memory paths, never fabricate tool results).
