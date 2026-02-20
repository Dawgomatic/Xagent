# Xagent Complete Guide
**By SWE100821** | Comprehensive documentation for Xagent AI Agent

---

## 📋 Overview

Xagent is an ultra-lightweight personal AI Assistant built in Go, running on hardware from $10 devices to high-end Xavier boards. This guide covers everything from installation to advanced usage.

**Key Features:**
- 🪶 Ultra-Lightweight: <10MB RAM footprint
- ⚡️ Lightning Fast: 1s boot time
- 🌍 True Portability: Single binary across platforms
- 🔒 Secure: Workspace sandboxing, no telemetry
- 100% Local: With Ollama, no data leaves your machine

---

## 🚀 Quick Installation

### One-Command Setup
```bash
cd /home/dawg/Desktop/AI_agents
./start.sh
```

This automatically:
1. Detects your platform (Xavier/RPi3/RPi4/x86_64)
2. Detects Ubuntu version (20.04/22.04/24.04)
3. Installs correct Python version
4. Installs Go 1.21.6
5. Installs Ollama + optimal model
6. Builds Xagent
7. Creates systemd services
8. Enables auto-start on boot

---

## 🛠️ Platform Support

| Platform | Ubuntu | Python | Model | Performance |
|----------|--------|--------|-------|-------------|
| **Jetson Xavier** | 20.04 | 3.8 | llama3.1:8b | ⚡⚡⚡⚡⚡ GPU |
| **RPi 4** | 22.04 | 3.10 | phi3:3.8b | ⚡⚡⚡ CPU |
| **RPi 4** | 24.04 | 3.12 | phi3:3.8b | ⚡⚡⚡⚡ CPU |
| **RPi 3** | 22.04 | 3.10 | N/A | Gateway only |
| **x86_64** | Any | Auto | llama3.1:8b | ⚡⚡⚡⚡⚡ |

---

## 📖 Usage

### Command Line Interface

```bash
# One-off queries
xagent agent -m "What is 2+2?"
xagent agent -m "Write a Python script to sort a list"

# Interactive mode
xagent agent

# With custom session
xagent agent -s "project-work"

# Check status
xagent status
```

### Gateway Mode (Chat Apps)

```bash
# Start gateway (for Telegram, Discord, etc.)
xagent gateway

# With debug logging
xagent gateway --debug
```

---

## 🔧 Configuration

### Main Config File: `~/.xagent/config.json`

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.xagent/workspace",
      "restrict_to_workspace": true,
      "provider": "vllm",
      "model": "llama3.1:8b",
      "max_tokens": 4096,
      "temperature": 0.7,
      "max_tool_iterations": 15
    }
  },
  "providers": {
    "vllm": {
      "api_key": "not-needed",
      "api_base": "http://localhost:11434/v1"
    }
  },
  "channels": {},
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": false,
    "interval": 30
  },
  "devices": {
    "enabled": false,
    "monitor_usb": false
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  }
}
```

### Workspace Layout

```
~/.xagent/workspace/
├── sessions/          # Conversation sessions and history
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state
├── cron/             # Scheduled jobs database
├── skills/           # Custom skills
├── AGENTS.md         # Agent behavior guide
├── HEARTBEAT.md      # Periodic task prompts
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent personality
├── TOOLS.md          # Tool descriptions
└── USER.md           # User preferences
```

---

## 🔒 Security Features

### Workspace Sandboxing
By default, Xagent runs in a sandboxed environment:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": true
    }
  }
}
```

**Protected tools:**
- `read_file` - Only files within workspace
- `write_file` - Only files within workspace
- `list_dir` - Only directories within workspace
- `exec` - Command paths must be within workspace

### Blocked Commands
Even with `restrict_to_workspace: false`, these are blocked:
- `rm -rf`, `del /f` - Bulk deletion
- `format`, `mkfs` - Disk formatting
- `dd if=` - Disk imaging
- `/dev/sd*` writes - Direct disk writes
- `shutdown`, `reboot` - System shutdown
- Fork bombs

### Security Best Practices
✅ Keep `restrict_to_workspace: true`  
✅ Use localhost-only gateway: `127.0.0.1`  
✅ Disable unused channels  
✅ Review skills before installing  
✅ Monitor logs regularly  

---

## 💬 Chat Channels

### Telegram (Recommended)

1. **Create bot:** Talk to @BotFather
2. **Get user ID:** Talk to @userinfobot
3. **Configure:**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

4. **Start:** `xagent gateway`

### Discord

1. **Create bot:** https://discord.com/developers/applications
2. **Enable:** MESSAGE CONTENT INTENT
3. **Get user ID:** Enable Developer Mode, right-click avatar
4. **Configure:**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

---

## 🤖 Skills System

### Built-in Skills

```bash
# List available built-in skills
xagent skills list-builtin

# Install all built-in skills
xagent skills install-builtin

# List installed skills
xagent skills list

# Show skill details
xagent skills show weather
```

### Install from GitHub

```bash
# Install from official repository
xagent skills install sipeed/xagent-skills/weather

# Remove skill
xagent skills remove weather
```

### Custom Skills

Create in `~/.xagent/workspace/skills/my-skill/SKILL.md`:

```markdown
---
name: my-skill
description: Custom skill description
---

# My Skill

This skill does X, Y, Z...

## Usage
Agent can use this information to help with tasks.
```

---

## ⏰ Scheduled Tasks & Automation

### Heartbeat (Periodic Tasks)

Create `~/.xagent/workspace/HEARTBEAT.md`:

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)
- Report current time
- Check system status

## Long Tasks (use spawn for async)
- Search the web for AI news
- Check email and report important messages
```

**Configuration:**
```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30  // minutes
  }
}
```

### Cron Jobs

```bash
# Add one-time reminder
xagent cron add -n "Meeting reminder" -m "Remind about meeting" -e 3600

# Add recurring task
xagent cron add -n "Daily backup" -m "Run backup" -c "0 2 * * *"

# List jobs
xagent cron list

# Remove job
xagent cron remove <job_id>

# Enable/disable
xagent cron enable <job_id>
xagent cron disable <job_id>
```

---

## 🔍 Web Search

### DuckDuckGo (Built-in, No API Key)

```json
{
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

### Brave Search (Optional, Better Results)

1. Get API key: https://brave.com/search/api (2000 free/month)
2. Configure:

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      }
    }
  }
}
```

---

## 🎨 AI Models

### Recommended Models

| Model | Size | RAM | Use Case |
|-------|------|-----|----------|
| llama3.1:8b | 4.7GB | 8GB | General (Xavier, desktop) |
| phi3:3.8b | 2.3GB | 4GB | Low RAM (RPi4) |
| mistral:7b | 4.1GB | 8GB | Fast responses |
| deepseek-coder:6.7b | 3.8GB | 8GB | Programming |
| qwen2.5:7b | 4.4GB | 8GB | Multilingual |
| tinyllama:1.1b | 600MB | 2GB | Very low resources |
| llama3.1:70b | 40GB | 48GB+ | Best quality (powerful systems) |

### Switch Models

```bash
# Download model
ollama pull mistral:7b

# Update config
nano ~/.xagent/config.json
# Change "model": "mistral:7b"

# Restart
./manage.sh restart
```

---

## 🐳 Docker Deployment (Optional)

### Using Docker Compose

```bash
cd /home/dawg/Desktop/AI_agents/xagent

# Copy config
cp ~/.xagent/config.json config/config.json

# Build and start
docker-compose --profile gateway up -d

# Check logs
docker-compose logs -f xagent-gateway

# Stop
docker-compose --profile gateway down
```

---

## 🔄 Auto-Start Services

### Systemd Services

After running `./start.sh`, these services are created:

**ollama.service** - AI model server
**xagent-gateway.service** - AI agent gateway

Both automatically start on boot.

### Manual Service Control

```bash
# Start/stop/restart
sudo systemctl start xagent-gateway
sudo systemctl stop xagent-gateway
sudo systemctl restart xagent-gateway

# Status
sudo systemctl status xagent-gateway

# Enable/disable auto-start
sudo systemctl enable xagent-gateway
sudo systemctl disable xagent-gateway

# View logs
sudo journalctl -u xagent-gateway -f
sudo journalctl -u ollama -f
```

### Using manage.sh

```bash
./manage.sh start       # Start all services
./manage.sh stop        # Stop all services
./manage.sh restart     # Restart all services
./manage.sh status      # Check status
./manage.sh logs        # View logs
./manage.sh enable      # Enable auto-start
./manage.sh disable     # Disable auto-start
```

---

## 📊 Performance Optimization

### Xavier (GPU)
- Use larger models (llama3.1:8b, llama3.1:70b)
- Enable CUDA acceleration (automatic)
- No token limits needed

### Raspberry Pi 4
- Use smaller models (phi3:3.8b, tinyllama:1.1b)
- Reduce max_tokens to 2048 or less
- Consider adding swap space:

```bash
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### Raspberry Pi 3
- Skip local models
- Use as gateway only
- Or configure cloud API (see CLOUD_API_SETUP.md)

---

## 🆘 Troubleshooting

### Services Won't Start
```bash
./manage.sh logs
sudo systemctl status ollama
sudo systemctl status xagent-gateway
```

### Ollama Not Responding
```bash
curl http://localhost:11434/api/tags
sudo systemctl restart ollama
```

### Agent Errors
```bash
xagent status
cat ~/.xagent/config.json
```

### Slow Responses
- Use smaller model
- Reduce max_tokens in config
- Check system resources: `htop`, `free -h`

### Out of Memory
- Use smaller model (tinyllama:1.1b)
- Add swap space (see above)
- Check: `ollama ps`

---

## 📚 Additional Documentation

- **Quick Start:** `QUICKSTART.md` - Fast reference
- **Security:** `SECURITY_AUDIT.md` - Security analysis
- **Ollama Setup:** `OLLAMA_SETUP.md` - Detailed Ollama guide
- **Cloud APIs:** `CLOUD_API_SETUP.md` - Use OpenAI/Anthropic
- **Multi-Platform:** `MULTIPLATFORM.md` - Platform details

---

## 🎯 Best Practices

1. **Keep `restrict_to_workspace: true`** for security
2. **Review skills before installing**
3. **Monitor logs regularly:** `./manage.sh logs`
4. **Update periodically:**
   ```bash
   cd /home/dawg/Desktop/AI_agents/xagent
   git pull
   make build
   ./manage.sh restart
   ```
5. **Backup configuration:**
   ```bash
   cp ~/.xagent/config.json ~/config.backup.json
   ```

---

## 📝 Summary

### Installation
```bash
./start.sh
```

### Daily Use
```bash
xagent agent -m "your question"
```

### Management
```bash
./manage.sh start|stop|status|logs
```

### Services
- Auto-start on boot
- Run in background
- Managed via systemd

**Everything works locally, privately, and securely!**

---

**Created by SWE100821** | Complete guide for local AI agents
