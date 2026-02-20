# System Diagram

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                         XAGENT SYSTEM                            │
│                                                                  │
│  ┌──────────┐    ┌──────────────┐    ┌─────────────────────┐    │
│  │  CLI /   │    │   Gateway    │    │   Channel Adapters  │    │
│  │  Agent   │───▶│  (HTTP API)  │◀───│  Telegram, Discord  │    │
│  │  REPL    │    │  :18790      │    │  Slack, WhatsApp    │    │
│  └────┬─────┘    └──────┬───────┘    └─────────────────────┘    │
│       │                 │                                        │
│       ▼                 ▼                                        │
│  ┌──────────────────────────────────┐                            │
│  │          AGENT LOOP              │                            │
│  │                                  │                            │
│  │  Context Builder ──▶ LLM Call    │                            │
│  │       ▲                  │       │                            │
│  │       │              ToolCall?   │                            │
│  │       │              ▼           │                            │
│  │  Session ◀── Tool Registry ──────│──▶ Tool Execution          │
│  │  Manager     (sandboxed)         │                            │
│  └──────────────────────────────────┘                            │
│       │              │           │                                │
│       ▼              ▼           ▼                                │
│  ┌─────────┐  ┌───────────┐  ┌──────────┐  ┌──────────────┐    │
│  │ Skills  │  │   Tools   │  │  State   │  │  LLM Check   │    │
│  │ Loader  │  │ exec,read │  │ Manager  │  │  HW Detect + │    │
│  │         │  │ write,web │  │          │  │  Model Score  │    │
│  └─────────┘  └───────────┘  └──────────┘  └──────┬───────┘    │
│                      │                              │            │
│                      ▼                              ▼            │
│               ┌─────────────┐               ┌─────────────┐     │
│               │  Workspace  │               │   Ollama     │     │
│               │ ~/.xagent/  │               │  :11434      │     │
│               │  workspace/ │               │  Local LLM   │     │
│               └─────────────┘               └─────────────┘     │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  BACKGROUND SERVICES                                      │   │
│  │  Heartbeat  |  Cron  |  Health :28790  |  Self-Upgrade   │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘

   ┌───────────────────────────────────────────────┐
   │  SYSTEMD                                       │
   │  ollama.service        ──▶ Ollama LLM server   │
   │  xagent-gateway.service ──▶ Xagent gateway     │
   │  memory-bridge.service  ──▶ Qdrant bridge (opt) │
   └───────────────────────────────────────────────┘
```

## Component Map

| Component | Location | Purpose |
|-----------|----------|---------|
| CLI entry | `cmd/xagent/main.go` | Command dispatch: agent, gateway, llm-check, skills, upgrade |
| Agent loop | `pkg/agent/` | LLM conversation loop with tool calling |
| Tool registry | `pkg/tools/` | Sandboxed tools: exec, filesystem, web, I2C/SPI, llm_check |
| Channels | `pkg/channels/` | Telegram, Discord, Slack, WhatsApp, LINE adapters |
| Providers | `pkg/providers/` | LLM backends: Ollama, OpenAI-compat, Anthropic, Codex |
| Config | `pkg/config/` | JSON config at `~/.xagent/config.json` |
| Skills | `pkg/skills/` | Skill loader + installer (SKILL.md format) |
| LLM Check | `pkg/llmcheck/` | Hardware detection, 4D model scoring, Ollama API client |
| HW Profile | `pkg/hwprofile/` | Hardware tier detection for adaptive scaling |
| Upgrade | `pkg/upgrade/` | Self-upgrade from GitHub releases with SHA256 verification |
| Health | `pkg/health/` | HTTP health check endpoint |
| Heartbeat | `pkg/heartbeat/` | Periodic task scheduler |
| Session | `pkg/session/` | Conversation history management |
| State | `pkg/state/` | Persistent key-value state |

## Installation Flow

```
start.sh
   │
   ├─ 1. Detect Platform
   │     ├─ Ubuntu version (20.04 / 22.04 / 24.04)
   │     ├─ Hardware (Xavier / RPi3 / RPi4 / x86_64)
   │     └─ Architecture (amd64 / arm64 / armv6l)
   │
   ├─ 2. Install Dependencies
   │     ├─ Python (3.8 / 3.10 / 3.12 per Ubuntu version)
   │     ├─ Go 1.26.0
   │     ├─ Build tools (gcc, make)
   │     └─ Node.js (optional, for llm-checker reference)
   │
   ├─ 3. Install Ollama
   │     ├─ curl install script
   │     ├─ Enable + start systemd service
   │     └─ Pull recommended model (auto-selected by hardware tier)
   │
   ├─ 4. Build Xagent
   │     ├─ make deps
   │     ├─ make build
   │     └─ ln -sf build/xagent /usr/local/bin/xagent
   │
   ├─ 5. Configure
   │     ├─ xagent onboard (creates ~/.xagent/)
   │     ├─ Generate config.json (Ollama provider, recommended model)
   │     └─ chmod 600 config.json
   │
   ├─ 6. Create Services
   │     ├─ /etc/systemd/system/xagent-gateway.service
   │     ├─ Hardened: ProtectSystem=strict, ReadWritePaths
   │     └─ systemctl enable + start
   │
   ├─ 7. Security Hardening
   │     ├─ /etc/hosts blocklist (Chinese cloud domains)
   │     ├─ iptables rules (Chinese CIDR ranges)
   │     └─ Workspace sandboxing enabled
   │
   └─ 8. Generate manage.sh
         ├─ start / stop / restart / status
         ├─ logs / health / upgrade / skills
         └─ Done. Services auto-start on boot.
```

## Runtime Flow

```
User Message
     │
     ▼
Channel Adapter (or CLI)
     │
     ▼
Gateway routes to Agent Loop
     │
     ▼
Context Builder
  ├─ Load session history
  ├─ Load skills (SKILL.md files)
  ├─ Load identity (IDENTITY.md, SOUL.md)
  ├─ Load memory (MEMORY.md)
  └─ Build system prompt
     │
     ▼
LLM Provider (Ollama / Cloud)
     │
     ├─ Text response ──▶ Send to user
     │
     └─ Tool call ──▶ Tool Registry
                         │
                         ├─ Validate (sandbox, deny-list)
                         ├─ Execute
                         └─ Return result to LLM
                              │
                              ▼
                         (loop until no more tool calls)
                              │
                              ▼
                         Final response ──▶ User
```

## Data Flow

```
~/.xagent/
├── config.json          ← Agent configuration (provider, model, channels)
└── workspace/
    ├── sessions/        ← Conversation history (per channel/user)
    ├── memory/          ← Long-term memory (MEMORY.md)
    ├── state/           ← Persistent state (key-value pairs)
    ├── cron/            ← Scheduled jobs
    ├── skills/          ← User-installed skills
    ├── IDENTITY.md      ← Agent identity and name
    ├── SOUL.md          ← Agent personality
    ├── USER.md          ← User preferences
    └── AGENT.md         ← Agent behavior guidelines
```
