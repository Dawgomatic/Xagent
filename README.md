# Xagent

A local-first AI agent framework that runs on anything from a Raspberry Pi to a GPU server. Hardware-aware, self-upgrading, and fully private.

---

## What It Does

Xagent is a personal AI assistant that runs on your own hardware using open-source models via Ollama. It detects your hardware, selects the optimal model, and provides a tool-equipped agent accessible via CLI or messaging channels (Telegram, Discord, Slack, etc.).

```bash
./start.sh                           # Install everything, auto-start on boot
xagent agent -m "What can you do?"   # Talk to the agent
```

---

## Key Capabilities

**Hardware-Aware Model Selection** -- Built-in 4D scoring engine (Quality, Speed, Fit, Context) scores 40+ models against your specific CPU/GPU/RAM and recommends the best fit.

**Adaptive Scaling** -- Automatically detects whether it's running on an H100, a Jetson Xavier, or a Raspberry Pi, and adjusts model selection, token limits, and resource usage accordingly.

**Tool-Equipped Agent** -- The agent can execute shell commands, read/write files, search the web, manage Ollama models, control hardware I/O, and spawn sub-agents -- all sandboxed to the workspace.

**Reinforcement Learning & Continuous Improvement** -- Integrates with OpenClaw-RL to train its underlying models. The agent uses a Sleep Cycle during idle periods to reflect, self-improve, research, pull code updates, and run reinforcement learning over collected awake data.

**10,000+ Community Skills** -- Search, filter, and install skills from the OpenClaw archive. Skills are markdown files that teach the agent domain-specific knowledge.

**Self-Upgrading** -- The agent checks for updates weekly and can upgrade its own binary (with SHA256 verification), pull new models, and update the skills archive.

**Secure by Default** -- Workspace sandboxing (Linux namespaces), command deny-lists, Chinese service blocklists, no telemetry, systemd hardening.

**Plan-Act-Reflect Loop** -- Structured multi-step reasoning. The agent plans before acting, reflects after each tool call, and replans when needed.

**Cognitive Memory Stack** -- Four-layer memory: semantic (vector search via Qdrant), hindsight (retain/recall/reflect), temporal (time-aware queries like "what happened yesterday"), and memory scoring (recency, salience, novelty, reference count).

**Agent-to-Agent (A2A) Mesh** -- Peer discovery via UDP broadcast, hardware-aware task routing (GPU tasks routed to GPU peers), and shared vault synchronization across agents.

**Model Context Protocol (MCP)** -- Connect to any MCP server (filesystem, database, API) and use its tools natively inside the agent loop. Config-driven, no code changes needed.

**Cognitive Dashboard** -- Web UI at `/dashboard` for real-time agent introspection: epoch history, provenance logs, skill inventory, connected peers.

**Skill Fitness & Composition** -- Skills are scored by success rate, usage, and recency. The agent can compose new skills from existing ones during sleep cycles.

**Voice Conversation Loop** -- Full STT (Groq Whisper) to agent to TTS (Piper/espeak) pipeline for hands-free interaction.

**Dynamic Model Switching** -- Automatically switches the LLM model when available compute changes (e.g., GPU becomes available or RAM pressure increases).

---

## Supported Hardware

| Platform | Model | Performance |
|----------|-------|-------------|
| GPU Server (A100/H100) | llama3.3:70b | 80-120 tok/s |
| Desktop GPU (RTX 3080+) | llama3.1:8b | 35-70 tok/s |
| Jetson Xavier | llama3.1:8b | 5-7 tok/s |
| Raspberry Pi 4 (8GB) | phi3:3.8b | 1.5-2.5 tok/s |
| Raspberry Pi 3 | Cloud API | Gateway only |

---

## Repository Structure

```
cmd/xagent/         CLI entry point
pkg/                Go packages
  agent/            Agent loop, planner, context compression, personality, dream mode
  memory/           Semantic, hindsight, temporal memory, scoring
  tools/            Sandboxed tools (exec, filesystem, web, feedback)
  llmcheck/         Hardware detection, model scoring, Ollama client
  channels/         Telegram, Discord, Slack, WhatsApp, LINE
  providers/        LLM backends (Ollama, OpenAI-compat, Anthropic)
  config/           Configuration management
  skills/           Skill loader, fitness tracker, composer
  upgrade/          Self-upgrade system (binary, models, RL checkpoints)
  hwprofile/        Hardware tier detection and resource watching
  vault/            Obsidian-compatible knowledge vault and graph view
  agent2agent/      A2A peer discovery, task routing, vault sync
  mcp/              Model Context Protocol client
  orchestration/    Multi-agent task DAG execution
  dashboard/        Web UI for agent introspection
  sensors/          Embodied cognition sensor monitor
  voice/            STT/TTS voice conversation loop
  sandbox/          Linux namespace sandbox for tool execution
  health/           Health, readiness, and metrics endpoints
workspace/          Built-in skills and agent identity files
skills/             OpenClaw community skill archive (10,000+)
reference/          Vanilla upstream repos (git submodules, read-only)
start.sh            One-command installer
Makefile            Build system
```

---

## Documentation

| Document | Contents |
|----------|----------|
| [docs/quickstart.md](docs/quickstart.md) | Get running in 5 minutes |
| [docs/install.md](docs/install.md) | Full installation guide with manual steps |
| [docs/features.md](docs/features.md) | Complete feature reference |
| [docs/system-diagram.md](docs/system-diagram.md) | Architecture diagrams and process flows |

---

## License

MIT
