#!/usr/bin/env bash
set -euo pipefail

WORKDIR="${WORKDIR:-/root/.openclaw/vrd-data}"
PIDFILE="${WORKDIR}/pids.env"

if [[ ! -f "${PIDFILE}" ]]; then
  echo "[ERR] pidfile not found: ${PIDFILE}"
  exit 1
fi

# shellcheck disable=SC1090
source "${PIDFILE}"
TOKEN_VALUE=""
if [[ -n "${TOKEN_FILE:-}" ]] && [[ -f "${TOKEN_FILE}" ]]; then
  TOKEN_VALUE="$(tr -d '\r\n' < "${TOKEN_FILE}" || true)"
fi

HOST="${PUBLIC_HOST:-127.0.0.1}"
PORT="${NOVNC_PORT:-6080}"
TOKEN_QS=""
if [[ -n "${TOKEN_VALUE}" ]]; then
  TOKEN_QS="?token=${TOKEN_VALUE}"
fi

ok() { echo "[OK] $*"; }
warn() { echo "[WARN] $*"; }
err() { echo "[ERR] $*"; }

check_pid() {
  local name="$1" pid="$2"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    ok "${name} pid ${pid} alive"
  else
    err "${name} not running"
  fi
}

check_pid "Xvfb" "${XVFB_PID:-}"
check_pid "fluxbox" "${FLUX_PID:-}"
check_pid "x11vnc" "${X11VNC_PID:-}"
check_pid "novnc" "${NOVNC_PID:-}"

HDR_HTML="$(curl -sSI "http://127.0.0.1:${PORT}/vnc.html${TOKEN_QS}" || true)"
if echo "${HDR_HTML}" | grep -qi "200"; then
  ok "vnc.html reachable"
else
  err "vnc.html unreachable"
fi

HDR_JS="$(curl -sSI "http://127.0.0.1:${PORT}/app/ui.js" || true)"
if echo "${HDR_JS}" | grep -qi "Content-Type: text/javascript"; then
  ok "module MIME text/javascript"
else
  err "module MIME invalid"
fi

if [[ -n "${TOKEN_VALUE}" ]]; then
  code_no_token="$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:${PORT}/vnc.html" || true)"
  if [[ "${code_no_token}" == "403" ]]; then
    ok "token gate enforced"
  else
    warn "token gate not enforced (code ${code_no_token})"
  fi
  if [[ -n "${TOKEN_EXPIRES_AT_HUMAN:-}" ]]; then
    ok "token expiry: ${TOKEN_EXPIRES_AT_HUMAN}"
  fi
fi

COOKIE_FILE="${CHROME_PROFILE_DIR:-}/Default/Cookies"
if [[ -f "${COOKIE_FILE}" ]]; then
  ok "chrome cookies file exists: ${COOKIE_FILE}"
else
  warn "chrome cookies file missing: ${COOKIE_FILE}"
fi

echo "health check complete"
