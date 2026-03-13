#!/bin/bash
#
# ClawdTalk - Setup Script (v1.1)
#
# Interactive setup for voice calling integration.
# Asks for API key, auto-detects names, and configures the gateway.
# Uses jq for all JSON manipulation (no python3 dependency).
#
# Usage: ./setup.sh
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/skill-config.json"

echo ""
echo " ClawdTalk Setup"
echo "==================="
echo ""
echo "This will set up voice calling for your Clawdbot/OpenClaw instance."
echo ""

# Check for required tools
echo " Checking requirements..."
for tool in node jq; do
    if ! command -v "$tool" &> /dev/null; then
        echo " Required tool '$tool' is not installed."
        exit 1
    fi
done
echo "   ✓ All required tools found"

# Check if already configured
if [ -f "$CONFIG_FILE" ]; then
    echo ""
    echo "  Configuration already exists!"
    echo ""
    echo "Current config: $CONFIG_FILE"
    echo ""
    read -p "Do you want to reconfigure? (y/N): " reconfigure
    if [[ ! "$reconfigure" =~ ^[Yy]$ ]]; then
        echo ""
        echo "Setup cancelled. Run './status.sh' to see current configuration."
        exit 0
    fi
    echo ""
fi

# Ask for API key
echo " API Key"
echo "==========="
echo ""
echo "You need an API key from ClawdTalk."
echo ""
echo "  1. Go to https://clawdtalk.com and sign in with Google"
echo "  2. Set up your phone number in Settings"
echo "  3. Generate an API key from the Dashboard"
echo ""
read -s -p "Enter your API key (or press Enter to skip for now): " api_key
echo ""

if [ -n "$api_key" ]; then
    echo "   ✓ API key saved"
else
    echo "     No API key entered — you can add it to skill-config.json later"
fi

# Auto-detect gateway config (support both clawdbot and openclaw)
echo ""
echo " Configuring voice agent..."

GATEWAY_CONFIG=""
CLI_NAME=""

if [ -f "${HOME}/.clawdbot/clawdbot.json" ]; then
    GATEWAY_CONFIG="${HOME}/.clawdbot/clawdbot.json"
    CLI_NAME="clawdbot"
elif [ -f "${HOME}/.openclaw/openclaw.json" ]; then
    GATEWAY_CONFIG="${HOME}/.openclaw/openclaw.json"
    CLI_NAME="openclaw"
fi

# Auto-detect CLI name
if [ -z "$CLI_NAME" ]; then
    if command -v clawdbot &> /dev/null; then
        CLI_NAME="clawdbot"
    elif command -v openclaw &> /dev/null; then
        CLI_NAME="openclaw"
    else
        CLI_NAME="clawdbot"
    fi
fi

voice_agent_added=false
main_agent_workspace=""

if [ -n "$GATEWAY_CONFIG" ] && [ -f "$GATEWAY_CONFIG" ]; then
    # Check if voice agent already exists (using jq)
    has_voice=$(jq -r '[.agents.list[]? | select(.id == "voice")] | length > 0' "$GATEWAY_CONFIG" 2>/dev/null || echo "false")

    # Read the main agent's name and workspace using jq
    main_agent_name=$(jq -r '(.agents.list[]? | select(.default == true or .id == "main") | .name) // "Assistant"' "$GATEWAY_CONFIG" 2>/dev/null || echo "Assistant")
    main_agent_workspace=$(jq -r '(.agents.list[]? | select(.default == true or .id == "main") | .workspace) // .agents.defaults.workspace // "/home/node/clawd"' "$GATEWAY_CONFIG" 2>/dev/null || echo "/home/node/clawd")

    if [ "$has_voice" = "true" ]; then
        echo "   ✓ Voice agent already configured in gateway"
        voice_agent_added=true
    else
        # Build the voice agent object (no systemPrompt — injected by ws-client via messages)
        voice_agent=$(jq -n \
            --arg name "${main_agent_name} Voice" \
            --arg workspace "$main_agent_workspace" \
            '{
                id: "voice",
                name: $name,
                workspace: $workspace
            }')

        # Add voice agent to agents.list and enable chatCompletions endpoint
        tmp_config=$(mktemp)
        if jq --argjson agent "$voice_agent" '
            .agents.list = (.agents.list // []) + [$agent] |
            .gateway.http.endpoints.chatCompletions.enabled = true
        ' "$GATEWAY_CONFIG" > "$tmp_config" 2>/dev/null; then
            mv "$tmp_config" "$GATEWAY_CONFIG"
            echo "   ✓ Added '${main_agent_name} Voice' agent to gateway config"
            echo "   ✓ Enabled /v1/chat/completions endpoint"
            voice_agent_added=true

            # Restart gateway to pick up new agent
            if command -v "$CLI_NAME" &> /dev/null; then
                echo "   ↻ Restarting gateway to apply changes..."
                $CLI_NAME gateway restart 2>/dev/null && echo "   ✓ Gateway restarted" || echo "     Restart failed — run '$CLI_NAME gateway restart' manually"
            else
                echo "     Run '$CLI_NAME gateway restart' to apply the new agent config"
            fi
        else
            rm -f "$tmp_config"
            echo "     Could not auto-configure — see manual steps below"
        fi
    fi
else
    echo "     Gateway config not found — see manual steps below"
    echo "   Checked: ~/.clawdbot/clawdbot.json and ~/.openclaw/openclaw.json"
fi

# Install Node dependencies
echo ""
echo " Installing dependencies..."
if [ -f "$SCRIPT_DIR/package.json" ]; then
    (cd "$SCRIPT_DIR" && npm install --production 2>/dev/null) && echo "   ✓ Dependencies installed" || echo "     npm install failed — run 'npm install' manually in the skill directory"
else
    echo "     No package.json found"
fi

# Detect user and agent names from workspace files
echo ""
echo " Detecting names from workspace..."

WORKSPACE="${main_agent_workspace:-$HOME/.openclaw/workspace}"
owner_name=""
agent_name=""

# Try to get owner name from USER.md ("What to call them:" or "Name:")
if [ -f "$WORKSPACE/USER.md" ]; then
    # First try "What to call them:" for the preferred name
    owner_name=$(grep -i "what to call them:" "$WORKSPACE/USER.md" 2>/dev/null | head -1 | sed 's/.*:\s*//' | tr -d '*' | xargs)
    # Fall back to "Name:" if not found
    if [ -z "$owner_name" ]; then
        owner_name=$(grep -i "^- \*\*Name:" "$WORKSPACE/USER.md" 2>/dev/null | head -1 | sed 's/.*:\s*//' | tr -d '*' | xargs)
        # Extract first name only
        owner_name=$(echo "$owner_name" | awk '{print $1}')
    fi
fi

# Try to get agent name from IDENTITY.md
if [ -f "$WORKSPACE/IDENTITY.md" ]; then
    agent_name=$(grep -i "^- \*\*Name:" "$WORKSPACE/IDENTITY.md" 2>/dev/null | head -1 | sed 's/.*:\s*//' | tr -d '*' | xargs)
fi

if [ -n "$owner_name" ]; then
    echo "   ✓ Owner name: $owner_name"
else
    echo "     Could not detect owner name from USER.md"
fi

if [ -n "$agent_name" ]; then
    echo "   ✓ Agent name: $agent_name"
else
    echo "     Could not detect agent name from IDENTITY.md"
fi

# Create skill-config.json
echo ""
echo " Creating skill configuration..."

# Build values
if [ -n "$api_key" ]; then
    api_key_json="\"$api_key\""
else
    api_key_json="null"
fi

owner_name_json="null"
agent_name_json="null"
if [ -n "$owner_name" ]; then
    owner_name_json="\"$owner_name\""
fi
if [ -n "$agent_name" ]; then
    agent_name_json="\"$agent_name\""
fi

# Build greeting with name if available
if [ -n "$owner_name" ]; then
    greeting="Hey $owner_name, what's up?"
else
    greeting="Hey, what's up?"
fi

cat > "$CONFIG_FILE" << EOF
{
  "api_key": $api_key_json,
  "server": "https://clawdtalk.com",
  "owner_name": $owner_name_json,
  "agent_name": $agent_name_json,
  "greeting": "$greeting",
  "max_conversation_turns": 20
}
EOF

echo "   ✓ Configuration saved to: $CONFIG_FILE"

# Display next steps
echo ""
echo " Setup Complete!"
echo "=================="
echo ""

if [ -z "$api_key" ]; then
    echo "Next steps:"
    echo ""
    echo "1. Get your API key from https://clawdtalk.com"
    echo "   • Sign in with Google"
    echo "   • Set up your phone number in Settings"
    echo "   • Generate an API key from the Dashboard"
    echo ""
    echo "2. Add it to skill-config.json:"
    echo "   Set the api_key field"
    echo ""
    echo "3. Start the connection:"
    echo "   ./scripts/connect.sh start"
else
    echo "Next steps:"
    echo ""
    echo "1. Make sure your phone number is set up at https://clawdtalk.com → Settings"
    echo ""
    echo "2. Start the connection:"
    echo "   ./scripts/connect.sh start"
fi
echo ""

if [ "$voice_agent_added" = true ]; then
    echo " Voice agent is configured and ready."
else
    echo "  Voice agent not auto-configured. Add it manually:"
    echo ""
    config_path="~/.clawdbot/clawdbot.json"
    if [ "$CLI_NAME" = "openclaw" ]; then
        config_path="~/.openclaw/openclaw.json"
    fi
    echo "   Edit $config_path and add to agents.list[]:"
    echo '   { "id": "voice", "name": "Voice" }'
    echo ""
    echo "   Also ensure chatCompletions is enabled:"
    echo '   "gateway": { "http": { "endpoints": { "chatCompletions": { "enabled": true } } } }'
    echo ""
    echo "   Then restart: $CLI_NAME gateway restart"
fi

echo ""
echo " Voice calls will use your main agent's full context and memory."
echo "   All tools available to your agent work on voice calls too."
echo ""
echo "To check status: ./status.sh"
echo ""
