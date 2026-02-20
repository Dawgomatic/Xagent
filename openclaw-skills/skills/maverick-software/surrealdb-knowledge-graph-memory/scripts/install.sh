#!/bin/bash
set -e

echo "=== SurrealDB Memory Skill Installer ==="

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     PLATFORM=linux;;
    Darwin*)    PLATFORM=macos;;
    MINGW*|MSYS*|CYGWIN*) PLATFORM=windows;;
    *)          PLATFORM="unknown";;
esac

echo "Detected platform: $PLATFORM"

# Check if SurrealDB is already installed
if command -v surreal &> /dev/null; then
    CURRENT_VERSION=$(surreal version 2>/dev/null | head -1 || echo "unknown")
    echo "SurrealDB already installed: $CURRENT_VERSION"
    read -p "Reinstall/update? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping SurrealDB installation."
        SKIP_INSTALL=true
    fi
fi

if [ -z "$SKIP_INSTALL" ]; then
    echo "Installing SurrealDB..."
    
    case "${PLATFORM}" in
        linux)
            curl -sSf https://install.surrealdb.com | sh
            ;;
        macos)
            if command -v brew &> /dev/null; then
                brew install surrealdb/tap/surreal
            else
                curl -sSf https://install.surrealdb.com | sh
            fi
            ;;
        windows)
            echo "On Windows, run: iwr https://windows.surrealdb.com -useb | iex"
            echo "Or use: choco install surreal"
            exit 1
            ;;
        *)
            echo "Unknown platform. Install manually from https://surrealdb.com/install"
            exit 1
            ;;
    esac
    
    echo "SurrealDB installed successfully!"
fi

# Verify installation
if ! command -v surreal &> /dev/null; then
    echo "ERROR: surreal command not found in PATH"
    echo "Add ~/.surrealdb to your PATH or reinstall"
    exit 1
fi

surreal version

# Create data directory
DATA_DIR="${HOME}/.openclaw/memory"
mkdir -p "$DATA_DIR"
echo "Data directory: $DATA_DIR"

# Install Python dependencies
echo ""
echo "Installing Python dependencies..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUIREMENTS="${SCRIPT_DIR}/requirements.txt"

if [ -f "$REQUIREMENTS" ]; then
    pip3 install -q -r "$REQUIREMENTS" 2>/dev/null || pip install -q -r "$REQUIREMENTS"
else
    pip3 install -q surrealdb openai pyyaml 2>/dev/null || pip install -q surrealdb openai pyyaml
fi

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Start SurrealDB:  surreal start --user root --pass root file:~/.openclaw/memory/knowledge.db"
echo "  2. Initialize schema: ./scripts/init-db.sh"
echo "  3. (Optional) Migrate: python3 scripts/migrate-sqlite.py"
