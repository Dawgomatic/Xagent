#!/usr/bin/env bash
set -euo pipefail

WORKDIR="${WORKDIR:-/root/.openclaw/vrd-data}"
PIDFILE="${WORKDIR}/pids.env"

is_alive() {
  local pid="$1"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    echo "up (${pid})"
  else
    echo "down"
  fi
}

if [[ ! -f "${PIDFILE}" ]]; then
  echo "status: down (no pidfile)"
  exit 1
fi

# shellcheck disable=SC1090
source "${PIDFILE}"
TOKEN_VALUE=""
if [[ -n "${TOKEN_FILE:-}" ]] && [[ -f "${TOKEN_FILE}" ]]; then
  TOKEN_VALUE="$(tr -d '\r\n' < "${TOKEN_FILE}" || true)"
fi

COOKIE_FILE="${CHROME_PROFILE_DIR:-}/Default/Cookies"
LOGIN_STATE="missing"
if [[ -f "${COOKIE_FILE}" ]]; then
  LOGIN_STATE="present"
fi
URL_HOST="${PUBLIC_HOST:-127.0.0.1}"
if [[ "${NOVNC_BIND:-127.0.0.1}" == "0.0.0.0" ]] && [[ -z "${PUBLIC_HOST:-}" ]]; then
  URL_HOST="0.0.0.0"
fi
TOKEN_QS=""
if [[ -n "${TOKEN_VALUE}" ]]; then
  TOKEN_QS="?token=${TOKEN_VALUE}"
fi
TOKEN_LABEL="disabled"
if [[ -n "${TOKEN_VALUE}" ]]; then
  TOKEN_LABEL="${TOKEN_VALUE:0:6}...${TOKEN_VALUE: -4}"
fi

echo "status:"
echo "  Xvfb:      $(is_alive "${XVFB_PID:-}")"
echo "  fluxbox:   $(is_alive "${FLUX_PID:-}")"
echo "  x11vnc:    $(is_alive "${X11VNC_PID:-}")"
echo "  novnc:     $(is_alive "${NOVNC_PID:-}")"
echo "  watcher:   $(is_alive "${WATCHER_PID:-}")"
echo "  display:   ${DISPLAY:-unknown}"
echo "  vnc_port:  ${VNC_PORT:-unknown}"
echo "  bind:      ${NOVNC_BIND:-unknown}"
echo "  profile:   ${CHROME_PROFILE_DIR:-unknown}"
echo "  cookies:   ${LOGIN_STATE} (${COOKIE_FILE})"
echo "  token:     ${TOKEN_LABEL}"
echo "  token_exp: ${TOKEN_EXPIRES_AT_HUMAN:-unknown}"
echo "  auto_stop: idle=${AUTO_STOP_IDLE_SECS:-0}s check=${AUTO_STOP_CHECK_SECS:-0}s"
echo "  novnc_url: http://${URL_HOST}:${NOVNC_PORT:-6080}/vnc.html${TOKEN_QS}"
echo "  ws_url:    ws://${URL_HOST}:${NOVNC_PORT:-6080}/websockify"
