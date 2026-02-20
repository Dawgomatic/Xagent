#!/usr/bin/env bash
set -euo pipefail

DISPLAY_NUM="${DISPLAY_NUM:-99}"
DISPLAY=":${DISPLAY_NUM}"
GEOM="${GEOM:-1920x1080}"
DEPTH="${DEPTH:-24}"
VNC_PORT="${VNC_PORT:-5901}"
NOVNC_PORT="${NOVNC_PORT:-6080}"
WORKDIR="${WORKDIR:-/root/.openclaw/vrd-data}"
NOVNC_WEB="${NOVNC_WEB:-/root/.openclaw/workspace/novnc-web}"
NOVNC_BIND="${NOVNC_BIND:-0.0.0.0}"
PUBLIC_HOST="${PUBLIC_HOST:-}"
ACCESS_TOKEN="${ACCESS_TOKEN:-}"
ACCESS_TOKEN_TTL_SECS="${ACCESS_TOKEN_TTL_SECS:-86400}"
AUTO_STOP_IDLE_SECS="${AUTO_STOP_IDLE_SECS:-900}"
AUTO_STOP_CHECK_SECS="${AUTO_STOP_CHECK_SECS:-15}"
AUTO_LAUNCH_URL="${AUTO_LAUNCH_URL:-}"

LOGDIR="${WORKDIR}/logs"
PIDFILE="${WORKDIR}/pids.env"
PASSFILE="${WORKDIR}/vncpass"
TOKEN_FILE="${WORKDIR}/access.token"
CHROME_PROFILE_DIR="${CHROME_PROFILE_DIR:-${WORKDIR}/chrome-profile}"

mkdir -p "${WORKDIR}" "${LOGDIR}"
mkdir -p "${CHROME_PROFILE_DIR}"
rm -f "/tmp/.X${DISPLAY_NUM}-lock" "/tmp/.X11-unix/X${DISPLAY_NUM}" || true

need_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[ERR] missing command: $1" >&2
    exit 1
  }
}

need_bin Xvfb
need_bin fluxbox
need_bin x11vnc
need_bin node
need_bin python3
# Playwright Chromium path
CHROME_BIN="${CHROME_BIN:-/root/.cache/ms-playwright/chromium-1208/chrome-linux64/chrome}"
if [[ ! -f "${CHROME_BIN}" ]]; then
  echo "[ERR] Chrome binary not found: ${CHROME_BIN}" >&2
  exit 1
fi

if [[ ! -d "${NOVNC_WEB}" ]]; then
  echo "[ERR] NOVNC_WEB not found: ${NOVNC_WEB}" >&2
  exit 1
fi

# 让 node 能稳定 resolve 到 workspace 的依赖（尤其是 ws）
WORKSPACE="${WORKSPACE:-/root/.openclaw/workspace}"
NODE_PATH="${NODE_PATH:-${WORKSPACE}/node_modules}"

NOVNC_SERVER="/root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/novnc_http_ws_proxy.js"
if [[ ! -f "${NOVNC_SERVER}" ]]; then
  echo "[ERR] missing noVNC server: ${NOVNC_SERVER}" >&2
  exit 1
fi

if [[ -z "${VNC_PASS:-}" ]]; then
  VNC_PASS="$(python3 - <<'PY'
import secrets,string
alphabet=string.ascii_letters+string.digits
print(''.join(secrets.choice(alphabet) for _ in range(16)))
PY
)"
fi

if [[ "${NOVNC_BIND}" == "0.0.0.0" ]] && [[ -z "${ACCESS_TOKEN}" ]]; then
  ACCESS_TOKEN="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(18))
PY
)"
fi
TOKEN_EXPIRES_AT=0
TOKEN_EXPIRES_AT_HUMAN="never"
if [[ -n "${ACCESS_TOKEN}" ]] && [[ "${ACCESS_TOKEN_TTL_SECS}" =~ ^[0-9]+$ ]] && (( ACCESS_TOKEN_TTL_SECS > 0 )); then
  TOKEN_EXPIRES_AT="$(( $(date +%s) + ACCESS_TOKEN_TTL_SECS ))"
  TOKEN_EXPIRES_AT_HUMAN="$(python3 - <<PY
import datetime
print(datetime.datetime.utcfromtimestamp(${TOKEN_EXPIRES_AT}).strftime('%Y-%m-%d %H:%M:%S UTC'))
PY
)"
fi

x11vnc -storepasswd "${VNC_PASS}" "${PASSFILE}" >/dev/null
chmod 600 "${PASSFILE}" 2>/dev/null || true

nohup Xvfb "${DISPLAY}" -screen 0 "${GEOM}x${DEPTH}" -ac +extension RANDR \
  >"${LOGDIR}/xvfb.log" 2>&1 &
XVFB_PID=$!

# Wait for Xvfb to accept connections (fluxbox/x11vnc will fail if started too early)
XSOCK="/tmp/.X11-unix/X${DISPLAY_NUM}"
for i in $(seq 1 50); do
  [[ -S "${XSOCK}" ]] || { sleep 0.1; continue; }
  # Try a quick unix socket connect (no extra X11 utils required)
  python3 - <<PY >/dev/null 2>&1 && break || true
import socket
s=socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.settimeout(0.2)
s.connect("${XSOCK}")
PY
  sleep 0.1
done

nohup env DISPLAY="${DISPLAY}" fluxbox \
  >"${LOGDIR}/fluxbox.log" 2>&1 &
FLUX_PID=$!

nohup x11vnc -display "${DISPLAY}" \
  -rfbauth "${PASSFILE}" \
  -rfbport "${VNC_PORT}" \
  -localhost \
  -forever \
  -shared \
  -noxdamage \
  -quiet >"${LOGDIR}/x11vnc.log" 2>&1 &
X11VNC_PID=$!

# noVNC 静态资源 + WebSocket 代理同端口提供，避免：
# - ES module 严格 MIME 校验失败
# - 静态端口 / ws 端口分离导致必须拼一大串 URL 参数
nohup env NODE_PATH="${NODE_PATH}" \
  NOVNC_WEB="${NOVNC_WEB}" \
  NOVNC_PORT="${NOVNC_PORT}" \
  LISTEN_HOST="${NOVNC_BIND}" \
  VNC_PORT="${VNC_PORT}" \
  WS_PATH="/websockify" \
  ACCESS_TOKEN="${ACCESS_TOKEN}" \
  TOKEN_EXPIRES_AT="${TOKEN_EXPIRES_AT}" \
  node "${NOVNC_SERVER}" \
  >"${LOGDIR}/novnc-http-ws.log" 2>&1 &
NOVNC_PID=$!

cat >"${PIDFILE}" <<EOF
DISPLAY=${DISPLAY}
DISPLAY_NUM=${DISPLAY_NUM}
VNC_PORT=${VNC_PORT}
NOVNC_PORT=${NOVNC_PORT}
WORKDIR=${WORKDIR}
AUTO_STOP_IDLE_SECS=${AUTO_STOP_IDLE_SECS}
AUTO_STOP_CHECK_SECS=${AUTO_STOP_CHECK_SECS}
AUTO_LAUNCH_URL=${AUTO_LAUNCH_URL}
CHROME_PROFILE_DIR=${CHROME_PROFILE_DIR}
NOVNC_BIND=${NOVNC_BIND}
PUBLIC_HOST=${PUBLIC_HOST}
TOKEN_FILE=${TOKEN_FILE}
ACCESS_TOKEN_TTL_SECS=${ACCESS_TOKEN_TTL_SECS}
TOKEN_EXPIRES_AT=${TOKEN_EXPIRES_AT}
TOKEN_EXPIRES_AT_HUMAN=${TOKEN_EXPIRES_AT_HUMAN}
XVFB_PID=${XVFB_PID}
FLUX_PID=${FLUX_PID}
X11VNC_PID=${X11VNC_PID}
NOVNC_PID=${NOVNC_PID}
EOF
chmod 600 "${PIDFILE}" 2>/dev/null || true

if [[ -n "${ACCESS_TOKEN}" ]]; then
  printf '%s\n' "${ACCESS_TOKEN}" >"${TOKEN_FILE}"
  chmod 600 "${TOKEN_FILE}" 2>/dev/null || true
else
  rm -f "${TOKEN_FILE}" 2>/dev/null || true
fi

WATCHER_PID=""
if [[ "${AUTO_STOP_IDLE_SECS}" =~ ^[0-9]+$ ]] && (( AUTO_STOP_IDLE_SECS > 0 )); then
  nohup env \
    IDLE_TIMEOUT="${AUTO_STOP_IDLE_SECS}" \
    CHECK_INTERVAL="${AUTO_STOP_CHECK_SECS}" \
    bash "/root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/watch_vrd_idle.sh" "${PIDFILE}" \
    >"${LOGDIR}/idle-watcher.log" 2>&1 &
  WATCHER_PID=$!
fi
echo "WATCHER_PID=${WATCHER_PID}" >>"${PIDFILE}"

sleep 1

echo "noVNC is up."

# Try to compute a user-facing URL. Prefer public IP; fall back to the first non-loopback IP.
PUBLIC_IP=""
# Public IP (best effort; may fail if no egress)
PUBLIC_IP=$(python3 - <<'PY' 2>/dev/null || true
import urllib.request
for u in ("https://api.ipify.org", "https://ifconfig.me/ip", "https://checkip.amazonaws.com"):
    try:
        v=urllib.request.urlopen(u, timeout=2).read().decode().strip()
        if v and len(v) < 64:
            print(v)
            break
    except Exception:
        pass
PY
)
LOCAL_IP=$(python3 - <<'PY' 2>/dev/null || true
import socket
s=socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
try:
    s.connect(("8.8.8.8", 80))
    print(s.getsockname()[0])
except Exception:
    pass
finally:
    s.close()
PY
)
HOST_IP="${PUBLIC_HOST:-${PUBLIC_IP:-${LOCAL_IP}}}"
TOKEN_QS=""
if [[ -n "${ACCESS_TOKEN}" ]]; then
  TOKEN_QS="?token=${ACCESS_TOKEN}"
fi

if [[ "${NOVNC_BIND}" == "0.0.0.0" ]] && [[ -n "${HOST_IP}" ]]; then
  echo "One-click URL (public): http://${HOST_IP}:${NOVNC_PORT}/vnc.html${TOKEN_QS}"
  echo "WebSocket (public): ws://${HOST_IP}:${NOVNC_PORT}/websockify"
else
  echo "URL (local bind): http://127.0.0.1:${NOVNC_PORT}/vnc.html${TOKEN_QS}"
  echo "WebSocket (local bind): ws://127.0.0.1:${NOVNC_PORT}/websockify"
  echo "SSH tunnel example:"
  echo "  ssh -L ${NOVNC_PORT}:127.0.0.1:${NOVNC_PORT} <user>@<server>"
fi

echo "Password: ${VNC_PASS}"
if [[ -n "${ACCESS_TOKEN}" ]]; then
  echo "Access token: ${ACCESS_TOKEN}"
  echo "Token expires: ${TOKEN_EXPIRES_AT_HUMAN}"
fi
echo "DISPLAY: ${DISPLAY}"
echo "PIDFILE: ${PIDFILE}"
echo "Logs: ${LOGDIR}"
echo "Bind: ${NOVNC_BIND}"
echo "Profile: ${CHROME_PROFILE_DIR}"
if [[ -n "${WATCHER_PID}" ]]; then
  echo "Auto-stop: enabled (idle ${AUTO_STOP_IDLE_SECS}s, check ${AUTO_STOP_CHECK_SECS}s)"
else
  echo "Auto-stop: disabled"
fi
echo
echo "Launch browser:"
echo "  DISPLAY=${DISPLAY} ${CHROME_BIN} --no-sandbox --disable-dev-shm-usage --user-data-dir=${CHROME_PROFILE_DIR} --profile-directory=Default https://lp.mogic.dev/login"
echo
echo "First-use guide:"
echo "  1) Open One-click URL above"
echo "  2) Input VNC password"
echo "  3) Complete login in Chrome"
echo "  4) Verify persistence: bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/status_vrd.sh"
echo "  5) Run health check: bash /root/.openclaw/workspace/skills/virtual-remote-desktop/scripts/health_vrd.sh"

if [[ -n "${AUTO_LAUNCH_URL}" ]]; then
  nohup env DISPLAY="${DISPLAY}" ${CHROME_BIN} \
    --no-sandbox \
    --disable-dev-shm-usage \
    --user-data-dir="${CHROME_PROFILE_DIR}" \
    --profile-directory=Default \
    --new-window "${AUTO_LAUNCH_URL}" \
    >"${LOGDIR}/chrome.log" 2>&1 &
  echo "Auto-launched Chrome: ${AUTO_LAUNCH_URL}"
fi
