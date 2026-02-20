# Pyzotero CLI Installation Guide

Comprehensive installation instructions for pyzotero CLI, with special considerations for PEP 668-compliant systems.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation Methods](#installation-methods)
3. [Platform-Specific Instructions](#platform-specific-instructions)
4. [Post-Installation Setup](#post-installation-setup)
5. [Troubleshooting](#troubleshooting)
6. [Uninstallation](#uninstallation)

---

## Prerequisites

### Required Software

- **Python 3.7+** - For running pyzotero CLI
- **pip** or **pipx** - For package installation
- **Zotero 7+** - For local database access (required for CLI)

### Check Your System

Check if Python 3 is installed:
```bash
python3 --version
```

Check if pip is installed:
```bash
pip3 --version
```

Check if pipx is installed:
```bash
pipx --version
```

Check if Zotero is installed:
```bash
zotero --version
# OR
ls /Applications/Zotero.app  # macOS
```

---

## Installation Methods

### Method 1: pipx (Recommended for PEP 668-compliant systems)

pipx installs Python applications in isolated virtual environments.

#### Why pipx?

- **PEP 668 compliant**: Prevents conflicts with system Python packages
- **Isolated environments**: Each app gets its own virtual environment
- **Clean uninstallation**: Easy to remove without side effects
- **Security**: Reduces risk of system-wide package conflicts

#### Installation

**Step 1: Install pipx**

**Debian/Ubuntu:**
```bash
sudo apt update
sudo apt install pipx -y
pipx ensurepath
```

**Arch Linux:**
```bash
sudo pacman -S pipx
pipx ensurepath
```

**Fedora:**
```bash
sudo dnf install pipx
pipx ensurepath
```

**RHEL/CentOS/Rocky Linux:**
```bash
sudo yum install pipx
pipx ensurepath
```

**Alpine Linux:**
```bash
sudo apk add pipx
pipx ensurepath
```

**macOS (Homebrew):**
```bash
brew install pipx
pipx ensurepath
```

**Using pip (any system):**
```bash
pip install --user pipx
export PATH="$HOME/.local/bin:$PATH"
pipx ensurepath
```

**Note:** After `pipx ensurepath`, you may need to log out and back in, or run:
```bash
export PATH="$HOME/.local/bin:$PATH"
source ~/.bashrc  # or ~/.zshrc
```

**Step 2: Install pyzotero CLI**

```bash
# Install CLI with pipx
pipx install "pyzotero[cli]"
```

**Step 3: Verify Installation**

```bash
pyzotero --help
```

You should see help information without errors.

---

### Method 2: pip (Generic installation)

For systems without PEP 668 restrictions or when using virtual environments.

#### User Installation (Recommended)

```bash
pip install --user "pyzotero[cli]"
export PATH="$HOME/.local/bin:$PATH"
```

Make sure to add to PATH permanently:
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### Virtual Environment (Best practice)

```bash
# Create virtual environment
python3 -m venv ~/.venvs/pyzotero
source ~/.venvs/pyzotero/bin/activate

# Install
pip install "pyzotero[cli]"

# To use later, activate first:
# source ~/.venvs/pyzotero/bin/activate
```

---

### Method 3: Conda (Anaconda/Miniconda)

```bash
# Using conda-forge channel
conda install -c conda-forge pyzotero

# Or with pip in conda environment
conda create -n pyzotero python=3.11
conda activate pyzotero
pip install "pyzotero[cli]"
```

---

## Platform-Specific Instructions

### Debian 11+
**PEP 668 compliant** - Must use pipx or virtual environments.

```bash
sudo apt update
sudo apt install python3 python3-pip pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### Ubuntu 23.04+
**PEP 668 compliant** - Must use pipx or virtual environments.

```bash
sudo apt update
sudo apt install python3 python3-pip pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### Arch Linux
```bash
sudo pacman -S python python-pip pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### Fedora 34+
**PEP 668 compliant** - Must use pipx or virtual environments.

```bash
sudo dnf install python3 python3-pip pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### RHEL 9+ / CentOS 9+
**PEP 668 compliant** - May need EPEL for pipx.

```bash
sudo dnf install python3 python3-pip
sudo dnf install epel-release
sudo dnf install pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### Alpine Linux
```bash
sudo apk add python3 py3-pip py3-pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

### macOS
```bash
# Using Homebrew
brew install python@3 pipx
pipx ensurepath
pipx install "pyzotero[cli]"

# Or use pip
pip3 install --user "pyzotero[cli]"
```

### Windows
**PowerShell:**
```powershell
# Install pipx
pip install pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

**Git Bash / WSL:**
```bash
pip install pipx
pipx ensurepath
pipx install "pyzotero[cli]"
```

---

## Post-Installation Setup

### 1. Enable Local Zotero Access (Required)

The pyzotero CLI requires access to your **local Zotero database**. You must enable this in Zotero:

**Zotero 7 Setup:**

1. Open Zotero
2. Go to **Edit > Preferences** (or **Zotero > Settings** on macOS)
3. Click on the **Advanced** tab
4. Check the box: **"Allow other applications on this computer to communicate with Zotero"**
5. Click **OK** or **Apply**
6. **Restart Zotero**

⚠️ **Important:** Zotero must be running for the CLI to work. The CLI connects to your local Zotero database, not the online API.

### 2. Verify Installation

**Check CLI availability:**
```bash
pyzotero --help
```

**List collections:**
```bash
pyzotero listcollections
```

**If you see connection errors:**
- Make sure Zotero is running
- Verify local API is enabled in Zotero preferences
- Try restarting Zotero

### 3. Test Basic Search

```bash
# Try a basic search
pyzotero search -q "test"
```

---

## Troubleshooting

### Problem: Permission Denied on pip Installation

**Symptoms:**
```
ERROR: Could not install packages due to an EnvironmentError: [Errno 13] Permission denied
```

**Cause (PEP 668):**
You're trying to install to system directories on a PEP 668-compliant system.

**Solutions:**

1. **Use pipx (recommended):**
```bash
pipx install "pyzotero[cli]"
```

2. **Use user installation:**
```bash
pip install --user "pyzotero[cli]"
export PATH="$HOME/.local/bin:$PATH"
```

3. **Use virtual environment:**
```bash
python3 -m venv venv
source venv/bin/activate
pip install "pyzotero[cli]"
```

4. **Force system installation (⚠️ dangerous):**
```bash
sudo python3 -m pip install --break-system-packages "pyzotero[cli]"
```
⚠️ **Warning:** This can break your system Python installation!

---

### Problem: Command Not Found

**Symptoms:**
```
pyzotero: command not found
```

**Solutions:**

1. **Verify installation:**
```bash
pipx list
# OR
pip show pyzotero
```

2. **Ensure PATH is set:**
```bash
pipx ensurepath
source ~/.bashrc  # or ~/.zshrc
```

3. **Add to PATH manually:**
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

4. **If using virtual environment:**
```bash
source ~/.venvs/pyzotero/bin/activate
pyzotero --help
```

---

### Problem: Connection Error / Zotero Not Responding

**Symptoms:**
```
ConnectionRefusedError: [Errno 111] Connection refused
# OR
Error: Unable to connect to Zotero
```

**Solutions:**

1. **Ensure Zotero is running:**
```bash
# Check if Zotero is running
ps aux | grep zotero

# If not running, start Zotero
zotero &
```

2. **Enable local API in Zotero:**
   - Open Zotero
   - **Edit > Preferences > Advanced**
   - Check: **"Allow other applications on this computer to communicate with Zotero"**
   - **Restart Zotero**

3. **Restart Zotero:**
```bash
# Close Zotero, then start it again
zotero
```

4. **Check Zotero version:**
   - Ensure you have Zotero 7 or newer
   - Older versions may not support the local API

---

### Problem: pipx Not Available

**Symptoms:**
```
pipx: command not found
```

**Solutions:**

1. **Install via pip:**
```bash
pip install --user pipx
export PATH="$HOME/.local/bin:$PATH"
pipx ensurepath
```

2. **Install via package manager:**
```bash
# Debian/Ubuntu
sudo apt install pipx

# Arch
sudo pacman -S pipx

# Fedora
sudo dnf install pipx
```

---

### Problem: No Results Found

**Symptoms:**
```bash
pyzotero search -q "anything"
# Returns no results
```

**Solutions:**

1. **Verify Zotero has data:**
   - Open Zotero
   - Check if you have items in your library

2. **Try a different search:**
```bash
# List item types to see what's available
pyzotero itemtypes

# List collections
pyzotero listcollections

# Search in a specific collection
pyzotero listcollections
pyzotero search --collection <COLLECTION_ID> -q "test"
```

3. **Check full-text indexing (for --fulltext search):**
   - In Zotero, ensure PDFs have been indexed
   - Right-click a PDF > "Retrieve Metadata" if needed

---

### Problem: JSON Output is Empty

**Symptoms:**
```bash
pyzotero search -q "test" --json
# Returns: []
```

**Solutions:**

1. **Verify search term exists:**
```bash
# Try without --json first
pyzotero search -q "test"
```

2. **Check Zotero is running and has data**

3. **Try full-text search:**
```bash
pyzotero search -q "test" --fulltext
```

---

## Uninstallation

### Using pipx

```bash
pipx uninstall pyzotero
```

### Using pip

```bash
pip uninstall pyzotero
```

### Remove Virtual Environment (if used)

```bash
# Remove virtual environment
rm -rf ~/.venvs/pyzotero

# Remove cloned repository (if installed from source)
rm -rf ~/pyzotero
```

### Remove PATH Entries

Edit your shell configuration and remove PATH entries:

**For .bashrc:**
```bash
nano ~/.bashrc
# Remove: export PATH="$HOME/.local/bin:$PATH"
```

**For .zshrc:**
```bash
nano ~/.zshrc
# Remove: export PATH="$HOME/.local/bin:$PATH"
```

---

## Security Best Practices

1. **Don't use `sudo pip install`** - Use pipx or user installation instead
2. **Don't use `--break-system-packages`** - This bypasses PEP 668 protections
3. **Use virtual environments** - For development projects
4. **Keep packages updated** - Run update commands regularly:
   ```bash
   pipx upgrade pyzotero
   # OR
   pip install --upgrade "pyzotero[cli]"
   ```

---

## Summary of Recommendations

| Scenario | Recommended Method |
|----------|-------------------|
| **PEP 668 systems** | `pipx install "pyzotero[cli]"` |
| **Non-PEP 668 systems** | `pip install --user "pyzotero[cli]"` |
| **Development/Testing** | Editable install with venv |
| **Conda environments** | `pip install "pyzotero[cli]"` in conda env |

---

**Important Reminders:**

1. **Zotero 7+ is required** for local database access
2. **Enable local API** in Zotero preferences (Advanced tab)
3. **Zotero must be running** to use the CLI
4. **Full-text search** requires PDFs to be indexed in Zotero
5. **Use pipx** on PEP 668-compliant systems to avoid conflicts

**Remember:** Always use pipx or virtual environments on PEP 668-compliant systems to avoid conflicts and maintain system stability.
