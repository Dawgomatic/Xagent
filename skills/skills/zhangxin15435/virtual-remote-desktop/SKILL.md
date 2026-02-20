---
name: virtual-remote-desktop
description: Starts and manages a secure noVNC virtual desktop on headless Linux using Xvfb, x11vnc, and a token-gated noVNC web proxy. Use for remote visual login, captcha handling, and start/stop/status/health operations.
read_when:
  - User asks for noVNC remote login on headless Linux
  - User needs visual captcha handling on server
  - User asks to start, stop, inspect, or health-check virtual desktop
metadata:
  {"clawdbot":{"emoji":"🖥️","requires":{"bins":["Xvfb","fluxbox","x11vnc","node","python3"],"paths":["/root/.openclaw/workspace/novnc-web"],"optionalBins":["google-chrome","chromium","/root/.cache/ms-playwright/chromium-1208/chrome-linux64/chrome"]},"safety":{"persists":["WORKDIR/logs","WORKDIR/chrome-profile","WORKDIR/pids.env","WORKDIR/vncpass","WORKDIR/access.token"],"network":["api.ipify.org","ifconfig.me","checkip.amazonaws.com"],"disclosure":"Stores browser profile data (cookies/session) for persistence. Run only on trusted hosts."}}}
---

# Virtual Remote Desktop (noVNC)

## Usage (Minimal Steps)

1) Start

```bash
bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/start_vrd.sh
```

2) Open the `One-click URL` from output, then enter the `VNC Password`.

3) After login, check status and health:

```bash
bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/status_vrd.sh
bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/health_vrd.sh
```

4) Stop:

```bash
bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/stop_vrd.sh
```

## Key Configuration (Common)

- `CHROME_PROFILE_DIR`: Persistent Chrome profile directory (default `${WORKDIR}/chrome-profile`)
- `AUTO_LAUNCH_URL`: URL to open automatically after startup
- `AUTO_STOP_IDLE_SECS`: Auto-stop timeout in seconds when idle (default 900)
- `NOVNC_BIND`: Listen address (default `0.0.0.0`)
- `ACCESS_TOKEN_TTL_SECS`: Access token TTL in seconds (default 86400)

## Security and Persistence Notes

- Uses a random `VNC_PASS` by default and token-gated access.
- Stores token in `WORKDIR/access.token` with file mode `600` (not in plain `pids.env`).
- Login data is persisted in `CHROME_PROFILE_DIR` when possible, but session longevity still depends on the target site's auth/session policy.
