# Installation

## One-Command Install

```bash
./start.sh
```

This auto-detects your platform, installs all dependencies, builds Xagent, configures services, and enables auto-start on boot.

---

## What Gets Installed

| Component | Purpose |
|-----------|---------|
| Go 1.26.0 | Build toolchain |
| Python 3.x | Memory bridge, skill converter |
| Ollama | Local LLM inference server |
| Xagent binary | AI agent (built from source) |
| Systemd services | Auto-start on boot |

---

## Supported Platforms

| Platform | Ubuntu | Python | Architecture | Recommended Model |
|----------|--------|--------|--------------|-------------------|
| Jetson Xavier | 20.04 | 3.8 | arm64 | llama3.1:8b |
| Raspberry Pi 4 | 22.04 | 3.10 | arm64 | phi3:3.8b |
| Raspberry Pi 4 | 24.04 | 3.12 | arm64 | phi3:3.8b |
| Raspberry Pi 3 | 22.04 | 3.10 | armv6l | Gateway only (use cloud API) |
| x86_64 Desktop | Any | System | amd64 | llama3.1:8b or larger |

---

## Manual Install

If you prefer manual control over each step:

### 1. Prerequisites

```bash
sudo apt update && sudo apt install -y git build-essential curl
```

### 2. Install Go

```bash
wget https://go.dev/dl/go1.26.0.linux-$(dpkg --print-architecture).tar.gz
sudo tar -C /usr/local -xzf go1.26.0.linux-*.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 3. Install Ollama

```bash
curl -fsSL https://ollama.com/install.sh | sh
sudo systemctl enable ollama
sudo systemctl start ollama
ollama pull phi3:3.8b   # or llama3.1:8b for more capable hardware
```

### 4. Build Xagent

```bash
cd /path/to/xagent
make deps
make build
sudo ln -sf $(pwd)/build/xagent /usr/local/bin/xagent
```

### 5. Initialize

```bash
xagent onboard
```

This creates `~/.xagent/` with `config.json` and the workspace directory.

### 6. Configure

Edit `~/.xagent/config.json`:

```json
{
  "providers": {
    "vllm": {
      "api_base": "http://localhost:11434/v1",
      "api_key": "not-needed"
    }
  },
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "phi3:3.8b",
      "restrict_to_workspace": true,
      "max_tokens": 4096,
      "temperature": 0.7
    }
  }
}
```

Lock permissions:

```bash
chmod 600 ~/.xagent/config.json
```

### 7. Create Systemd Service (optional)

```bash
sudo tee /etc/systemd/system/xagent-gateway.service > /dev/null << EOF
[Unit]
Description=Xagent Gateway
After=network.target ollama.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$HOME
ExecStart=/usr/local/bin/xagent gateway
Restart=on-failure
RestartSec=10
ProtectSystem=strict
ReadWritePaths=$HOME/.xagent

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable xagent-gateway
sudo systemctl start xagent-gateway
```

---

## Verify

```bash
xagent status                    # Check agent status
xagent llm-check hw-detect      # Verify hardware detection
xagent agent -m "Hello"         # Test the agent
```

---

## Uninstall

```bash
sudo systemctl stop xagent-gateway
sudo systemctl disable xagent-gateway
sudo rm /etc/systemd/system/xagent-gateway.service
sudo rm /usr/local/bin/xagent
rm -rf ~/.xagent     # Remove all data
```
