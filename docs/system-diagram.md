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
| Identity | `pkg/identity/` | Unique AgentID + per-boot SessionID + time tracking |
| Epoch | `pkg/epoch/` | Wake/sleep lifecycle journaling (the "day" above sessions) |
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

## Lifecycle Hierarchy

<!-- SWE100821: Agent lifecycle layers mapped to human analogy -->

Each agent instance has a layered lifecycle, analogous to human consciousness:

| Layer | Human | Xagent | Persistence |
|-------|-------|--------|-------------|
| **Soul** | Who I am, forever | `AgentID` + `IDENTITY.md` + `SOUL.md` | Permanent (survives all restarts) |
| **Lifetime memory** | Things I know | `MEMORY.md`, daily notes | Permanent (file-based) |
| **Epoch** | One day (wake → sleep) | `workspace/epochs/*.json` | Across restarts (one file per boot) |
| **Session** | One conversation | `workspace/sessions/*.json` | Within/across epochs (per channel:user) |
| **Turn** | One Q&A exchange | `processMessage()` call | Ephemeral (in-memory only) |

```
PERMANENT ──────────────────────────────────────────────────────────
  AgentID (uuid, created once, never changes)
  IDENTITY.md / SOUL.md / USER.md
  MEMORY.md (long-term knowledge)

EPOCH (one boot cycle) ─────────────────────────────────────────────
  Wake:  Load previous epoch journal → inject into system prompt
  Live:  Record events, accumulate stats (messages, tool calls)
  Sleep: Write epoch journal (events + stats + reflection)
         └─ workspace/epochs/20260220-143205-<session>.json

SESSION (one conversation) ─────────────────────────────────────────
  Per channel:user pair (e.g. telegram:123456)
  History + summary, auto-pruned after TTL
         └─ workspace/sessions/<channel_user>.json

TURN (one request/response) ────────────────────────────────────────
  Context build → LLM call → tool calls → response
  Ephemeral, not persisted directly
```

### Epoch Wake/Sleep Flow

```
               ┌─────────────────────┐
               │    Process Start    │
               │   (gateway boot)    │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │   identity.New()    │
               │  Load/create AgentID│
               │  Mint SessionID     │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │   epoch.Wake()      │◀── Load last epoch journal
               │  Create new epoch   │    from workspace/epochs/
               │  Inject prev epoch  │──▶ System prompt gets
               │  into context       │    "Previous Session" section
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │   Agent runs...     │
               │  RecordEvent()      │──▶ Events logged in-memory
               │  UpdateStats()      │──▶ Counters incremented
               └──────────┬──────────┘
                          │
                     SIGTERM/SIGINT
                          │
                          ▼
               ┌─────────────────────┐
               │   epoch.Sleep()     │
               │  Capture final stats│
               │  Write reflection   │
               │  Save journal to    │──▶ workspace/epochs/
               │  disk (atomic)      │    <timestamp>-<session>.json
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │   Process Exit      │
               └─────────────────────┘
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
  ├─ Load identity (IDENTITY.md, SOUL.md, AgentID)
  ├─ Load previous epoch journal (wake-up recall)
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
    ├── epochs/          ← Epoch journals (one per boot cycle)
    ├── memory/          ← Long-term memory (MEMORY.md)
    ├── state/           ← Persistent state (key-value pairs, identity.json)
    ├── cron/            ← Scheduled jobs
    ├── skills/          ← User-installed skills
    ├── IDENTITY.md      ← Agent identity and name
    ├── SOUL.md          ← Agent personality
    ├── USER.md          ← User preferences
    └── AGENT.md         ← Agent behavior guidelines
```
