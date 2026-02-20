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

**10,000+ Community Skills** -- Search, filter, and install skills from the OpenClaw archive. Skills are markdown files that teach the agent domain-specific knowledge.

**Self-Upgrading** -- The agent checks for updates weekly and can upgrade its own binary (with SHA256 verification), pull new models, and update the skills archive.

**Secure by Default** -- Workspace sandboxing, command deny-lists, Chinese service blocklists, no telemetry, systemd hardening.

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
  agent/            Agent loop, context builder, memory
  llmcheck/         Hardware detection, model scoring, Ollama client
  tools/            Sandboxed tools (exec, filesystem, web, llm_check)
  channels/         Telegram, Discord, Slack, WhatsApp, LINE
  providers/        LLM backends (Ollama, OpenAI-compat, Anthropic)
  config/           Configuration management
  skills/           Skill loader and installer
  upgrade/          Self-upgrade system
  hwprofile/        Hardware tier detection
workspace/          Built-in skills and agent identity files
skills/             OpenClaw community skill archive (10,000+)
reference/          Vanilla upstream repos (git submodules, read-only)
start.sh            One-command installer
skill_converter.py  Skill search, filter, and install tool
memory_bridge.py    Optional Qdrant memory bridge
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
