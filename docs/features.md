# Features

## Local-First AI Agent

Xagent runs entirely on your hardware. No data leaves your network unless you explicitly configure a cloud provider. The default setup uses Ollama for local LLM inference.

---

## Hardware-Aware Model Selection (LLM Check)

Built-in 4D scoring engine that analyzes your hardware and recommends the optimal model.

**Scoring dimensions:**
- **Quality** -- Model family rank, parameter count, quantization penalty, task-specific bonuses
- **Speed** -- Backend-specific TPS coefficients (CUDA, ROCm, Metal, CPU), quantization multipliers
- **Fit** -- Model memory footprint vs available RAM/VRAM
- **Context** -- Context window length vs target use case

**Weight presets by use case:**

| Use Case | Quality | Speed | Fit | Context |
|----------|---------|-------|-----|---------|
| General | 0.40 | 0.35 | 0.15 | 0.10 |
| Coding | 0.55 | 0.20 | 0.15 | 0.10 |
| Reasoning | 0.60 | 0.15 | 0.10 | 0.15 |
| Chat | 0.40 | 0.40 | 0.15 | 0.05 |
| Fast | 0.25 | 0.55 | 0.15 | 0.05 |

**Supported backends:** NVIDIA CUDA (H100 through GTX series), AMD ROCm (MI300, 7900 XTX), Apple Metal (M1-M4), Intel Arc, CPU (AVX-512, AVX2, NEON).

**40+ curated models** across families: Llama 3.x, Qwen 2.5, DeepSeek-R1, Phi, Gemma, Mistral, StarCoder, LLaVA, embeddings.

```bash
xagent llm-check check
xagent llm-check recommend coding
xagent llm-check benchmark phi3
```

---

## Adaptive Platform Scaling

Automatically detects hardware tier and configures itself:

| Tier | RAM/VRAM | Typical Hardware | Default Model |
|------|----------|------------------|---------------|
| ultra_high | 80+ GB | A100, H100 | llama3.3:70b |
| high | 24+ GB | RTX 4090, Xavier | llama3.1:8b |
| medium | 8-16 GB | RPi 4 (8GB) | phi3:3.8b |
| low | 2-4 GB | RPi 3 | Gateway only |

---

## Tool System

The agent has access to sandboxed tools:

| Tool | Capability |
|------|-----------|
| `exec` | Shell command execution (deny-listed: rm -rf, curl, ssh, etc.) |
| `read_file` | Read files within workspace |
| `write_file` | Write/create files within workspace |
| `edit_file` | Patch files within workspace |
| `list_dir` | List directory contents |
| `web_search` | DuckDuckGo / Brave search |
| `web_fetch` | Fetch URL content |
| `llm_check` | Hardware analysis, model recommendation, Ollama management |
| `i2c` / `spi` | Hardware I/O (Linux) |
| `message` | Send messages to user via connected channels |
| `spawn` | Launch sub-agents for parallel tasks |

All file and exec operations are sandboxed to `~/.xagent/workspace/` when `restrict_to_workspace` is enabled (default).

---

## Skill System

Skills are markdown files (`SKILL.md`) that provide domain-specific knowledge and instructions to the agent.

**Built-in skills:** Docker, SSH, Pi health, security, task decomposition, hardware I/O, skill creation, and more.

**Community skills:** 10,000+ skills from the OpenClaw archive, searchable and installable via `skill_converter.py`.

**Create your own:**
```bash
xagent skills create my-skill
# Edit workspace/skills/my-skill/SKILL.md
```

The agent can also create skills autonomously using the built-in `skill-creator` skill.

---

## Multi-Channel Support

Connect the agent to messaging platforms:

| Channel | Status |
|---------|--------|
| Telegram | Supported |
| Discord | Supported |
| Slack | Supported |
| WhatsApp | Supported |
| LINE | Supported |
| MaixCam | Supported |

Configure channels in `~/.xagent/config.json` with bot tokens and allowed user IDs.

---

## Self-Upgrade

```bash
xagent upgrade --check    # Check for new releases
xagent upgrade            # Download + verify + replace binary
xagent upgrade --model    # Pull latest Ollama model
xagent upgrade --all      # Upgrade everything
```

Binary upgrades are verified with SHA256 checksums from GitHub releases. The agent checks for updates weekly via the heartbeat system.

---

## Security Hardening

- **Workspace sandboxing** -- All agent file/exec operations confined to `~/.xagent/workspace/`
- **Command deny-list** -- Blocks destructive commands (rm -rf, format, dd), network exfiltration (curl, wget, ssh, nc), privilege escalation (useradd, crontab)
- **Chinese service blocklist** -- `/etc/hosts` and `iptables` rules block Chinese cloud domains and IP ranges
- **Skill scanner** -- Flags skills that reference Chinese services before installation
- **Config permissions** -- `config.json` stored with `600` permissions
- **Systemd hardening** -- `ProtectSystem=strict`, `ReadWritePaths` limited to workspace
- **No telemetry** -- Zero analytics, tracking, or phone-home behavior
- **Chinese channels disabled** -- QQ, Feishu, DingTalk stubs return errors on init
- **Chinese providers removed** -- Zhipu, Moonshot, ShengSuanYun stripped from provider config

---

## Ollama Integration

Full Ollama REST API integration:

- List installed models
- Pull/delete models
- Benchmark models (tokens/sec)
- Monitor running models
- Auto-pull recommended model during install

---

## Memory and State

- **Session history** -- Per-channel conversation history with automatic context windowing
- **Long-term memory** -- `MEMORY.md` for persistent knowledge across sessions
- **State manager** -- Atomic key-value persistence for tools and agent state
- **Identity** -- Configurable agent personality via `IDENTITY.md` and `SOUL.md`

---

## Scheduled Tasks

Cron-style task scheduling via the heartbeat system:

- Weekly upgrade checks
- Periodic health reports
- Custom scheduled tasks via `~/.xagent/workspace/cron/`

---

## Optional: Memory Bridge

Qdrant-powered vector memory for semantic search across conversation history:

```bash
# Requires Qdrant running on localhost:6333
# Configured via memory_bridge.py
```
