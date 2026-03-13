#!/bin/bash
# Unified Installation & Service Setup Script
# Author: SWE100821
# Date: 2026-02-17
# Purpose: Install everything, create services, enable auto-start on boot

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration - SWE100821: Auto-detect install dir for portability
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
USER=$(whoami)
HOME_DIR=$HOME

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }

# Banner
show_banner() {
    clear
    echo "=============================================="
    echo "   AI Agents Master Installation Script"
    echo "   By SWE100821"
    echo "=============================================="
    echo ""
}

# Detect platform and OS
detect_system() {
    log_info "Detecting system configuration..."
    
    # OS Detection
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_VERSION=$VERSION_ID
        OS_CODENAME=$VERSION_CODENAME
    fi
    
    # Platform Detection
    if [ -f /etc/nv_tegra_release ]; then
        PLATFORM="xavier"
    elif [ -f /proc/device-tree/model ]; then
        MODEL=$(cat /proc/device-tree/model)
        if [[ $MODEL == *"Raspberry Pi 4"* ]]; then
            PLATFORM="rpi4"
        elif [[ $MODEL == *"Raspberry Pi 3"* ]]; then
            PLATFORM="rpi3"
        else
            PLATFORM="rpi"
        fi
    else
        PLATFORM="x86_64"
    fi
    
    # Python version selection
    case $OS_VERSION in
        20.04) PYTHON_VERSION="3.8" ;;
        22.04) PYTHON_VERSION="3.10" ;;
        24.04) PYTHON_VERSION="3.12" ;;
        *) PYTHON_VERSION="3" ;;
    esac
    
    # Ollama support
    case $PLATFORM in
        xavier|x86_64|rpi4) OLLAMA_SUPPORTED=true ;;
        rpi3) OLLAMA_SUPPORTED=false ;;
        *) OLLAMA_SUPPORTED=true ;;
    esac
    
    log_success "Detected: $PLATFORM (Ubuntu $OS_VERSION, Python $PYTHON_VERSION)"
}

# Install system dependencies
install_dependencies() {
    log_info "Installing system dependencies..."
    
    sudo apt-get update -qq
    
    PACKAGES="build-essential git curl wget vim nano"
    PACKAGES="$PACKAGES libssl-dev libffi-dev pkg-config"
    PACKAGES="$PACKAGES python${PYTHON_VERSION} python${PYTHON_VERSION}-dev"
    PACKAGES="$PACKAGES python${PYTHON_VERSION}-venv python3-pip"
    
    sudo apt-get install -y $PACKAGES > /dev/null 2>&1
    
    log_success "System dependencies installed"
}

# Install Go
install_go() {
    log_info "Installing Go..."
    
    if command -v go &> /dev/null; then
        log_success "Go already installed"
        return
    fi
    
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) GO_ARCH="amd64" ;;
        aarch64|arm64) GO_ARCH="arm64" ;;
        *) GO_ARCH="arm64" ;;
    esac
    
    # SWE100821: Must match go.mod (>=1.25.7). Dockerfile uses 1.26.0.
    GO_VERSION="1.26.0"
    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    wget -q --show-progress "https://go.dev/dl/${GO_TAR}" -O /tmp/${GO_TAR}
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/${GO_TAR}
    rm /tmp/${GO_TAR}
    
    # Add to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    
    log_success "Go installed: $(/usr/local/go/bin/go version)"
}

# Install Ollama
install_ollama() {
    if [ "$OLLAMA_SUPPORTED" = false ]; then
        log_warning "Skipping Ollama (not recommended for this platform)"
        return
    fi
    
    log_info "Installing Ollama..."
    
    if command -v ollama &> /dev/null; then
        log_success "Ollama already installed"
        sudo systemctl enable ollama > /dev/null 2>&1 || true
        sudo systemctl start ollama > /dev/null 2>&1 || true
        return
    fi
    
    curl -fsSL https://ollama.com/install.sh | sh > /dev/null 2>&1
    
    # Ensure service is enabled
    sudo systemctl enable ollama > /dev/null 2>&1
    sudo systemctl start ollama > /dev/null 2>&1
    
    sleep 3
    
    log_success "Ollama installed and enabled"
}

# SWE100821: Hardware-aware model selection — correlates RAM, GPU VRAM, and platform
detect_compute_tier() {
    RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    RAM_MB=$((RAM_KB / 1024))
    
    GPU_VRAM_MB=0
    GPU_NAME="none"
    if command -v nvidia-smi &> /dev/null; then
        GPU_VRAM_MB=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 | tr -d ' ' || echo 0)
        GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 || echo "unknown")
    elif [ -f /etc/nv_tegra_release ]; then
        GPU_NAME="Tegra"
        GPU_VRAM_MB=$RAM_MB  # shared memory on Jetson
    fi
    
    CPU_CORES=$(nproc 2>/dev/null || echo 1)
    
    # Tier classification (mirrors pkg/hwprofile/hwprofile.go classify())
    if [ "$GPU_VRAM_MB" -gt 0 ] && [ "$GPU_NAME" != "none" ]; then
        COMPUTE_TIER="gpu"
    elif [ "$RAM_MB" -ge 32000 ]; then
        COMPUTE_TIER="high"
    elif [ "$RAM_MB" -ge 8000 ]; then
        COMPUTE_TIER="mid"
    elif [ "$RAM_MB" -ge 2000 ]; then
        COMPUTE_TIER="low"
    else
        COMPUTE_TIER="minimal"
    fi
    
    log_success "Hardware: ${CPU_CORES} cores, ${RAM_MB}MB RAM, GPU=${GPU_NAME} (${GPU_VRAM_MB}MB VRAM)"
    log_success "Compute tier: $COMPUTE_TIER"
}

# SWE100821: Select Ollama model based on compute tier and VRAM
select_model() {
    case $COMPUTE_TIER in
        gpu)
            if [ "$GPU_VRAM_MB" -ge 48000 ]; then
                MODEL="llama3.1:70b"
            elif [ "$GPU_VRAM_MB" -ge 16000 ]; then
                MODEL="llama3.1:8b"
            elif [ "$GPU_VRAM_MB" -ge 8000 ]; then
                MODEL="llama3.1:8b"
            elif [ "$GPU_VRAM_MB" -ge 4000 ]; then
                MODEL="phi3:3.8b"
            else
                MODEL="tinyllama:1.1b"
            fi
            ;;
        high)   MODEL="llama3.1:8b" ;;
        mid)    MODEL="llama3.1:8b" ;;
        low)    MODEL="phi3:3.8b" ;;
        *)      MODEL="tinyllama:1.1b" ;;
    esac
}

# SWE100821: Select tuning parameters based on compute tier
select_tuning() {
    case $COMPUTE_TIER in
        gpu)
            MAX_TOKENS=8192; TEMPERATURE="0.7"; MAX_ITER=25; MAX_SUB=5; MSG_TIMEOUT=300
            ;;
        high)
            MAX_TOKENS=8192; TEMPERATURE="0.7"; MAX_ITER=20; MAX_SUB=4; MSG_TIMEOUT=300
            ;;
        mid)
            MAX_TOKENS=4096; TEMPERATURE="0.7"; MAX_ITER=15; MAX_SUB=3; MSG_TIMEOUT=300
            ;;
        low)
            MAX_TOKENS=2048; TEMPERATURE="0.5"; MAX_ITER=10; MAX_SUB=1; MSG_TIMEOUT=600
            ;;
        *)
            MAX_TOKENS=1024; TEMPERATURE="0.3"; MAX_ITER=5; MAX_SUB=1; MSG_TIMEOUT=900
            ;;
    esac
}

# Download appropriate model
download_model() {
    if [ "$OLLAMA_SUPPORTED" = false ]; then
        return
    fi
    
    log_info "Downloading AI model..."
    
    # SWE100821: Use hardware-correlated model selection
    detect_compute_tier
    select_model
    select_tuning
    
    log_info "Selected model: $MODEL (tier=$COMPUTE_TIER, max_tokens=$MAX_TOKENS)"
    
    ollama pull $MODEL > /dev/null 2>&1 || {
        log_warning "Failed to download model, will retry later"
        return
    }
    
    # Persist detected hardware config for configure_xagent
    cat > ~/.ollama_model << HWEOF
MODEL=$MODEL
COMPUTE_TIER=$COMPUTE_TIER
MAX_TOKENS=$MAX_TOKENS
TEMPERATURE=$TEMPERATURE
MAX_ITER=$MAX_ITER
MAX_SUB=$MAX_SUB
MSG_TIMEOUT=$MSG_TIMEOUT
RAM_MB=$RAM_MB
GPU_VRAM_MB=$GPU_VRAM_MB
CPU_CORES=$CPU_CORES
HWEOF
    
    log_success "Model downloaded: $MODEL"
}

# Install Xagent
install_xagent() {
    log_info "Building Xagent..."
    
    cd "$INSTALL_DIR"
    
    make deps > /dev/null 2>&1
    make build > /dev/null 2>&1
    
    sudo ln -sf "$INSTALL_DIR/build/xagent" /usr/local/bin/xagent
    
    log_success "Xagent built and installed"
}

# Configure Xagent - SWE100821: Hardware-correlated config auto-generation
configure_xagent() {
    log_info "Configuring Xagent..."
    
    mkdir -p ~/.xagent
    
    # SWE100821: Load hardware-detected settings from download_model phase
    if [ -f ~/.ollama_model ]; then
        source ~/.ollama_model
    else
        MODEL="llama3.1:8b"
        MAX_TOKENS=4096
        TEMPERATURE="0.7"
        MAX_ITER=15
    fi
    
    log_info "Config: model=$MODEL max_tokens=$MAX_TOKENS temp=$TEMPERATURE iterations=$MAX_ITER"
    
    cat > ~/.xagent/config.json << EOF
{
  "agents": {
    "defaults": {
      "workspace": "~/.xagent/workspace",
      "restrict_to_workspace": true,
      "provider": "vllm",
      "model": "$MODEL",
      "max_tokens": $MAX_TOKENS,
      "temperature": $TEMPERATURE,
      "max_tool_iterations": $MAX_ITER
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
    "enabled": false
  },
  "devices": {
    "enabled": false
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  }
}
EOF
    
    chmod 600 ~/.xagent/config.json
    xagent onboard > /dev/null 2>&1 || true
    
    log_success "Xagent configured (tier=$COMPUTE_TIER)"
}

# Create systemd service for Xagent Gateway
create_xagent_service() {
    log_info "Creating Xagent systemd service..."
    
    # SWE100821: Service file for auto-start
    sudo tee /etc/systemd/system/xagent-gateway.service > /dev/null << EOF
[Unit]
Description=Xagent AI Agent Gateway
Documentation=https://github.com/sipeed/xagent
After=network.target ollama.service
Wants=ollama.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$HOME_DIR
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
Environment="HOME=$HOME_DIR"
ExecStart=/usr/local/bin/xagent gateway
# SWE100821: Health check - systemd can verify gateway is alive
ExecStartPost=/bin/sh -c 'sleep 2 && curl -sf http://127.0.0.1:18791/healthz || exit 1'
Restart=on-failure
RestartSec=10
# SWE100821: Use journald for log rotation, compression, and journalctl support
StandardOutput=journal
StandardError=journal
SyslogIdentifier=xagent-gateway

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=$HOME_DIR/.xagent $HOME_DIR/.config/xagent

[Install]
WantedBy=multi-user.target
EOF
    
    # SWE100821: Memory bridge service (optional — requires Qdrant + pip deps)
    if [ -f "$INSTALL_DIR/memory_bridge.py" ]; then
        sudo tee /etc/systemd/system/memory-bridge.service > /dev/null << EOF
[Unit]
Description=Xagent Memory Bridge - SWE100821
After=network.target ollama.service
Wants=ollama.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$INSTALL_DIR
Environment="HOME=$HOME_DIR"
ExecStart=/usr/bin/python3 $INSTALL_DIR/memory_bridge.py watch
Restart=on-failure
RestartSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=memory-bridge

[Install]
WantedBy=multi-user.target
EOF
        log_success "Memory bridge service created (optional — enable with: sudo systemctl enable memory-bridge)"
    fi

    sudo systemctl daemon-reload
    
    log_success "Xagent service created"
}

# Enable and start services
enable_services() {
    log_info "Enabling services to start on boot..."
    
    # Enable Ollama
    if [ "$OLLAMA_SUPPORTED" = true ]; then
        sudo systemctl enable ollama > /dev/null 2>&1
        if ! systemctl is-active --quiet ollama; then
            sudo systemctl start ollama
        fi
        log_success "Ollama service enabled"
    fi
    
    # Enable Xagent (but don't start yet - user may want to configure first)
    sudo systemctl enable xagent-gateway > /dev/null 2>&1
    log_success "Xagent gateway service enabled (not started yet)"
}

# Block outbound connections to known Chinese cloud/service domains
setup_network_blocklist() {
    log_info "Setting up network security blocklist..."

    BLOCKLIST_DOMAINS=(
        # Chinese AI model providers
        "api.deepseek.com"
        "api.moonshot.cn"
        "open.bigmodel.cn"
        "dashscope.aliyuncs.com"
        "router.shengsuanyun.com"
        "ark.cn-beijing.volces.com"
        "console.xfyun.cn"
        # Chinese messaging platforms
        "api.weixin.qq.com"
        "qyapi.weixin.qq.com"
        "oapi.dingtalk.com"
        "open.dingtalk.com"
        "open.feishu.cn"
        # Chinese cloud providers
        "aliyuncs.com"
        "cloud.tencent.com"
        "mirrors.tencentyun.com"
        "bce.baidu.com"
        "huaweicloud.com"
        # Chinese social media
        "xiaohongshu.com"
        "weibo.com"
        "douyin.com"
        "juejin.cn"
        "tieba.baidu.com"
    )

    HOSTS_MARKER="# === Xagent Chinese Service Blocklist ==="
    HOSTS_FILE="/etc/hosts"

    if grep -q "$HOSTS_MARKER" "$HOSTS_FILE" 2>/dev/null; then
        log_info "Blocklist already present in /etc/hosts, skipping"
    else
        {
            echo ""
            echo "$HOSTS_MARKER"
            echo "# Blocks outbound connections to Chinese cloud services for security."
            echo "# Remove this section if you need access to these services."
            for domain in "${BLOCKLIST_DOMAINS[@]}"; do
                echo "0.0.0.0 $domain"
            done
            echo "# === End Xagent Blocklist ==="
        } | sudo tee -a "$HOSTS_FILE" > /dev/null
        log_success "Added ${#BLOCKLIST_DOMAINS[@]} Chinese domains to /etc/hosts blocklist"
    fi

    # Optional: iptables rules for broader .cn TLD blocking (DNS-level)
    if command -v iptables &> /dev/null; then
        # Block DNS queries that resolve to known Chinese cloud IP ranges
        # These are major Chinese cloud provider CIDR blocks
        CN_CIDRS=(
            "47.88.0.0/14"      # Alibaba Cloud International
            "47.92.0.0/14"      # Alibaba Cloud China
            "120.24.0.0/14"     # Alibaba Cloud
            "139.196.0.0/16"    # Alibaba Cloud
            "101.132.0.0/16"    # Alibaba Cloud
            "106.14.0.0/15"     # Alibaba Cloud
            "112.124.0.0/14"    # Alibaba Cloud
            "123.56.0.0/14"     # Alibaba Cloud
            "182.92.0.0/16"     # Alibaba Cloud
            "203.107.0.0/16"    # Alibaba Cloud DNS
            "49.51.0.0/16"      # Tencent Cloud
            "111.230.0.0/15"    # Tencent Cloud
            "119.29.0.0/16"     # Tencent Cloud
            "129.211.0.0/16"    # Tencent Cloud
            "140.143.0.0/16"    # Tencent Cloud
            "148.70.0.0/16"     # Tencent Cloud
            "159.75.0.0/16"     # Tencent Cloud
            "180.76.0.0/16"     # Baidu Cloud
            "106.12.0.0/15"     # Baidu Cloud
            "110.242.68.0/22"   # Baidu
            "114.116.0.0/16"    # Huawei Cloud
            "124.70.0.0/16"     # Huawei Cloud
            "139.9.0.0/16"      # Huawei Cloud
        )

        IPTABLES_CHAIN="XAGENT_BLOCK"
        if ! sudo iptables -L "$IPTABLES_CHAIN" -n &>/dev/null; then
            sudo iptables -N "$IPTABLES_CHAIN" 2>/dev/null
            for cidr in "${CN_CIDRS[@]}"; do
                sudo iptables -A "$IPTABLES_CHAIN" -d "$cidr" -j DROP 2>/dev/null
            done
            # Insert into OUTPUT chain so the device can't talk to these IPs
            sudo iptables -I OUTPUT -j "$IPTABLES_CHAIN" 2>/dev/null
            log_success "Added iptables rules blocking ${#CN_CIDRS[@]} Chinese cloud CIDR ranges"
        else
            log_info "iptables blocklist chain already exists, skipping"
        fi

        # Persist iptables rules across reboots
        if command -v netfilter-persistent &> /dev/null; then
            sudo netfilter-persistent save 2>/dev/null
        elif command -v iptables-save &> /dev/null; then
            sudo iptables-save | sudo tee /etc/iptables/rules.v4 > /dev/null 2>&1
        fi
    else
        log_warn "iptables not available -- /etc/hosts blocklist only (DNS-based)"
    fi
}

# Create management script
create_management_script() {
    log_info "Creating management script..."
    
    # SWE100821: Easy management script
    cat > "$INSTALL_DIR/manage.sh" << 'EOFMANAGE'
#!/bin/bash
# AI Agents Management Script - SWE100821

case "$1" in
    start)
        echo "Starting services..."
        sudo systemctl start ollama 2>/dev/null && echo "✓ Ollama started"
        sudo systemctl start xagent-gateway && echo "✓ Xagent started"
        ;;
    stop)
        echo "Stopping services..."
        sudo systemctl stop xagent-gateway && echo "✓ Xagent stopped"
        sudo systemctl stop ollama 2>/dev/null && echo "✓ Ollama stopped"
        ;;
    restart)
        echo "Restarting services..."
        sudo systemctl restart ollama 2>/dev/null && echo "✓ Ollama restarted"
        sudo systemctl restart xagent-gateway && echo "✓ Xagent restarted"
        ;;
    status)
        echo "=== Service Status ==="
        systemctl status ollama 2>/dev/null | grep -E "Active:|Main PID:" || echo "Ollama: not installed"
        echo ""
        systemctl status xagent-gateway | grep -E "Active:|Main PID:"
        echo ""
        echo "=== Xagent Status ==="
        xagent status
        ;;
    logs)
        echo "=== Recent Logs ==="
        echo "--- Xagent Gateway ---"
        # SWE100821: Use journalctl now that we log to journald
        sudo journalctl -u xagent-gateway -n 30 --no-pager 2>/dev/null || echo "No Xagent logs"
        echo ""
        echo "--- Ollama ---"
        sudo journalctl -u ollama -n 20 --no-pager 2>/dev/null || echo "No Ollama logs"
        ;;
    enable)
        echo "Enabling auto-start on boot..."
        sudo systemctl enable ollama 2>/dev/null && echo "✓ Ollama enabled"
        sudo systemctl enable xagent-gateway && echo "✓ Xagent enabled"
        ;;
    disable)
        echo "Disabling auto-start..."
        sudo systemctl disable xagent-gateway && echo "✓ Xagent disabled"
        sudo systemctl disable ollama 2>/dev/null && echo "✓ Ollama disabled"
        ;;
    test)
        echo "Testing agent..."
        xagent agent -m "Quick test: what is 2+2?"
        ;;
    # SWE100821: Health check command for quick liveness verification
    health)
        echo "=== Health Check ==="
        curl -sf http://127.0.0.1:18791/healthz && echo "" || echo " Gateway health check FAILED"
        curl -sf http://127.0.0.1:18791/readyz  && echo "" || echo " Gateway readiness check FAILED"
        ;;
    upgrade)
        shift
        xagent upgrade "\$@"
        ;;
    skills)
        SCRIPT_DIR="\$(cd "\$(dirname "\$0")" && pwd)"
        CONVERTER="\$SCRIPT_DIR/skill_converter.py"
        if [ ! -f "\$CONVERTER" ]; then
            echo "Error: skill_converter.py not found"
            exit 1
        fi
        shift
        python3 "\$CONVERTER" "\$@"
        ;;
    *)
        echo "AI Agents Management Script"
        echo ""
        echo "Usage: \$0 {start|stop|restart|status|logs|enable|disable|test|health|upgrade|skills}"
        echo ""
        echo "Commands:"
        echo "  start    - Start all services"
        echo "  stop     - Stop all services"
        echo "  restart  - Restart all services"
        echo "  status   - Show service status"
        echo "  logs     - Show recent logs"
        echo "  enable   - Enable auto-start on boot"
        echo "  disable  - Disable auto-start on boot"
        echo "  test     - Test the agent"
        echo "  health   - Run health check"
        echo "  upgrade  - Self-upgrade Xagent (--check, --model, --all)"
        echo "  skills   - Manage OpenClaw skills (list, search, install, info)"
        echo ""
        echo "Skills Commands:"
        echo "  skills list                       - List available OpenClaw skills"
        echo "  skills search <keyword>           - Search skills by keyword"
        echo "  skills search <keyword> --deps    - Search with dependency check"
        echo "  skills info <owner>/<slug>        - Show skill details + dependencies"
        echo "  skills check <owner>/<slug>       - Check deps without installing"
        echo "  skills check-installed            - Check deps for installed skills"
        echo "  skills ready [keyword]            - List skills ready on this device"
        echo "  skills install <owner>/<slug>     - Install a skill"
        echo "  skills install-curated            - Install curated skill set"
        echo "  skills bulk --filter <keyword>    - Bulk install matching skills"
        exit 1
        ;;
esac
EOFMANAGE
    
    chmod +x "$INSTALL_DIR/manage.sh"
    
    log_success "Management script created: $INSTALL_DIR/manage.sh"
}

# Create simple README
create_simple_readme() {
    cat > "$INSTALL_DIR/QUICK_START.md" << 'EOF'
# Quick Start Guide - SWE100821

## Management Commands

```bash
cd $INSTALL_DIR

# Start services
./manage.sh start

# Stop services
./manage.sh stop

# Check status
./manage.sh status

# View logs
./manage.sh logs

# Test agent
./manage.sh test
```

## Auto-Start on Boot

Services are **already enabled** to start on boot!

To disable:
```bash
./manage.sh disable
```

To re-enable:
```bash
./manage.sh enable
```

## Direct Usage

```bash
# CLI mode (one-off)
xagent agent -m "your question"

# Interactive mode
xagent agent

# Check status
xagent status
```

## Service Management

```bash
# Manual service control
sudo systemctl start xagent-gateway
sudo systemctl stop xagent-gateway
sudo systemctl restart xagent-gateway
sudo systemctl status xagent-gateway

# View service logs
sudo journalctl -u xagent-gateway -f
```

## Configuration

- Config: `~/.xagent/config.json`
- Workspace: `~/.xagent/workspace/`
- Logs: `/var/log/xagent-gateway.log`

## Installed Services

1. **ollama.service** - AI model server (port 11434)
2. **xagent-gateway.service** - AI agent gateway (port 18790)

Both start automatically on boot!
EOF
    
    log_success "Quick start guide created"
}

# Ensure OpenClaw skills archive is present
clone_openclaw_skills() {
    local SKILLS_DIR="$INSTALL_DIR/skills"
    if [ -d "$SKILLS_DIR/skills" ]; then
        log_info "Skills archive already present"
    else
        log_info "Cloning OpenClaw skills hub (10,000+ community skills)..."
        if git clone --depth 1 https://github.com/moltbot/skills.git "$SKILLS_DIR" 2>/dev/null; then
            log_success "Skills hub cloned to $SKILLS_DIR"
        else
            log_warn "Could not clone skills hub (no internet?). Skipping."
            log_warn "Clone manually later: git clone https://github.com/moltbot/skills.git skills"
            return 0
        fi
    fi
    
    log_info "Skills hub ready. Use './manage.sh skills list' to browse."
    log_info "Install skills with: ./manage.sh skills install <owner>/<slug>"
}

# Cleanup old MD files - SWE100821: Keep root clean
cleanup_docs() {
    log_info "Organizing documentation..."
    
    mkdir -p "$INSTALL_DIR/docs"
    
    # Move ALL .md and .txt files to docs/ (except README.md)
    cd "$INSTALL_DIR"
    for file in *.md *.txt; do
        if [ -f "$file" ] && [ "$file" != "README.md" ]; then
            mv "$file" docs/ 2>/dev/null || true
        fi
    done
    
    log_success "Documentation organized in $INSTALL_DIR/docs/"
}

# Main installation
main() {
    show_banner
    
    log_info "Starting unified installation..."
    echo ""
    
    detect_system
    echo ""
    
    log_info "This will:"
    echo "  1. Install all dependencies (Python, Go, Ollama)"
    echo "  2. Build Xagent"
    echo "  3. Create systemd services"
    echo "  4. Enable auto-start on boot"
    echo ""
    
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Installation cancelled"
        exit 0
    fi
    
    echo ""
    log_info "Installing... (this may take 5-10 minutes)"
    echo ""
    
    # Installation steps
    install_dependencies
    install_go
    
    if [ "$OLLAMA_SUPPORTED" = true ]; then
        install_ollama
        download_model
    fi
    
    install_xagent
    configure_xagent
    clone_openclaw_skills
    create_xagent_service
    enable_services
    setup_network_blocklist
    create_management_script
    create_simple_readme
    cleanup_docs
    
    # Summary
    echo ""
    echo "=============================================="
    log_success "Installation Complete!"
    echo "=============================================="
    echo ""
    echo "Platform: $PLATFORM (Ubuntu $OS_VERSION)"
    echo "Python: $PYTHON_VERSION"
    echo "Ollama: $([ "$OLLAMA_SUPPORTED" = true ] && echo 'Installed' || echo 'Skipped')"
    if [ "$OLLAMA_SUPPORTED" = true ]; then
        echo "Model: $(cat ~/.ollama_model 2>/dev/null | cut -d= -f2 || echo 'Not set')"
    fi
    echo ""
    echo "Services enabled for auto-start on boot:"
    echo "  ✓ ollama.service $([ "$OLLAMA_SUPPORTED" = false ] && echo '(skipped)' || echo '')"
    echo "  ✓ xagent-gateway.service"
    echo ""
    echo "Quick Commands:"
    echo "  Start:   ./manage.sh start"
    echo "  Stop:    ./manage.sh stop"
    echo "  Status:  ./manage.sh status"
    echo "  Test:    ./manage.sh test"
    echo ""
    echo "Skills (10,000+ from OpenClaw hub):"
    echo "  Search:  ./manage.sh skills search <keyword>"
    echo "  Install: ./manage.sh skills install <owner>/<slug>"
    echo "  Curated: ./manage.sh skills install-curated"
    echo ""
    echo "Or use directly:"
    echo "  xagent agent -m 'Hello!'"
    echo ""
    echo "Documentation: $INSTALL_DIR/docs/"
    echo "Quick Start:   $INSTALL_DIR/QUICK_START.md"
    echo ""
    
    read -p "Start services now? (Y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        echo ""
        log_info "Starting services..."
        sudo systemctl start ollama 2>/dev/null || true
        sleep 2
        sudo systemctl start xagent-gateway
        sleep 2
        echo ""
        log_success "Services started!"
        echo ""
        echo "Check status: ./manage.sh status"
        echo "View logs:    ./manage.sh logs"
    else
        echo ""
        log_info "Services not started. Start manually with: ./manage.sh start"
    fi
    
    echo ""
    log_success "Setup complete! Reboot to test auto-start."
}

# Run main
main "$@"
