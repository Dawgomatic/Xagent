# PicoClaw Secure Setup Guide
**Date:** 2026-02-16  
**Author:** SWE100821  
**Purpose:** Install and configure picoclaw with security-hardened settings

---

## Prerequisites

1. **Go 1.21+** (for building from source)
2. **API Key** from a Western provider:
   - [Anthropic](https://console.anthropic.com) (recommended) - Claude models
   - [OpenAI](https://platform.openai.com) - GPT models
   - [OpenRouter](https://openrouter.ai/keys) - Multiple models via one API

---

## Step 1: Install PicoClaw

### Option A: Build from Source (Recommended for Security)

```bash
# Clone the repository
cd /home/dawg/Desktop/AI_agents
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Verify the source code (optional but recommended)
# Review go.mod for dependencies
cat go.mod

# Build dependencies
make deps

# Build the binary
make build

# Install to system (optional)
sudo make install

# Or use the local binary
./picoclaw version
```

### Option B: Docker (Maximum Isolation)

```bash
cd /home/dawg/Desktop/AI_agents/picoclaw

# Review the Dockerfile first
cat Dockerfile

# Build Docker image
docker build -t picoclaw:secure .

# We'll configure it in Step 2
```

---

## Step 2: Secure Configuration

### Create Config Directory
```bash
mkdir -p ~/.picoclaw
```

### Create Secure config.json

```bash
# Create the secure configuration
cat > ~/.picoclaw/config.json << 'EOF'
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "anthropic",
      "model": "claude-sonnet-4.5",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "anthropic": {
      "api_key": "YOUR_ANTHROPIC_API_KEY_HERE",
      "api_base": "https://api.anthropic.com/v1"
    },
    "openai": {
      "api_key": "",
      "api_base": "https://api.openai.com/v1"
    },
    "openrouter": {
      "api_key": "",
      "api_base": "https://openrouter.ai/api/v1"
    },
    "groq": {
      "api_key": "",
      "api_base": "https://api.groq.com/openai/v1"
    },
    "zhipu": {
      "api_key": "",
      "api_base": ""
    },
    "moonshot": {
      "api_key": "",
      "api_base": ""
    },
    "deepseek": {
      "api_key": "",
      "api_base": ""
    }
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "",
      "allow_from": []
    },
    "discord": {
      "enabled": false,
      "token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "allow_from": []
    },
    "feishu": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "dingtalk": {
      "enabled": false,
      "client_id": "",
      "client_secret": "",
      "allow_from": []
    },
    "whatsapp": {
      "enabled": false,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    },
    "slack": {
      "enabled": false,
      "bot_token": "",
      "app_token": "",
      "allow_from": []
    },
    "line": {
      "enabled": false,
      "channel_secret": "",
      "channel_access_token": "",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    },
    "maixcam": {
      "enabled": false,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    },
    "onebot": {
      "enabled": false,
      "ws_url": "ws://127.0.0.1:3001",
      "access_token": "",
      "reconnect_interval": 5,
      "group_trigger_prefix": [],
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": "",
        "max_results": 5
      },
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
EOF
```

### Configure Your API Key

```bash
# Get your Anthropic API key from https://console.anthropic.com

# Edit the config file
nano ~/.picoclaw/config.json

# Replace YOUR_ANTHROPIC_API_KEY_HERE with your actual key
# Or use sed:
sed -i 's/YOUR_ANTHROPIC_API_KEY_HERE/sk-ant-your-actual-key-here/' ~/.picoclaw/config.json
```

---

## Step 3: Initialize Workspace

```bash
# Initialize picoclaw (creates workspace with default files)
picoclaw onboard

# Or if using local binary:
cd /home/dawg/Desktop/AI_agents/picoclaw
./picoclaw onboard
```

This creates:
```
~/.picoclaw/workspace/
├── sessions/          # Conversation history
├── memory/            # Long-term memory
├── state/             # Persistent state
├── cron/              # Scheduled jobs
├── skills/            # Custom skills
├── AGENTS.md          # Agent behavior guide
├── IDENTITY.md        # Agent identity
├── SOUL.md            # Agent personality
├── TOOLS.md           # Tool descriptions
└── USER.md            # User preferences
```

---

## Step 4: Verify Security Settings

### Check Configuration
```bash
# Verify configuration loaded correctly
picoclaw status

# Expected output should show:
# - Anthropic API: ✓
# - Chinese APIs (Zhipu, Moonshot): not set
# - Workspace: ~/.picoclaw/workspace ✓
```

### Test Sandbox Restrictions
```bash
# Create a test file outside workspace
echo "test" > /tmp/test_file.txt

# Try to read it (should fail if restrictions are working)
picoclaw agent -m "Read the file /tmp/test_file.txt"

# Expected: Error message about path outside workspace
```

### Verify No Chinese Endpoints
```bash
# Monitor network connections while running
# Terminal 1: Start picoclaw
picoclaw agent

# Terminal 2: Monitor connections
sudo netstat -tunapl | grep picoclaw

# Should only see connections to:
# - api.anthropic.com (or your chosen provider)
# - html.duckduckgo.com (if web search used)
```

---

## Step 5: Docker Setup (Optional - Maximum Isolation)

### Create Docker Compose with Secure Config

```bash
cd /home/dawg/Desktop/AI_agents/picoclaw

# Create docker-compose.secure.yml
cat > docker-compose.secure.yml << 'EOF'
version: '3.8'

services:
  picoclaw-agent:
    build: .
    container_name: picoclaw-secure
    volumes:
      - ./config/config.json:/root/.picoclaw/config.json:ro
      - picoclaw-workspace:/root/.picoclaw/workspace
    environment:
      - PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=true
    networks:
      - picoclaw-net
    restart: unless-stopped
    # Security options
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only if needed for gateway
    read_only: false  # Agent needs to write to workspace
    tmpfs:
      - /tmp
    user: "1000:1000"  # Run as non-root

volumes:
  picoclaw-workspace:

networks:
  picoclaw-net:
    driver: bridge
EOF
```

### Prepare Docker Config
```bash
# Copy your config to the docker directory
mkdir -p config
cp ~/.picoclaw/config.json config/config.json

# Edit to use Docker-friendly paths
sed -i 's|~/.picoclaw/workspace|/root/.picoclaw/workspace|g' config/config.json
```

### Run in Docker
```bash
# Build and start
docker-compose -f docker-compose.secure.yml up -d

# View logs
docker-compose -f docker-compose.secure.yml logs -f

# Run commands inside container
docker exec -it picoclaw-secure picoclaw agent -m "Hello, test message"

# Stop
docker-compose -f docker-compose.secure.yml down
```

---

## Step 6: Enable Safe Channels (Optional)

### Telegram (Recommended - Most Secure)

1. **Create Telegram Bot:**
   - Open Telegram, search `@BotFather`
   - Send `/newbot`, follow prompts
   - Copy the token

2. **Get Your User ID:**
   - Search `@userinfobot` on Telegram
   - Send any message to get your ID

3. **Update Config:**
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

4. **Start Gateway:**
```bash
picoclaw gateway
```

### Discord (Also Safe)

1. **Create Discord Bot:**
   - Go to https://discord.com/developers/applications
   - Create application → Bot → Add Bot
   - Copy bot token
   - Enable MESSAGE CONTENT INTENT

2. **Get Your User ID:**
   - Discord Settings → Advanced → Developer Mode
   - Right-click your avatar → Copy User ID

3. **Update Config:**
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

4. **Invite Bot to Server:**
   - OAuth2 → URL Generator
   - Scopes: `bot`
   - Permissions: `Send Messages`, `Read Message History`
   - Open generated URL, add to server

---

## Step 7: Monitoring and Maintenance

### Monitor Logs
```bash
# If using system install
journalctl -u picoclaw -f

# If running manually
picoclaw gateway > ~/picoclaw.log 2>&1 &
tail -f ~/picoclaw.log
```

### Check for Unexpected Connections
```bash
# Create monitoring script
cat > ~/monitor_picoclaw.sh << 'EOF'
#!/bin/bash
# SWE100821: Monitor picoclaw network connections

echo "=== PicoClaw Network Monitor ==="
echo "Checking for unexpected connections..."
echo ""

# Get picoclaw PID
PID=$(pgrep -f picoclaw)

if [ -z "$PID" ]; then
    echo "⚠️  PicoClaw not running"
    exit 1
fi

# Check connections
echo "Active connections:"
sudo netstat -tunapl | grep $PID | while read line; do
    echo "$line" | grep -q "api.anthropic.com" && echo "✅ $line"
    echo "$line" | grep -q "api.openai.com" && echo "✅ $line"
    echo "$line" | grep -q "openrouter.ai" && echo "✅ $line"
    echo "$line" | grep -q "duckduckgo.com" && echo "✅ $line"
    echo "$line" | grep -q "bigmodel.cn" && echo "⚠️  CHINESE: $line"
    echo "$line" | grep -q "moonshot.cn" && echo "⚠️  CHINESE: $line"
    echo "$line" | grep -q "deepseek.com" && echo "⚠️  CHINESE: $line"
done

echo ""
echo "Monitor complete at $(date)"
EOF

chmod +x ~/monitor_picoclaw.sh

# Run it
~/monitor_picoclaw.sh
```

### Regular Security Checks
```bash
# Create weekly security audit script
cat > ~/audit_picoclaw.sh << 'EOF'
#!/bin/bash
# SWE100821: Weekly security audit

echo "=== PicoClaw Security Audit ==="
date

# Check config permissions
echo ""
echo "1. Config File Permissions:"
ls -la ~/.picoclaw/config.json

# Check for Chinese endpoints in config
echo ""
echo "2. Chinese Endpoints Check:"
grep -i "\.cn\|zhipu\|moonshot\|deepseek" ~/.picoclaw/config.json && \
    echo "⚠️  Chinese endpoints found in config" || \
    echo "✅ No Chinese endpoints configured"

# Check workspace restrictions
echo ""
echo "3. Workspace Restrictions:"
grep "restrict_to_workspace" ~/.picoclaw/config.json

# Check enabled channels
echo ""
echo "4. Enabled Channels:"
grep -A2 "\"enabled\": true" ~/.picoclaw/config.json

echo ""
echo "Audit complete"
EOF

chmod +x ~/audit_picoclaw.sh

# Run weekly via cron
(crontab -l 2>/dev/null; echo "0 9 * * 1 ~/audit_picoclaw.sh > ~/picoclaw_audit.log 2>&1") | crontab -
```

---

## Step 8: Usage Examples

### CLI Mode (Most Secure)
```bash
# One-off query
picoclaw agent -m "What is the weather today?"

# Interactive mode
picoclaw agent

# With custom session
picoclaw agent -s "project-session" -m "Continue our project discussion"
```

### Gateway Mode (For Chat Apps)
```bash
# Start gateway (Telegram/Discord)
picoclaw gateway

# With debug logging
picoclaw gateway --debug
```

### Safe Web Search
```bash
# DuckDuckGo is enabled by default (no API key needed)
picoclaw agent -m "Search for latest AI news"

# To use Brave Search (optional, requires API key):
# 1. Get key from https://brave.com/search/api (free tier)
# 2. Update config:
#    "brave": {
#      "enabled": true,
#      "api_key": "YOUR_BRAVE_KEY"
#    }
```

---

## Step 9: Skills Management (Careful!)

### List Built-in Skills
```bash
picoclaw skills list-builtin
```

### Install Only Trusted Skills
```bash
# ⚠️ ONLY install from verified sources
# Review the skill code first on GitHub

# Example: Install from official picoclaw skills repo
picoclaw skills install sipeed/picoclaw-skills/weather

# Or install built-in skills (safer)
picoclaw skills install-builtin
```

### Review Skill Before Using
```bash
# Show skill details
picoclaw skills show weather

# Check skill source files
cat ~/.picoclaw/workspace/skills/weather/SKILL.md
```

---

## Security Checklist

### ✅ Initial Setup
- [ ] Built from source (or reviewed Dockerfile)
- [ ] Using Western AI provider (Anthropic/OpenAI/OpenRouter)
- [ ] `restrict_to_workspace: true` set
- [ ] Chinese API keys left blank/empty
- [ ] Chinese channels disabled (QQ, Feishu, DingTalk)
- [ ] Config file permissions: `chmod 600 ~/.picoclaw/config.json`
- [ ] Gateway bound to localhost: `"host": "127.0.0.1"`

### ✅ Runtime Security
- [ ] Monitor script installed and running
- [ ] No unexpected network connections detected
- [ ] Only trusted skills installed
- [ ] Logs being collected and reviewed
- [ ] Docker isolation (if using)

### ✅ Ongoing Maintenance
- [ ] Weekly security audit scheduled
- [ ] Regular config backups
- [ ] Update checks: `git pull` in picoclaw directory
- [ ] Review changelogs before updating

---

## Troubleshooting

### Error: "API key not configured"
```bash
# Check config is loaded
picoclaw status

# Verify API key in config
grep "api_key" ~/.picoclaw/config.json | head -5
```

### Error: "Path outside working dir"
```bash
# This is expected! Workspace restrictions are working.
# All file operations must be inside ~/.picoclaw/workspace/
```

### Connection Timeout
```bash
# Check internet connectivity
curl -I https://api.anthropic.com

# Check if firewall blocking
sudo iptables -L -n | grep 443
```

### Docker Issues
```bash
# Check container logs
docker logs picoclaw-secure

# Shell into container
docker exec -it picoclaw-secure /bin/sh

# Rebuild clean
docker-compose -f docker-compose.secure.yml down -v
docker-compose -f docker-compose.secure.yml build --no-cache
```

---

## Uninstall (if needed)

```bash
# Stop any running instances
pkill picoclaw

# Remove binary
sudo rm /usr/local/bin/picoclaw

# Remove config and data (⚠️ deletes everything)
rm -rf ~/.picoclaw

# Remove source
rm -rf /home/dawg/Desktop/AI_agents/picoclaw

# Remove Docker
docker-compose -f docker-compose.secure.yml down -v
docker rmi picoclaw:secure
```

---

## References
- Security Audit: `/home/dawg/Desktop/AI_agents/picoclaw_security_audit.md`
- Official Docs: https://github.com/sipeed/picoclaw
- Anthropic API: https://console.anthropic.com
- OpenAI API: https://platform.openai.com
- OpenRouter: https://openrouter.ai

---

**Setup Guide Created:** 2026-02-16  
**Author:** SWE100821
