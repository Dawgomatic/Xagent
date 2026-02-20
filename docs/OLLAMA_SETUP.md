# PicoClaw + Ollama Setup (Fully Local & Private)
**Date:** 2026-02-16  
**Author:** SWE100821  
**Purpose:** Run picoclaw with local opensource models via Ollama

---

## Why Ollama? 🎯

✅ **100% Local & Private** - No data leaves your machine  
✅ **No API Costs** - Free unlimited usage  
✅ **No Internet Required** - Works offline after model download  
✅ **No Chinese Services** - Fully under your control  
✅ **Open Source Models** - Llama, Mistral, Qwen, etc.

---

## Prerequisites

1. **System Requirements:**
   - 16GB+ RAM (8GB minimum for smaller models)
   - 10GB+ disk space for models
   - Linux/macOS (Windows via WSL2)

2. **Software:**
   - Ollama installed
   - Go 1.21+ (for building picoclaw)

---

## Step 1: Install Ollama

### Linux
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Verify installation
ollama --version
```

### macOS
```bash
# Download from https://ollama.com/download
# Or use Homebrew:
brew install ollama

# Start Ollama service
ollama serve
```

### Check Ollama is Running
```bash
# Test the API endpoint
curl http://localhost:11434/api/tags
```

---

## Step 2: Download Models

### Recommended Models for Picoclaw

```bash
# Option 1: Llama 3.1 8B (Best balance)
ollama pull llama3.1:8b

# Option 2: Qwen 2.5 7B (Good for coding)
ollama pull qwen2.5:7b

# Option 3: Mistral 7B (Fast and efficient)
ollama pull mistral:7b

# Option 4: DeepSeek Coder (Best for coding tasks)
ollama pull deepseek-coder:6.7b

# Option 5: Llama 3.1 70B (Best quality, needs 48GB+ RAM)
ollama pull llama3.1:70b
```

### Test a Model
```bash
# Quick test
ollama run llama3.1:8b "What is 2+2?"

# Verify API works
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.1:8b",
  "prompt": "Hello, world!",
  "stream": false
}'
```

---

## Step 3: Install PicoClaw

```bash
cd /home/dawg/Desktop/AI_agents

# Clone repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Build dependencies
make deps

# Build binary
make build

# Test
./picoclaw version
```

---

## Step 4: Configure for Ollama

### Create Secure Local Config

```bash
# Create config directory
mkdir -p ~/.picoclaw

# Create config for Ollama
cat > ~/.picoclaw/config.json << 'EOF'
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "vllm",
      "model": "llama3.1:8b",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
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
EOF

# Secure the config
chmod 600 ~/.picoclaw/config.json
```

**Key Configuration:**
- `"provider": "vllm"` - Uses the VLLM provider (works with Ollama's OpenAI-compatible API)
- `"api_base": "http://localhost:11434/v1"` - Points to Ollama
- `"api_key": "not-needed"` - Ollama doesn't require authentication locally

---

## Step 5: Initialize Workspace

```bash
cd /home/dawg/Desktop/AI_agents/picoclaw

# Initialize picoclaw
./picoclaw onboard

# Verify status
./picoclaw status
```

Expected output:
```
🦞 picoclaw Status
Version: dev
...
Config: /home/dawg/.picoclaw/config.json ✓
Workspace: /home/dawg/.picoclaw/workspace ✓
Model: llama3.1:8b
...
```

---

## Step 6: Test the Setup

### Basic Test
```bash
cd /home/dawg/Desktop/AI_agents/picoclaw

# Simple query
./picoclaw agent -m "What is 2+2?"

# More complex query
./picoclaw agent -m "Write a hello world program in Python"
```

### Test Tool Use
```bash
# Test web search (DuckDuckGo)
./picoclaw agent -m "Search for latest news about AI"

# Test file operations (in sandbox)
./picoclaw agent -m "Create a file called test.txt with 'Hello World' in it"

# Verify sandbox (should fail - path outside workspace)
./picoclaw agent -m "Read the file /etc/passwd"
```

### Interactive Mode
```bash
# Start interactive session
./picoclaw agent

# Type your queries:
> What files are in my workspace?
> Create a todo list for today
> exit
```

---

## Step 7: Performance Optimization

### Ollama Configuration

```bash
# Edit Ollama systemd service (Linux)
sudo systemctl edit ollama

# Add these settings:
[Service]
Environment="OLLAMA_NUM_PARALLEL=2"
Environment="OLLAMA_MAX_LOADED_MODELS=1"
Environment="OLLAMA_FLASH_ATTENTION=1"

# Restart Ollama
sudo systemctl restart ollama
```

### PicoClaw Settings for Local Models

```json
{
  "agents": {
    "defaults": {
      "max_tokens": 4096,
      "temperature": 0.7,
      "max_tool_iterations": 15
    }
  }
}
```

**Notes:**
- Reduce `max_tokens` if experiencing slowness (4096 is good balance)
- Lower `temperature` (0.5-0.6) for more deterministic responses
- Reduce `max_tool_iterations` for faster responses

---

## Model Comparison

| Model | Size | RAM Needed | Speed | Quality | Best For |
|-------|------|------------|-------|---------|----------|
| **llama3.1:8b** | 4.7GB | 8GB | ⚡⚡⚡ | ★★★★ | General use (recommended) |
| **qwen2.5:7b** | 4.4GB | 8GB | ⚡⚡⚡⚡ | ★★★★ | Coding, multilingual |
| **mistral:7b** | 4.1GB | 8GB | ⚡⚡⚡⚡ | ★★★ | Fast responses |
| **deepseek-coder:6.7b** | 3.8GB | 8GB | ⚡⚡⚡⚡ | ★★★★★ | Programming tasks |
| **llama3.1:70b** | 40GB | 48GB | ⚡ | ★★★★★ | Best quality, slow |
| **phi3:3.8b** | 2.3GB | 4GB | ⚡⚡⚡⚡⚡ | ★★★ | Low-resource systems |

### Switch Models Anytime
```bash
# Download new model
ollama pull qwen2.5:7b

# Update config
sed -i 's/"model": "llama3.1:8b"/"model": "qwen2.5:7b"/' ~/.picoclaw/config.json

# Test
./picoclaw agent -m "Hello from new model!"
```

---

## Complete Secure Config (Ollama)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
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
    },
    "anthropic": {
      "api_key": "",
      "api_base": ""
    },
    "openai": {
      "api_key": "",
      "api_base": ""
    },
    "zhipu": {
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
      "enabled": false
    },
    "dingtalk": {
      "enabled": false
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": false,
        "api_key": ""
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
```

---

## Advanced: Multiple Models

### Configure Multiple Providers
```json
{
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "llama3.1:8b"
    }
  },
  "providers": {
    "vllm": {
      "api_base": "http://localhost:11434/v1",
      "api_key": "not-needed"
    }
  }
}
```

### Use Different Models per Task
```bash
# General queries: Llama 3.1
./picoclaw agent -m "What's the weather?"

# For coding: Switch to DeepSeek Coder
# Edit config to use deepseek-coder:6.7b, then:
./picoclaw agent -m "Write a Python function to sort a list"

# For speed: Switch to Mistral
# Edit config to use mistral:7b, then:
./picoclaw agent -m "Quick question: what is 2+2?"
```

---

## Troubleshooting

### Issue: "Failed to connect to API"
```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# If not running, start it:
ollama serve  # Run in separate terminal

# Or as service (Linux):
sudo systemctl start ollama
sudo systemctl enable ollama
```

### Issue: "Model not found"
```bash
# List available models
ollama list

# Pull the model you want
ollama pull llama3.1:8b
```

### Issue: "Out of memory"
```bash
# Check memory usage
free -h

# Use smaller model
ollama pull mistral:7b  # or phi3:3.8b

# Update config to use smaller model
sed -i 's/llama3.1:8b/mistral:7b/' ~/.picoclaw/config.json
```

### Issue: Slow responses
```bash
# 1. Use smaller model (mistral, phi3)
# 2. Reduce max_tokens in config
# 3. Enable GPU acceleration (if available)

# Check if using GPU:
ollama ps
```

### Issue: Workspace errors
```bash
# Verify sandbox is working
./picoclaw agent -m "List files in /tmp"
# Should fail with "path outside working dir"

# Check config
grep "restrict_to_workspace" ~/.picoclaw/config.json
# Should show: "restrict_to_workspace": true
```

---

## Security Benefits (Ollama vs Cloud)

| Feature | Ollama (Local) | Cloud APIs |
|---------|----------------|------------|
| **Data Privacy** | ✅ Never leaves machine | ⚠️ Sent to third party |
| **Internet Required** | ❌ Offline after download | ✅ Always |
| **API Costs** | ✅ Free | 💰 Pay per token |
| **Chinese Services** | ❌ None | ⚠️ Possible |
| **Audit Trail** | ✅ Local logs only | ⚠️ Server logs |
| **Backdoor Risk** | ✅ Zero | ⚠️ Possible |
| **Model Control** | ✅ Full control | ⚠️ Provider controlled |

---

## Monitoring Your Local Setup

### Check What's Running
```bash
# Create monitoring script
cat > ~/check_picoclaw_local.sh << 'EOF'
#!/bin/bash
# SWE100821: Monitor local picoclaw setup

echo "=== PicoClaw Local Setup Check ==="
date
echo ""

# Check Ollama
echo "1. Ollama Status:"
if pgrep -x ollama > /dev/null; then
    echo "   ✅ Ollama running"
    curl -s http://localhost:11434/api/tags | grep -o '"name":"[^"]*"' | head -3
else
    echo "   ❌ Ollama not running"
fi

echo ""
echo "2. PicoClaw Status:"
if pgrep -f picoclaw > /dev/null; then
    echo "   ✅ PicoClaw running"
else
    echo "   ⏸️  PicoClaw not running"
fi

echo ""
echo "3. Network Check (should only see localhost):"
netstat -tuln | grep -E '11434|18790'

echo ""
echo "4. Config Check:"
grep -E "restrict_to_workspace|api_base" ~/.picoclaw/config.json

echo ""
echo "=== All Local - No Cloud Services ==="
EOF

chmod +x ~/check_picoclaw_local.sh
~/check_picoclaw_local.sh
```

---

## Telegram Bot (Optional - Local Processing)

Even with Telegram, all processing stays local:

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

**What happens:**
1. Telegram message received → picoclaw
2. picoclaw sends to Ollama (localhost)
3. Response generated locally
4. Reply sent via Telegram

**Privacy:** Only your messages and responses go through Telegram. The AI processing is 100% local.

---

## Quick Commands Reference

```bash
# Check Ollama models
ollama list

# Pull new model
ollama pull llama3.1:8b

# Test Ollama directly
ollama run llama3.1:8b "test query"

# Start picoclaw (CLI)
cd /home/dawg/Desktop/AI_agents/picoclaw
./picoclaw agent

# Start picoclaw (gateway for Telegram)
./picoclaw gateway

# Check status
./picoclaw status

# Monitor both services
ps aux | grep -E "ollama|picoclaw"
```

---

## Recommended Setup for Different Use Cases

### 1. Privacy-Focused User
```json
{
  "model": "llama3.1:8b",
  "restrict_to_workspace": true,
  "channels": {}  // No external channels
}
```

### 2. Developer (Coding Assistant)
```json
{
  "model": "deepseek-coder:6.7b",
  "restrict_to_workspace": false,  // Need project access
  "max_tool_iterations": 20
}
```

### 3. Low-Resource System
```json
{
  "model": "phi3:3.8b",
  "max_tokens": 2048,
  "max_tool_iterations": 10
}
```

### 4. Best Quality (High-End System)
```json
{
  "model": "llama3.1:70b",
  "max_tokens": 8192,
  "temperature": 0.8
}
```

---

## Summary: Why This is Better

✅ **100% Private** - No data sent to any company (Chinese or Western)  
✅ **$0 Cost** - No API fees ever  
✅ **Offline Capable** - Works without internet  
✅ **Full Control** - You control the model and data  
✅ **No Backdoors** - Everything runs on your machine  
✅ **Open Source** - Both Ollama and models are open  
✅ **Faster (Local)** - No network latency  

---

## Next Steps

1. ✅ Install Ollama: `curl -fsSL https://ollama.com/install.sh | sh`
2. ✅ Download model: `ollama pull llama3.1:8b`
3. ✅ Clone picoclaw: `git clone https://github.com/sipeed/picoclaw.git`
4. ✅ Build: `cd picoclaw && make build`
5. ✅ Configure: Copy config above to `~/.picoclaw/config.json`
6. ✅ Test: `./picoclaw agent -m "Hello!"`

---

**Created by SWE100821** | 2026-02-16  
**For:** Fully local, private AI agent setup
