# AI Agents - Complete Installation System
**By SWE100821** | One-command setup for Xavier, RPi3, RPi4, x86_64

---

## 🚀 Quick Start

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

**That's it!** Services auto-start on boot.

---

## 📦 What Gets Installed

- ✅ **Platform detection** (Xavier/RPi3/RPi4/x86_64)
- ✅ **Python** (3.8/3.10/3.12 based on Ubuntu version)
- ✅ **Go 1.21.6**
- ✅ **Ollama** + optimal model for your hardware
- ✅ **PicoClaw** AI agent
- ✅ **Systemd services** (auto-start on boot)

---

## 🔄 Auto-Start on Boot

After running `./start.sh`, these services start automatically on every boot:
- `ollama.service` - AI model server
- `picoclaw-gateway.service` - AI agent gateway

Manage with: `./manage.sh enable|disable`

---

## 📚 Documentation

See `docs/` folder for detailed guides:
- `docs/QUICK_START.md` - Quick reference
- `docs/picoclaw_security_audit.md` - Security analysis
- `docs/picoclaw_ollama_setup.md` - Detailed setup
- `docs/MULTIPLATFORM_GUIDE.md` - Platform specifics

---

## 🎯 Usage

```bash
# CLI mode
picoclaw agent -m "your question"

# Interactive mode
picoclaw agent

# Check status
picoclaw status
```

---

## 🛠️ Platform Support

| Platform | Ubuntu | Python | Model |
|----------|--------|--------|-------|
| Xavier | 20.04 | 3.8 | llama3.1:8b |
| RPi4 | 22.04 | 3.10 | phi3:3.8b |
| RPi4 | 24.04 | 3.12 | phi3:3.8b |
| RPi3 | 22.04 | 3.10 | Gateway only |

---

## 📁 Structure

```
.
├── start.sh          # Master installer (run once)
├── manage.sh         # Service management (run anytime)
├── README.md         # This file
└── docs/             # All documentation
```

---

## ✅ Complete Setup in 3 Steps

1. **Install:** `./start.sh`
2. **Start:** `./manage.sh start`
3. **Use:** `picoclaw agent -m "Hello!"`

Services auto-start on reboot!

---

**Created by SWE100821** | 100% local, private, secure AI agents
