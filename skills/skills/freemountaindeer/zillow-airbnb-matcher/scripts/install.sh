#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Zillow × Airbnb Matcher — OpenClaw Skill Installer
# Run this once to set up the skill
#
# Usage:
#   bash scripts/install.sh
#   bash scripts/install.sh --with-token YOUR_RAPIDAPI_KEY_HERE
#
# What it does:
#   1. Checks Node.js is installed (requires v16+)
#   2. Installs npm dependencies
#   4. Runs the demo to confirm everything works
# ─────────────────────────────────────────────────────────────────────────────

set -e  # Stop if any command fails

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$SKILL_DIR/.env"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}=====================================================${NC}"
echo -e "${BLUE}   🏠 Zillow × Airbnb Matcher — Skill Installer      ${NC}"
echo -e "${BLUE}=====================================================${NC}"
echo ""

# ─── Step 1: Check Node.js ────────────────────────────────────────────────────

echo -e "${YELLOW}Step 1: Checking Node.js...${NC}"

if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed.${NC}"
    echo "   Install it from: https://nodejs.org (choose 'LTS' version)"
    echo "   Or on Linux: curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash - && sudo apt-get install -y nodejs"
    exit 1
fi

NODE_VERSION=$(node -e "process.stdout.write(process.version.slice(1).split('.')[0])")
if [ "$NODE_VERSION" -lt 16 ]; then
    echo -e "${RED}❌ Node.js v$NODE_VERSION found, but v16+ is required.${NC}"
    echo "   Update from: https://nodejs.org"
    exit 1
fi

echo -e "${GREEN}✅ Node.js $(node --version) found${NC}"

# ─── Step 2: Install dependencies ────────────────────────────────────────────

echo ""
echo -e "${YELLOW}Step 2: Installing dependencies...${NC}"

cd "$SKILL_DIR"

if [ ! -f "package.json" ]; then
    echo -e "${RED}❌ package.json not found. Are you running this from the skill directory?${NC}"
    exit 1
fi

npm install --silent 2>&1 | tail -5

echo -e "${GREEN}✅ Dependencies installed${NC}"

# ─── Step 3: API Token setup ──────────────────────────────────────────────────

echo ""
echo -e "${YELLOW}Step 3: API Token setup...${NC}"

# Parse --with-token argument
RAPIDAPI_KEY=""
for arg in "$@"; do
    if [[ "$arg" == "--with-token" ]]; then
        shift
        RAPIDAPI_KEY="$1"
        break
    fi
done

# Check for token in common .env locations
EXISTING_TOKEN=""
if [ -f "$ENV_FILE" ]; then
    EXISTING_TOKEN=$(grep "^RAPIDAPI_KEY=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 | tr -d '"')
fi
if [ -z "$EXISTING_TOKEN" ] && [ -f "$HOME/.clawdbot/.env" ]; then
    EXISTING_TOKEN=$(grep "^RAPIDAPI_KEY=" $HOME/.clawdbot/.env 2>/dev/null | cut -d'=' -f2 | tr -d '"')
fi

if [ -n "$RAPIDAPI_KEY" ]; then
    # Write token to skill .env
    if [ -f "$ENV_FILE" ]; then
        # Update existing file
        if grep -q "^RAPIDAPI_KEY=" "$ENV_FILE"; then
            sed -i "s/^RAPIDAPI_KEY=.*/RAPIDAPI_KEY=$RAPIDAPI_KEY/" "$ENV_FILE"
        else
            echo "RAPIDAPI_KEY=$RAPIDAPI_KEY" >> "$ENV_FILE"
        fi
    else
        echo "RAPIDAPI_KEY=$RAPIDAPI_KEY" > "$ENV_FILE"
    fi
    echo -e "${GREEN}✅ RapidAPI key saved to .env${NC}"
elif [ -n "$EXISTING_TOKEN" ]; then
    echo -e "${GREEN}✅ RapidAPI key found in existing .env${NC}"
else
    echo -e "${YELLOW}ℹ️  No RapidAPI key provided.${NC}"
    echo "   The demo mode will still work without a token."
    echo "   For live searches, get a free token at: https://rapidapi.com"
    echo "   Then run: echo 'RAPIDAPI_KEY=your_token' >> $ENV_FILE"
fi

# ─── Step 4: Run demo test ────────────────────────────────────────────────────

echo ""
echo -e "${YELLOW}Step 4: Running demo test...${NC}"
echo ""

node "$SKILL_DIR/scripts/search.js" --demo

echo ""
echo -e "${GREEN}=====================================================${NC}"
echo -e "${GREEN}  ✅ Installation complete! ${NC}"
echo -e "${GREEN}=====================================================${NC}"
echo ""
echo "Commands:"
echo "  Demo:           node scripts/search.js --demo"
echo "  Live search:    node scripts/search.js --zip 78704"
echo "  By city:        node scripts/search.js --city \"Nashville, TN\""
echo "  Commercial:     node scripts/search.js --demo --commercial"
echo ""
echo "Chat with your agent:"
echo "  \"search airbnb 78704\""
echo "  \"check properties Nashville TN\""
echo "  \"airbnb demo\""
echo ""
echo "For full setup guide, open: GUIDE-FOR-MATTHEW.md"
echo ""
