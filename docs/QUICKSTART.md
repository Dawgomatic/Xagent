# Quick Start Guide
**SWE100821** | Complete quick reference for AI Agents

---

## 🚀 Installation (One Command)

```bash
cd /home/dawg/Desktop/AI_agents
./start.sh
```

**What it does:**
- Auto-detects platform (Xavier/RPi3/RPi4/x86_64)
- Installs Python (3.8/3.10/3.12), Go, Ollama
- Downloads optimal model for your hardware
- Builds PicoClaw
- Creates systemd services (auto-start on boot)

**Installation takes:** 5-10 minutes

---

## 📋 Management Commands

```bash
cd /home/dawg/Desktop/AI_agents

./manage.sh start      # Start all services
./manage.sh stop       # Stop all services
./manage.sh restart    # Restart all services
./manage.sh status     # Check service status
./manage.sh logs       # View recent logs
./manage.sh test       # Test agent
./manage.sh enable     # Enable auto-start on boot
./manage.sh disable    # Disable auto-start
```

---

## 💬 Using the Agent

### CLI Mode (One-off queries)
```bash
picoclaw agent -m "What is 2+2?"
picoclaw agent -m "Write a Python hello world"
picoclaw agent -m "Search for AI news"
```

### Interactive Mode
```bash
picoclaw agent

# Then type your questions:
> What files are in my workspace?
> Create a todo list
> exit
```

### Check Status
```bash
picoclaw status        # Show configuration
picoclaw --version     # Show version
```

---

## 🔄 Auto-Start on Boot

Services **automatically start** on boot after installation.

**Installed services:**
- `ollama.service` - AI model server (port 11434)
- `picoclaw-gateway.service` - AI agent gateway (port 18790)

**Manual control:**
```bash
sudo systemctl start picoclaw-gateway
sudo systemctl stop picoclaw-gateway
sudo systemctl restart picoclaw-gateway
sudo systemctl status picoclaw-gateway

# View live logs
sudo journalctl -u picoclaw-gateway -f
sudo journalctl -u ollama -f
```

---

## 📂 File Locations

```
~/.picoclaw/
├── config.json              # Configuration
└── workspace/               # Agent workspace
    ├── sessions/            # Chat history
    ├── memory/              # Long-term memory
    ├── skills/              # Custom skills
    ├── IDENTITY.md          # Agent identity
    └── USER.md              # Your preferences

/var/log/
├── picoclaw-gateway.log     # Gateway logs
└── picoclaw-gateway-error.log

~/.ollama_model              # Current model name
```

---

## 🔧 Common Tasks

### Change AI Model
```bash
# List available models
ollama list

# Download new model
ollama pull mistral:7b

# Update config
nano ~/.picoclaw/config.json
# Change "model": "mistral:7b"

# Restart
./manage.sh restart
```

### Add Telegram Bot
```bash
# 1. Create bot: Talk to @BotFather on Telegram
# 2. Get user ID: Talk to @userinfobot
# 3. Edit config:
nano ~/.picoclaw/config.json

# Add under "channels":
{
  "telegram": {
    "enabled": true,
    "token": "YOUR_BOT_TOKEN",
    "allow_from": ["YOUR_USER_ID"]
  }
}

# 4. Restart
./manage.sh restart
```

### Install Skills
```bash
# List built-in skills
picoclaw skills list-builtin

# Install all built-in skills
picoclaw skills install-builtin

# Install from GitHub
picoclaw skills install sipeed/picoclaw-skills/weather
```

### View Agent Memory
```bash
cat ~/.picoclaw/workspace/memory/MEMORY.md
```

### Schedule Tasks
```bash
# Add reminder
picoclaw cron add -n "Check email" -m "Check my email" -e 3600

# List scheduled tasks
picoclaw cron list
```

---

## 🛠️ Platform-Specific Tips

### Jetson Xavier (Ubuntu 20.04)
- ✅ Best performance with GPU acceleration
- Model: `llama3.1:8b` (can handle larger models)
- Can enable all features without limitations

### Raspberry Pi 4 (Ubuntu 22.04/24.04)
- ✅ Good performance with CPU
- Model: `phi3:3.8b` (optimized for low resources)
- For faster: Use `tinyllama:1.1b`

### Raspberry Pi 3
- ⚠️ No local model (insufficient resources)
- Use as gateway only or configure cloud API
- See: `CLOUD_API_SETUP.md`

---

## 🐛 Troubleshooting

### Services won't start
```bash
# Check what's wrong
./manage.sh logs

# Check Ollama
curl http://localhost:11434/api/tags

# Restart everything
./manage.sh restart
```

### "Failed to connect to API"
```bash
# Check Ollama is running
systemctl status ollama

# Start if needed
sudo systemctl start ollama
```

### Agent gives errors
```bash
# Check configuration
picoclaw status

# View config
cat ~/.picoclaw/config.json

# Test Ollama directly
ollama run llama3.1:8b "test message"
```

### Slow responses
```bash
# Use smaller model
ollama pull tinyllama:1.1b

# Update config
sed -i 's/phi3:3.8b/tinyllama:1.1b/' ~/.picoclaw/config.json

# Reduce tokens
# Edit ~/.picoclaw/config.json
# Change "max_tokens": 2048
```

### Out of memory
```bash
# Check memory
free -h

# Use smaller model
ollama pull tinyllama:1.1b

# Add swap (RPi)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### Reinstall from scratch
```bash
# Stop services
./manage.sh stop

# Remove old installation
rm -rf ~/.picoclaw
sudo systemctl disable picoclaw-gateway
sudo rm /etc/systemd/system/picoclaw-gateway.service

# Reinstall
cd /home/dawg/Desktop/AI_agents
./start.sh
```

---

## 📊 Configuration Reference

### Basic Config (~/.picoclaw/config.json)
```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "vllm",
      "model": "llama3.1:8b",
      "max_tokens": 4096,
      "temperature": 0.7
    }
  },
  "providers": {
    "vllm": {
      "api_key": "not-needed",
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

### Available Models

| Model | Size | RAM | Speed | Best For |
|-------|------|-----|-------|----------|
| llama3.1:8b | 4.7GB | 8GB | ⚡⚡⚡ | General (Xavier) |
| phi3:3.8b | 2.3GB | 4GB | ⚡⚡⚡⚡ | Low RAM (RPi4) |
| mistral:7b | 4.1GB | 8GB | ⚡⚡⚡⚡ | Fast responses |
| deepseek-coder:6.7b | 3.8GB | 8GB | ⚡⚡⚡ | Programming |
| tinyllama:1.1b | 600MB | 2GB | ⚡⚡⚡⚡⚡ | Very low RAM |

---

## 📚 More Documentation

- **Security:** `SECURITY_AUDIT.md` - Full security review
- **Ollama Setup:** `OLLAMA_SETUP.md` - Detailed Ollama guide
- **Cloud APIs:** `CLOUD_API_SETUP.md` - Use OpenAI/Anthropic instead
- **Multi-Platform:** `MULTIPLATFORM.md` - Platform compatibility
- **Full Guide:** `PICOCLAW_GUIDE.md` - Complete documentation

---

## 🆘 Getting Help

1. **Check logs:** `./manage.sh logs`
2. **Check status:** `./manage.sh status`
3. **Test agent:** `./manage.sh test`
4. **Read docs:** See above for specific guides

---

## ✅ Daily Workflow

```bash
# Morning (services auto-start on boot, so skip this usually)
./manage.sh status

# Use the agent
picoclaw agent -m "What's on my schedule today?"

# Evening - services keep running (or stop to save resources)
./manage.sh stop
```

---

**That's everything you need!** For detailed setup, see other docs in this folder.

**Created by SWE100821** | Local, private, secure AI agents
