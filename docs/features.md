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
| WhatsApp | Supported (via Node.js bridge) |
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

## Agent Continuous Improvement (Sleep Cycle)

During extended continuous idle periods, the agent scales its offline background operations (Sleep Cycle) in proportion to the activity and data processed while "awake":

- **Self-Improvement**: Updates reference repositories and pulls new code from GitHub.
- **Background Thinking**: Writes new code for itself, tests hypotheses, and stores learning.
- **Dynamic Resource Scaling**: The processing power footprint for this self-improvement scales linearly with the amount of data the agent observed while "awake" (more interaction -> deeper "sleep" improvement).
- **Dynamic Wake-ups**: Seamlessly handles sudden wake-up interruptions if a user interacts with the agent mid-sleep.

Location: `pkg/agent/sleep.go`

---

## Reinforcement Learning (RL) Framework

Xagent incorporates an integrated OpenClaw-RL training pipeline that learns from user conversations over time:

- **RL Proxy Server**: A self-hosted OpenAI-compat proxy (`docker-compose.rl.yml`) intercepts interactions, aggregates session metadata, and exports conversation trajectories.
- **Feedback Loops**: Collects user feedback implicitly and explicitly to refine the underlying models.
- **Automated Training**: Learns from and trains on conversations in the background (fenced within the sleep cycle), continuously optimizing agent behavior without manual intervention.

---

## Obsidian-Compatible Knowledge Vault

Rather than exclusively maintaining a linear memory array, the agent persists state using an Obsidian-compatible Knowledge Vault:

- **Graph Visualization**: Agent memories, code artifacts, and thought processes are written as interlinked Markdown documents.
- **Knowledge Webs**: Relationships are visualized using Markdown graph views for human inspectability.

Location: `pkg/vault`

---

## Observer UI Integration

An overlay interface is injected into PrismarineJS Minecraft Web Clients representing `claw_world`:

- **Bird's Eye Monitoring**: Translates standard third-person/first-person views into an isometric "observer mode" where human operators can casually oversee agent actions, states, and tasks.
- **HUD Integration**: Injects an overlay panel tailored to viewing agent status, obscuring default player-centric game indicators.

---

## Submodules
Xagent utilizes several standalone upstream open-source projects via git submodules for domain-specific context:

- **BitNet**: 1-bit LLM technology references (`reference/BitNet`).
- **hindsight**: Research references for reasoning (`reference/hindsight`).
- **openclaw-rl**: Core reinforcement learning loop implementation (`reference/openclaw-rl`).

---

## Optional: Memory Bridge

Qdrant-powered vector memory for semantic search across conversation history:

```bash
# Requires Qdrant running on localhost:6333
# Configured via memory_bridge.py
```

---

<!-- SWE100821: New features added 2026-02-21 -->

## Plan-Act-Reflect Loop

Enhanced reasoning with three-phase processing:

1. **Plan** — LLM generates a multi-step plan before executing
2. **Act** — Tools are called according to the plan
3. **Reflect** — After each tool result, the agent evaluates progress and may replan

Includes a **chain-of-thought scratchpad** — internal reasoning buffer that persists across tool calls but isn't sent to the user.

```
User message → Plan (3-5 steps) → Execute step 1 → Reflect → Execute step 2 → ... → Final response
```

Location: `pkg/agent/planner.go`

---

## Tool Middleware Layer

Cross-cutting concerns for all tool executions:

| Feature | Description |
|---------|-------------|
| **Pre/Post Hooks** | Run custom logic before/after any tool call |
| **Result Caching** | Deterministic tools (read_file, list_dir) cached with TTL |
| **Circuit Breaker** | Tools that fail 3x in a row are temporarily disabled |
| **Usage Analytics** | Per-tool success rate, latency, call count |
| **Tool-Use Learning** | Injects performance hints into system prompt |

Location: `pkg/tools/middleware.go`

---

## Tool Call Approval Mode

Configurable approval gate for destructive operations. When enabled:

- `exec`, `write_file`, `edit_file` send a confirmation prompt to the user before executing
- Recently-approved tools are auto-approved for a configurable window (default: 5 min)
- Especially useful for autonomous channels (Telegram, Discord)

Location: `pkg/tools/approval.go`

---

## Integrated Semantic Memory

Native Go Qdrant client replaces the disconnected Python memory_bridge.py:

- Auto-embeds conversation summaries and epoch journals via Ollama (`nomic-embed-text`)
- On each new message, similarity-searches and injects top-k relevant memories into context
- Falls back gracefully to file-based memory if Qdrant is unavailable
- Collection auto-created on first use

Location: `pkg/memory/semantic.go`

---

## Memory Importance Scoring

Multi-factor scoring for memory retrieval:

| Factor | Weight | Description |
|--------|--------|-------------|
| Recency | 0.25 | Exponential decay with 7-day half-life |
| Salience | 0.20 | Keyword detection (important, error, secret, etc.) |
| Novelty | 0.15 | Unique terms not seen in other recent memories |
| Reference | 0.10 | Log-scaled count of how often recalled |
| Semantic | 0.30 | Vector similarity score from Qdrant |

Location: `pkg/memory/scoring.go`

---

## Automatic Memory Consolidation

Periodic cron job that prevents memory bloat:

- **Weekly**: Clusters daily notes → generates weekly summary → archives daily notes
- **Monthly**: Merges weekly summaries → generates monthly summary → appends key insights to MEMORY.md
- Runs via the existing cron system

Location: `pkg/memory/consolidation.go`

---

## Specialized Subagent Roles

Built-in role templates for subagents:

| Role | Tools Allowed | Purpose |
|------|---------------|---------|
| **researcher** | web_search, web_fetch, read_file | Information gathering |
| **coder** | read_file, write_file, edit_file, exec | Code implementation |
| **reviewer** | read_file, list_dir | Code review (read-only) |
| **planner** | read_file, list_dir | Task decomposition |
| **sysadmin** | exec, read_file, list_dir | System health monitoring |

Each role has a tailored system prompt and restricted tool set.

Location: `pkg/orchestration/roles.go`

---

## Task DAG Execution

Directed acyclic graph engine for complex multi-step tasks:

- Models tasks as nodes with dependencies
- Executes independent tasks in parallel, dependent tasks sequentially
- Validates DAG for cycles and missing dependencies
- Human-readable execution summary with status markers

```
Task A (research) ──┐
                    ├── Task C (synthesis) ── Task D (output)
Task B (analysis) ──┘
```

Location: `pkg/orchestration/dag.go`

---

## Subagent Result Aggregation

When multiple subagents run in parallel, the aggregator:

- Collects all results
- Synthesizes a unified, coherent response via LLM
- Notes conflicts or contradictions between results
- Falls back to concatenation if synthesis fails

Location: `pkg/orchestration/aggregator.go`

---

## Dream Mode (Offline Reflection)

Genuinely unique — autonomous offline reflection during idle periods:

- When no messages arrive for 2+ hours, the agent enters "dream mode"
- Reviews recent epoch journals and daily notes
- Identifies patterns, contradictions, and gaps in knowledge
- Generates insights and writes them to daily notes
- Optionally sends a proactive message: " While reflecting, I noticed..."
- Minimum 12-hour interval between dream sessions

Location: `pkg/agent/dream.go`

---

## Personality Evolution

Tracks interaction patterns and adapts agent behavior:

- Observes user message length, response preferences, frequent topics
- Computes traits: verbosity (0=terse, 1=verbose), formality, creativity
- Injects adaptations into system prompt: "User prefers concise, direct responses"
- Generates monthly "personality diff" showing trait evolution
- Stores profile in `workspace/state/personality.json`

Location: `pkg/agent/personality.go`

---

## Context Compression via Distillation

Two-model approach for better context utilization:

- Small/fast model summarizes older conversation history
- Main model gets: compressed history + recent messages at full fidelity
- Preserves more information in the same context window vs simple truncation
- Configurable recent window size (default: 6 messages kept uncompressed)

Location: `pkg/agent/compression.go`

---

## Namespace-Based Sandbox (Linux)

Process isolation using Linux namespaces:

| Namespace | Effect |
|-----------|--------|
| PID | Can't see host processes |
| Network | No network by default (whitelist available) |
| UTS | Isolated hostname |

Additionally:
- Resource limits via cgroups (CPU%, memory MB)
- `SIGKILL` on parent death (`Pdeathsig`)
- Optional filesystem jail with bind mounts
- Falls back to basic timeout isolation on non-Linux

Location: `pkg/sandbox/namespace_linux.go`, `pkg/sandbox/namespace_other.go`

---

## Agent-to-Agent Communication

HTTP-based protocol for multi-agent cooperation:

- Structured message format: query, notify, response
- Peer registry with endpoint URLs
- Point-to-point and broadcast messaging
- HTTP handler mountable on the gateway (`/a2a`)
- Supports use cases like sensor-Pi → analysis-desktop

Location: `pkg/agent2agent/protocol.go`

---

## Skill Auto-Discovery

When the agent encounters a task it can't handle:

- Searches the 10,000+ skill archive by keyword matching
- Ranks results by relevance (name, description, tags)
- Suggests installation: "These skills might help: ..."
- Also triggered on tool errors via `SuggestForError()`
- Turns failures into learning opportunities

Location: `pkg/skills/autodiscover.go`

---

## Dynamic Tool Loading from Skills

Skills can register custom tools at runtime via SKILL.md frontmatter:

```yaml
---
tools:
  - name: weather_fetch
    description: Fetch weather for a location
    command: "curl -s 'wttr.in/{location}?format=3'"
    parameters:
      location: { type: string, description: City name }
---
```

Skills become first-class citizens in the tool system, not just prompt context.

Location: `pkg/skills/dynamic_tools.go`

---

## Hardware-Reactive Behavior

Automatic actions when hardware events occur:

| Device Class | Event | Action |
|-------------|-------|--------|
| Camera | Plugged in | " Camera connected. I can help with photos." |
| USB Storage | Plugged in | " USB storage connected. Want me to index it?" |
| Audio | Plugged in | " Audio device connected. Voice mode available." |
| USB Storage | Removed | " USB device disconnected." |

Custom reactions can be added programmatically.

Location: `pkg/devices/reactive.go`

---

## Voice Conversation Mode (TTS)

Completes the voice loop: voice input (Whisper) → agent → TTS output:

- **Piper** (preferred) — high-quality local TTS, no cloud dependency
- **espeak-ng** (fallback) — available on most Linux systems
- Auto-detects available TTS engine
- Generates WAV files with automatic cleanup

Location: `pkg/voice/tts.go`

---

## Provenance Tracking

Full lineage tracking for every agent response:

- Which tools were called (with success/failure and latency)
- Which skills contributed
- Which memories were recalled
- Which provider/model generated the response
- Plan step count and total iterations
- Stored as daily JSONL files in `workspace/provenance/`

Location: `pkg/agent/provenance.go`

---

## Composable Workflows (Recipes)

User-defined multi-step workflows as JSON files:

```json
{
  "name": "morning-briefing",
  "trigger": { "type": "cron", "schedule": "0 7 * * *" },
  "steps": [
    { "tool": "web_search", "args": { "query": "top tech news" } },
    { "tool": "exec", "args": { "command": "cat /sys/class/thermal/thermal_zone0/temp" } }
  ],
  "synthesize": "Give me a morning briefing with news and system health.",
  "channel": "telegram",
  "enabled": true
}
```

Bridges the gap between cron jobs and full agent conversations. Supports cron, event, and manual triggers.

Location: `pkg/workflows/recipe.go`
