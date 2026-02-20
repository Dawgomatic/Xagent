# Xagent - AI Agent Framework

One-command setup for Xavier, RPi3, RPi4, and x86_64.

---

## Quick Start

```bash
# Install everything (one command)
./start.sh

# Manage services
./manage.sh start     # Start all services
./manage.sh stop      # Stop everything
./manage.sh status    # Check status
./manage.sh logs      # View logs
./manage.sh test      # Test agent
```

Services auto-start on boot.

---

## What Gets Installed

- Platform detection (Xavier/RPi3/RPi4/x86_64)
- Python (3.8/3.10/3.12 based on Ubuntu version)
- Go 1.26.0
- Ollama + optimal model for your hardware
- Xagent AI agent binary
- Systemd services (auto-start on boot)

---

## Auto-Start on Boot

After running `./start.sh`, these services start automatically on every boot:
- `ollama.service` - AI model server
- `xagent-gateway.service` - AI agent gateway

Manage with: `./manage.sh enable|disable`

---

## Documentation

See `docs/` folder for detailed guides.

---

## Usage

```bash
# CLI mode
xagent agent -m "your question"

# Interactive mode
xagent agent

# Check status
xagent status

# Hardware-aware model recommendation
xagent llm-check hw-detect          # Detect hardware (CPU, GPU, RAM)
xagent llm-check check              # Full analysis: score all models
xagent llm-check recommend coding   # Top picks for coding
xagent llm-check installed          # Rank your installed Ollama models
xagent llm-check pull llama3.2:3b   # Download a model
xagent llm-check benchmark phi3     # Benchmark a model
```

---

## Platform Support

| Platform | Ubuntu | Python | Model |
|----------|--------|--------|-------|
| Xavier | 20.04 | 3.8 | llama3.1:8b |
| RPi4 | 22.04 | 3.10 | phi3:3.8b |
| RPi4 | 24.04 | 3.12 | phi3:3.8b |
| RPi3 | 22.04 | 3.10 | Gateway only |

---

## Repository Structure

```
.
├── cmd/xagent/       # CLI entry point
├── pkg/              # Go packages (agent, config, channels, providers, etc.)
├── workspace/        # Built-in skills
├── skills/           # OpenClaw community skill archive (10,000+)
├── reference/        # Vanilla upstream repos (git submodules, read-only)
│   ├── picoclaw/     # Original picoclaw source
│   ├── nanobot/      # Original nanobot project
│   └── openclaw-skills/  # Original skill archive
├── start.sh          # Master installer (run once)
├── manage.sh         # Service management (generated at install)
├── skill_converter.py # Skills search, install, and conversion tool
├── memory_bridge.py  # Optional Qdrant memory bridge
├── Makefile          # Build system
├── go.mod / go.sum   # Go module definition
└── docs/             # Documentation
```

---

## Setup in 3 Steps

1. **Install:** `./start.sh`
2. **Start:** `./manage.sh start`
3. **Use:** `xagent agent -m "Hello!"`

---

100% local, private, secure AI agents.
