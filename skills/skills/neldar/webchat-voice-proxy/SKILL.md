---
name: webchat-voice-proxy
description: >
  Voice input and microphone button for OpenClaw WebChat Control UI. Adds a mic
  button to chat, records audio via browser MediaRecorder, transcribes locally
  via faster-whisper, and injects text into the conversation. Includes HTTPS/WSS
  reverse proxy, TLS cert management, and gateway hook for update safety. Fully
  local speech-to-text, no API costs. Real-time VU meter shows voice activity.
  Keywords: voice input, microphone, WebChat, Control UI, speech to text, STT,
  local transcription, MediaRecorder, HTTPS proxy, voice button, mic button.
requires:
  config_paths:
    - ~/.openclaw/openclaw.json (appends allowedOrigins entry)
  modified_paths:
    - <npm-global>/openclaw/dist/control-ui/index.html (injects script tag)
    - <npm-global>/openclaw/dist/control-ui/assets/voice-input.js (copies asset)
    - ~/.config/systemd/user/openclaw-voice-https.service (creates unit)
    - ~/.openclaw/hooks/voice-input-inject/ (creates startup hook)
    - ~/.openclaw/workspace/voice-input/ (copies runtime files)
    - ~/.openclaw/workspace/voice-input/certs/ (generates self-signed TLS cert)
  env:
    - VOICE_HTTPS_PORT (optional, default: 8443)
    - VOICE_HOST (optional, auto-detected from hostname -I)
    - VOICE_ALLOWED_ORIGIN (optional, default: https://<VOICE_HOST>:<VOICE_HTTPS_PORT>)
  persistence:
    - "User systemd service: openclaw-voice-https.service (HTTPS/WSS proxy)"
    - "Gateway startup hook: voice-input-inject (re-injects JS after updates)"
  privileges: user-level only, no root/sudo required
  dependencies:
    - python3 with aiohttp (pip)
    - faster-whisper transcription service on port 18790
    - openssl (for self-signed cert generation)
---

# WebChat Voice Proxy

Set up a reboot-safe voice stack for OpenClaw WebChat (including the current polished mic/stop/hourglass UI states):
- HTTPS Control UI on port 8443
- `/transcribe` proxy to local faster-whisper service
- WebSocket passthrough to gateway (`ws://127.0.0.1:18789`)
- Voice button script injection into Control UI
- Real-time VU meter: button shadow/scale reacts to voice level

## Prerequisites (required)

This skill requires a **local faster-whisper HTTP service**.
Expected default:
- URL: `http://127.0.0.1:18790/transcribe`
- systemd user service: `openclaw-transcribe.service`

Verify before deployment:
```bash
systemctl --user is-active openclaw-transcribe.service
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18790/transcribe -X POST -H 'Content-Type: application/octet-stream' --data-binary 'x'
```

If this dependency is missing, set up faster-whisper first (model load + HTTP endpoint), then run this skill.

Related skills:
- `faster-whisper-local-service` (backend prerequisite)
- `webchat-voice-full-stack` (meta-installer that deploys both backend + proxy)

## Workflow

1. Ensure transcription service exists and is running (`openclaw-transcribe.service`).
2. Deploy `voice-input.js` to Control UI assets and inject script tag into `index.html`.
3. Configure gateway allowed origin for external HTTPS UI.
4. Run HTTPS+WSS proxy as persistent user systemd service (`openclaw-voice-https.service`).
5. Verify pairing/token/origin errors and resolve in order.

## Deploy

Run (auto-detect host IP):
```bash
bash scripts/deploy.sh
```

Or set host/port explicitly:
```bash
VOICE_HOST=10.0.0.42 VOICE_HTTPS_PORT=8443 bash scripts/deploy.sh
```

This script is idempotent.

## Quick verify

Run:
```bash
bash scripts/status.sh
```

Expected:
- both services active
- injection present
- `https:200`

## Common fixes

- `404 /chat?...` → SPA fallback missing in HTTPS proxy.
- `origin not allowed` → ensure deploy used correct `VOICE_HOST` and added matching HTTPS origin to `gateway.controlUi.allowedOrigins`.
- `token missing` → open URL with `?token=...` once.
- `pairing required` → approve pending device via `openclaw devices approve <requestId> --token <gateway-token>`.
- Mic breaks after reboot → cert paths must be persistent (not `/tmp`).
- No transcription result → check local faster-whisper endpoint first.

See `references/troubleshooting.md` for exact commands.

## What this skill modifies

Before installing, be aware of all system changes `deploy.sh` makes:

| What | Path | Action |
|---|---|---|
| Control UI HTML | `<npm-global>/openclaw/dist/control-ui/index.html` | Adds `<script>` tag for voice-input.js |
| Control UI asset | `<npm-global>/openclaw/dist/control-ui/assets/voice-input.js` | Copies mic button JS |
| Gateway config | `~/.openclaw/openclaw.json` | Adds HTTPS origin to `gateway.controlUi.allowedOrigins` |
| Systemd service | `~/.config/systemd/user/openclaw-voice-https.service` | Creates + enables persistent HTTPS proxy |
| Gateway hook | `~/.openclaw/hooks/voice-input-inject/` | Installs startup hook that re-injects JS after updates |
| Workspace files | `~/.openclaw/workspace/voice-input/` | Copies voice-input.js, https-server.py |
| TLS certs | `~/.openclaw/workspace/voice-input/certs/` | Auto-generated self-signed cert on first run |

The injected JS (`voice-input.js`) runs inside the Control UI and interacts with the chat input. Review the source before deploying.

## CORS Policy

The `/transcribe` proxy endpoint uses a configurable `Access-Control-Allow-Origin` header.
Set `VOICE_ALLOWED_ORIGIN` env var to restrict. Default: `https://<VOICE_HOST>:<VOICE_HTTPS_PORT>`.

## Uninstall

To fully revert all changes:

```bash
bash scripts/uninstall.sh
```

This will:
1. Stop and remove `openclaw-voice-https.service`
2. Remove the gateway startup hook
3. Remove `voice-input.js` from Control UI and undo the index.html injection
4. Remove the HTTPS origin from gateway config
5. Restart the gateway

Workspace files (`voice-input/`) and TLS certs are kept by default.
To remove them too:
```bash
rm -rf ~/.openclaw/workspace/voice-input
```

The faster-whisper backend is **not** touched by uninstall — remove it separately via `faster-whisper-local-service` if needed.
