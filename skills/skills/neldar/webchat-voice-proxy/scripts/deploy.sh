#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
WORKSPACE="${WORKSPACE:-$HOME/.openclaw/workspace}"
VOICE_DIR="$WORKSPACE/voice-input"
UI_DIR="${OPENCLAW_UI_DIR:-$(npm -g root 2>/dev/null)/openclaw/dist/control-ui}"
INDEX="$UI_DIR/index.html"
ASSET_DIR="$UI_DIR/assets"
CFG="${OPENCLAW_CONFIG_PATH:-$HOME/.openclaw/openclaw.json}"
VOICE_HTTPS_PORT="${VOICE_HTTPS_PORT:-8443}"
VOICE_HOST="${VOICE_HOST:-}"

if [[ -z "$VOICE_HOST" ]]; then
  VOICE_HOST="$(hostname -I 2>/dev/null | awk '{print $1}')"
fi
if [[ -z "$VOICE_HOST" ]]; then
  VOICE_HOST="127.0.0.1"
fi

ALLOWED_ORIGIN="https://${VOICE_HOST}:${VOICE_HTTPS_PORT}"

mkdir -p "$VOICE_DIR"

# 0) Copy bundled assets from skill -> workspace runtime dir
cp -f "$SKILL_DIR/assets/voice-input.js" "$VOICE_DIR/voice-input.js"
cp -f "$SKILL_DIR/assets/https-server.py" "$VOICE_DIR/https-server.py"

# 1) Deploy voice-input asset + inject index (idempotent)
mkdir -p "$ASSET_DIR"
cp -f "$VOICE_DIR/voice-input.js" "$ASSET_DIR/voice-input.js"

if ! grep -q 'voice-input.js' "$INDEX"; then
  sed -i 's|</body>|    <script src="./assets/voice-input.js"></script>\n  </body>|' "$INDEX"
fi

# 2) Ensure gateway allowedOrigins contains computed origin
python3 - << PY
import json
p='${CFG}'
origin='${ALLOWED_ORIGIN}'
with open(p,'r',encoding='utf-8') as f: c=json.load(f)
g=c.setdefault('gateway',{})
cu=g.setdefault('controlUi',{})
orig=cu.setdefault('allowedOrigins',[])
if origin not in orig:
    orig.append(origin)
    with open(p,'w',encoding='utf-8') as f: json.dump(c,f,indent=2,ensure_ascii=False); f.write('\n')
    print(f'allowedOrigin ensured: {origin}')
PY

# 3) Install/refresh HTTPS proxy service
mkdir -p "$HOME/.config/systemd/user"
cat > "$HOME/.config/systemd/user/openclaw-voice-https.service" <<UNIT
[Unit]
Description=OpenClaw Voice HTTPS Proxy (Control UI + WS + Transcribe)
After=network.target

[Service]
Type=simple
ExecStart=${WORKSPACE}/.venv-faster-whisper/bin/python ${VOICE_DIR}/https-server.py
Restart=always
RestartSec=2
Environment=VOICE_HTTPS_PORT=${VOICE_HTTPS_PORT}
Environment=VOICE_ALLOWED_ORIGIN=${ALLOWED_ORIGIN}

[Install]
WantedBy=default.target
UNIT

systemctl --user daemon-reload
systemctl --user enable --now openclaw-voice-https.service
systemctl --user restart openclaw-voice-https.service

# 4) Install gateway startup hook (survives openclaw update)
HOOK_DIR="$HOME/.openclaw/hooks/voice-input-inject"
mkdir -p "$HOOK_DIR"
cp -f "$SKILL_DIR/hooks/handler.ts" "$HOOK_DIR/handler.ts"
cp -f "$SKILL_DIR/hooks/inject.sh" "$HOOK_DIR/inject.sh"
cp -f "$SKILL_DIR/hooks/HOOK.md" "$HOOK_DIR/HOOK.md"
chmod +x "$HOOK_DIR/inject.sh"
echo "hook installed: $HOOK_DIR"

# 5) Restart gateway so allowedOrigins/injection hook applies cleanly
openclaw gateway restart >/dev/null 2>&1 || true

echo "deploy:ok host=${VOICE_HOST} port=${VOICE_HTTPS_PORT} origin=${ALLOWED_ORIGIN}"
