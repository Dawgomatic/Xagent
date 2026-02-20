# Multi-Platform Installation Guide
**By SWE100821** | 2026-02-16  
**For:** Xavier (Ubuntu 20.04), RPi 3/4 (Ubuntu 22.04/24.04)

---

## 📋 Platform Compatibility Matrix

| Platform | Ubuntu | Python | Go | Ollama | Recommended Model |
|----------|--------|--------|-----|--------|-------------------|
| **Jetson Xavier** | 20.04 | 3.8 | 1.21.6 | ✅ GPU | llama3.1:8b |
| **Raspberry Pi 4** | 22.04 | 3.10 | 1.21.6 | ✅ CPU | phi3:3.8b |
| **Raspberry Pi 4** | 24.04 | 3.12 | 1.21.6 | ✅ CPU | phi3:3.8b |
| **Raspberry Pi 3** | 22.04 | 3.10 | 1.21.6 | ⚠️ Skip | N/A |
| **x86_64 Desktop** | Any | Auto | 1.21.6 | ✅ GPU | llama3.1:8b |

---

## 🚀 Quick Install (All Platforms)

```bash
cd /home/dawg/Desktop/AI_agents

# Make executable
chmod +x install_multiplatform.sh

# Run installer (detects everything automatically)
./install_multiplatform.sh
```

The script will:
1. ✅ Auto-detect your platform (Xavier, RPi3, RPi4, etc.)
2. ✅ Auto-detect Ubuntu version (20.04, 22.04, 24.04)
3. ✅ Install correct Python version for your Ubuntu
4. ✅ Install compatible Go version for your architecture
5. ✅ Install Ollama (if recommended for your platform)
6. ✅ Download appropriate model for your hardware
7. ✅ Build and configure PicoClaw
8. ✅ Create platform-specific configuration

---

## 🔍 What Gets Installed?

### Ubuntu 20.04 (Xavier)
```
Python: 3.8
Go: 1.21.6 (arm64)
Ollama: Yes (with CUDA support)
Model: llama3.1:8b
Performance: Excellent (GPU accelerated)
```

### Ubuntu 22.04 (RPi 3/4)
```
Python: 3.10
Go: 1.21.6 (arm64)
Ollama: 
  - RPi4: Yes (CPU only)
  - RPi3: No (insufficient resources)
Model: phi3:3.8b (RPi4 only)
Performance: 
  - RPi4: Good (optimized model)
  - RPi3: Use cloud API instead
```

### Ubuntu 24.04 (RPi 4)
```
Python: 3.12
Go: 1.21.6 (arm64)
Ollama: Yes (CPU only)
Model: phi3:3.8b
Performance: Good (latest optimizations)
```

---

## 📦 Installation Details

### Automatic Detection

The script detects:

**OS Version:**
- Reads `/etc/os-release`
- Determines Ubuntu 20.04, 22.04, or 24.04
- Selects appropriate Python version

**Hardware Platform:**
- Xavier: Checks `/etc/nv_tegra_release`
- Raspberry Pi: Checks `/proc/device-tree/model`
- Generic: Uses `uname -m` for architecture

**Python Selection:**
- Ubuntu 20.04 → Python 3.8
- Ubuntu 22.04 → Python 3.10
- Ubuntu 24.04 → Python 3.12

**Go Architecture:**
- x86_64 → amd64
- aarch64/arm64 → arm64
- armv7l → armv6l

---

## 🎯 Platform-Specific Recommendations

### Jetson Xavier (Ubuntu 20.04)
**Best Setup:**
```bash
# Runs automatically with installer
# Uses GPU acceleration via CUDA
Model: llama3.1:8b
Performance: ~5 tokens/sec (excellent)
```

**Manual Optimization:**
```bash
# Enable TensorRT for even faster inference
export OLLAMA_CUDA_VERSION=11.4
ollama pull llama3.1:8b
```

### Raspberry Pi 4 (Ubuntu 22.04/24.04)
**Best Setup:**
```bash
# Runs automatically with installer
# Uses CPU only (no GPU)
Model: phi3:3.8b
Performance: ~2 tokens/sec (usable)
```

**For Better Performance:**
```bash
# Use even smaller model if slow
ollama pull tinyllama:1.1b

# Update config
sed -i 's/phi3:3.8b/tinyllama:1.1b/' ~/.picoclaw/config.json
```

### Raspberry Pi 3 (Ubuntu 22.04)
**Recommendation:** Skip local model, use cloud API

```bash
# Install PicoClaw only
./install_multiplatform.sh
# (Will skip Ollama automatically)

# Configure for cloud API (optional)
cat > ~/.picoclaw/config.json << 'EOF'
{
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "placeholder"
    }
  },
  "providers": {
    "vllm": {
      "api_key": "your-api-key",
      "api_base": "https://your-cloud-api.com/v1"
    }
  }
}
EOF
```

Or run agent on different machine and use RPi3 as client only.

---

## 🛠️ Manual Installation Steps

If you prefer manual control:

### 1. Detect Your System
```bash
# Check OS version
cat /etc/os-release

# Check hardware
cat /proc/device-tree/model  # For RPi
cat /etc/nv_tegra_release     # For Xavier

# Check architecture
uname -m
```

### 2. Install Python (Version-Specific)

**Ubuntu 20.04:**
```bash
sudo apt-get update
sudo apt-get install -y python3.8 python3.8-dev python3.8-venv python3-pip
```

**Ubuntu 22.04:**
```bash
sudo apt-get update
sudo apt-get install -y python3.10 python3.10-dev python3.10-venv python3-pip
```

**Ubuntu 24.04:**
```bash
sudo apt-get update
sudo apt-get install -y python3.12 python3.12-dev python3.12-venv python3-pip
```

### 3. Install Go

**For ARM64 (Xavier, RPi 3/4):**
```bash
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**For x86_64:**
```bash
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 4. Install Ollama (If Supported)

```bash
curl -fsSL https://ollama.com/install.sh | sh
sudo systemctl start ollama
sudo systemctl enable ollama
```

### 5. Download Model

**Xavier or Desktop:**
```bash
ollama pull llama3.1:8b
```

**Raspberry Pi 4:**
```bash
ollama pull phi3:3.8b
```

**Raspberry Pi 3:**
```bash
# Skip - not recommended
```

### 6. Install PicoClaw

```bash
cd /home/dawg/Desktop/AI_agents
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw
make deps
make build
sudo ln -s $(pwd)/picoclaw /usr/local/bin/picoclaw
```

---

## 🔧 Troubleshooting by Platform

### Jetson Xavier Issues

**Problem: CUDA not detected**
```bash
# Check CUDA version
nvcc --version

# Reinstall Ollama with CUDA support
curl -fsSL https://ollama.com/install.sh | sh
```

**Problem: Out of memory**
```bash
# Check memory usage
tegrastats

# Use smaller model
ollama pull mistral:7b
```

### Raspberry Pi 4 Issues

**Problem: Very slow responses**
```bash
# Use smallest model
ollama pull tinyllama:1.1b

# Reduce token limit in config
sed -i 's/"max_tokens": 4096/"max_tokens": 1024/' ~/.picoclaw/config.json
```

**Problem: System freezes**
```bash
# Increase swap space
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### Raspberry Pi 3 Issues

**Problem: Ollama won't install**
```
This is expected - RPi3 has insufficient resources.
Use cloud API or run agent on different machine.
```

---

## 📊 Performance Comparison

| Platform | Model | Tokens/Sec | RAM Usage | Startup Time |
|----------|-------|------------|-----------|--------------|
| Xavier | llama3.1:8b | ~5-7 | 6GB | 3s |
| RPi4 (22.04) | phi3:3.8b | ~1.5-2 | 3GB | 10s |
| RPi4 (24.04) | phi3:3.8b | ~2-2.5 | 2.5GB | 8s |
| RPi3 | N/A | N/A | N/A | N/A |
| Desktop (x86) | llama3.1:8b | ~10-15 | 6GB | 2s |

---

## 🔍 Verify Installation

After running the installer:

```bash
# Check platform info
cat /home/dawg/Desktop/AI_agents/PLATFORM_INFO.txt

# Test PicoClaw
picoclaw status

# Test agent
picoclaw agent -m "What platform am I running on?"

# Check Ollama (if installed)
curl http://localhost:11434/api/tags

# View configuration
cat ~/.picoclaw/config.json | grep -E "model|api_base"
```

---

## 🚀 Next Steps After Installation

### All Platforms:
```bash
# 1. Test basic functionality
picoclaw agent -m "Hello!"

# 2. Check logs
picoclaw status

# 3. Read documentation
cat /home/dawg/Desktop/AI_agents/README_PICOCLAW.md
```

### Xavier (High Performance):
```bash
# Enable more features
picoclaw skills list-builtin
picoclaw skills install-builtin

# Add Telegram bot
# See README_PICOCLAW.md for setup
```

### RPi 4 (Good Performance):
```bash
# Test web search
picoclaw agent -m "Search for today's weather"

# Try coding task
picoclaw agent -m "Write a simple Python calculator"
```

### RPi 3 (Limited):
```bash
# Use as gateway only
picoclaw gateway

# Or configure for cloud API
# See picoclaw_secure_setup_guide.md
```

---

## 📝 Configuration Files by Platform

### Xavier (20.04)
```json
{
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "llama3.1:8b",
      "max_tokens": 8192,
      "temperature": 0.7
    }
  },
  "providers": {
    "vllm": {
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

### RPi 4 (22.04/24.04)
```json
{
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "phi3:3.8b",
      "max_tokens": 2048,
      "temperature": 0.7
    }
  },
  "providers": {
    "vllm": {
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

### RPi 3 (22.04) - Cloud API
```json
{
  "agents": {
    "defaults": {
      "provider": "vllm",
      "model": "remote-model",
      "max_tokens": 4096
    }
  },
  "providers": {
    "vllm": {
      "api_key": "your-api-key",
      "api_base": "https://api.your-provider.com/v1"
    }
  }
}
```

---

## 🔄 Updating

```bash
# Update PicoClaw
cd /home/dawg/Desktop/AI_agents/picoclaw
git pull
make build

# Update Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Update models
ollama pull llama3.1:8b  # or your model
```

---

## 🆘 Support

**Platform detected incorrectly?**
```bash
# Edit the detection logic in install_multiplatform.sh
# Or set manually:
export PLATFORM=xavier  # or rpi3, rpi4
./install_multiplatform.sh
```

**Python version issues?**
```bash
# Check what's installed
ls /usr/bin/python*

# Update config to use specific version
sudo update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.8 1
```

**Need help?**
- See: `README_PICOCLAW.md`
- See: `picoclaw_ollama_setup.md`
- Check: `/home/dawg/Desktop/AI_agents/PLATFORM_INFO.txt`

---

## Summary

✅ **Single script** handles all platforms  
✅ **Auto-detects** OS version and hardware  
✅ **Installs** correct Python, Go, Ollama versions  
✅ **Configures** optimal model for your hardware  
✅ **Works** on Xavier, RPi3, RPi4, and desktop  

**Just run:** `./install_multiplatform.sh`

---

**Created by SWE100821** | 2026-02-16  
**For multi-platform AI agent deployment**
