#!/bin/bash
# publish-to-clawhub.sh — Publish Buzz BD skill to ClawHub
# 
# Prerequisites:
#   npm install -g clawhub
#   clawhub login (authenticate with GitHub)
#
# Usage:
#   chmod +x scripts/publish-to-clawhub.sh
#   ./scripts/publish-to-clawhub.sh

set -e

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SLUG="buzz-bd"
NAME="Buzz BD"
VERSION="1.0.0"
CHANGELOG="Initial release — DexScreener token scanning, 100-point scoring, prospect briefs, ERC-8004 registered (ETH #25045, Base #17483)"

echo " Publishing Buzz BD skill to ClawHub..."
echo "   Directory: $SKILL_DIR"
echo "   Slug: $SLUG"
echo "   Version: $VERSION"
echo ""

# Verify SKILL.md exists
if [ ! -f "$SKILL_DIR/SKILL.md" ]; then
  echo " SKILL.md not found in $SKILL_DIR"
  exit 1
fi

# Verify no secrets in files
echo " Security check — scanning for leaked secrets..."
if grep -r "sk-ant-\|sk-proj-\|fc-[a-f0-9]\{32\}\|PRIVATE_KEY" "$SKILL_DIR" --include="*.mjs" --include="*.js" --include="*.md" --include="*.json" 2>/dev/null; then
  echo " SECRETS DETECTED — aborting publish!"
  echo "   Remove all API keys, private keys, and credentials before publishing."
  exit 1
fi
echo "    No secrets found"

# Publish
echo ""
echo " Publishing to ClawHub..."
clawhub publish "$SKILL_DIR" \
  --slug "$SLUG" \
  --name "$NAME" \
  --version "$VERSION" \
  --changelog "$CHANGELOG"

echo ""
echo " Buzz BD v$VERSION published to ClawHub!"
echo "   View: https://clawhub.ai/skills/$SLUG"
echo ""
echo " Install command for users:"
echo "   clawhub install $SLUG"
echo ""
echo " ERC-8004: ETH #25045 | Base #17483"
