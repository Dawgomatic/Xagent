#!/usr/bin/env bash
# Auto-Capture: Save conversation after AI turn

# Check if auto-capture is enabled (default: true)
VANAR_AUTO_CAPTURE="${VANAR_AUTO_CAPTURE:-true}"
[[ "$VANAR_AUTO_CAPTURE" != "true" ]] && exit 0

API_BASE="https://api-neutron.vanarchain.com"
CONFIG_FILE="${HOME}/.config/neutron/credentials.json"

API_KEY="${NEUTRON_API_KEY:-}"
AGENT_ID="${NEUTRON_AGENT_ID:-}"
USER_ID="${YOUR_AGENT_IDENTIFIER:-1}"

if [[ -z "$API_KEY" ]] && [[ -f "$CONFIG_FILE" ]]; then
    API_KEY=$(jq -r '.api_key // empty' "$CONFIG_FILE" 2>/dev/null || true)
    AGENT_ID=$(jq -r '.agent_id // empty' "$CONFIG_FILE" 2>/dev/null || true)
    USER_ID=$(jq -r '.your_agent_identifier // "1"' "$CONFIG_FILE" 2>/dev/null || true)
fi

[[ -z "$API_KEY" || -z "$AGENT_ID" ]] && exit 0

USER_MSG="${OPENCLAW_USER_MESSAGE:-}"
AI_RESP="${OPENCLAW_AI_RESPONSE:-}"

[[ -z "$USER_MSG" && -z "$AI_RESP" ]] && exit 0

TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
TITLE="Conversation - ${TS}"
CONTENT="User: ${USER_MSG}
Assistant: ${AI_RESP}"

QUERY_PARAMS="appId=${AGENT_ID}&externalUserId=${USER_ID}"

curl -s -X POST "${API_BASE}/seeds?${QUERY_PARAMS}" \
    -H "Authorization: Bearer ${API_KEY}" \
    -F "text=[\"${CONTENT}\"]" \
    -F 'textTypes=["text"]' \
    -F 'textSources=["auto_capture"]' \
    -F "textTitles=[\"${TITLE}\"]" > /dev/null 2>&1 &
