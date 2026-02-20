#!/usr/bin/env bash
# Auto-Recall: Query memories before AI turn and inject as context
# This hook runs before each tool use

# Check if auto-recall is enabled (default: true)
VANAR_AUTO_RECALL="${VANAR_AUTO_RECALL:-true}"
[[ "$VANAR_AUTO_RECALL" != "true" ]] && exit 0

set -euo pipefail

API_BASE="https://api-neutron.vanarchain.com"
CONFIG_FILE="${HOME}/.config/neutron/credentials.json"

# Load credentials
API_KEY="${NEUTRON_API_KEY:-}"
AGENT_ID="${NEUTRON_AGENT_ID:-}"
USER_ID="${YOUR_AGENT_IDENTIFIER:-1}"

if [[ -z "$API_KEY" ]] && [[ -f "$CONFIG_FILE" ]]; then
    API_KEY=$(jq -r '.api_key // empty' "$CONFIG_FILE" 2>/dev/null || true)
    AGENT_ID=$(jq -r '.agent_id // empty' "$CONFIG_FILE" 2>/dev/null || true)
    USER_ID=$(jq -r '.your_agent_identifier // "1"' "$CONFIG_FILE" 2>/dev/null || true)
fi

# Skip if no credentials
if [[ -z "$API_KEY" || -z "$AGENT_ID" ]]; then
    exit 0
fi

# Get the user's latest message from stdin (OpenClaw passes context)
USER_MESSAGE="${OPENCLAW_USER_MESSAGE:-}"

if [[ -z "$USER_MESSAGE" ]]; then
    exit 0
fi

# Query for relevant memories
QUERY_PARAMS="appId=${AGENT_ID}&externalUserId=${USER_ID}"

response=$(curl -s -X POST "${API_BASE}/seeds/query?${QUERY_PARAMS}" \
    -H "Authorization: Bearer ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"query\":\"${USER_MESSAGE}\",\"limit\":5,\"threshold\":0.5}" 2>/dev/null || echo "{}")

# Extract memories if any
memories=$(echo "$response" | jq -r '.results[]?.content // empty' 2>/dev/null | head -500)

if [[ -n "$memories" ]]; then
    echo "---"
    echo "RECALLED MEMORIES:"
    echo "$memories"
    echo "---"
fi
