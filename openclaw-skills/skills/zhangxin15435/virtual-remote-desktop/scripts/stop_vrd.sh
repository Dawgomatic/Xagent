#!/usr/bin/env bash
set -euo pipefail

WORKDIR="${WORKDIR:-/root/.openclaw/vrd-data}"
PIDFILE="${WORKDIR}/pids.env"
KILL_BROWSER="${KILL_BROWSER:-0}"
SELF_PID=$$

kill_if_alive() {
  local pid="$1"
  if [[ -n "${pid}" ]] && [[ "${pid}" != "${SELF_PID}" ]] && kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}" 2>/dev/null || true
  fi
}

if [[ -f "${PIDFILE}" ]]; then
  # shellcheck disable=SC1090
  source "${PIDFILE}"
  kill_if_alive "${WATCHER_PID:-}"
  # 兼容旧版本（曾经用 python http.server 起静态服务）
  kill_if_alive "${HTTP_PID:-}"
  kill_if_alive "${NOVNC_PID:-}"
  kill_if_alive "${X11VNC_PID:-}"
  kill_if_alive "${FLUX_PID:-}"
  kill_if_alive "${XVFB_PID:-}"
  if [[ "${KILL_BROWSER}" == "1" ]] && [[ -n "${CHROME_PROFILE_DIR:-}" ]]; then
    pkill -f "google-chrome.*--user-data-dir=${CHROME_PROFILE_DIR}" 2>/dev/null || true
  fi
  rm -f "${TOKEN_FILE:-}" 2>/dev/null || true
  rm -f "${PIDFILE}"
else
  echo "[WARN] PIDFILE not found: ${PIDFILE}"
fi

sleep 1

# Fallback cleanup for stale processes
pkill -f "x11vnc -display :${DISPLAY_NUM:-99}" 2>/dev/null || true
pkill -f "Xvfb :${DISPLAY_NUM:-99}" 2>/dev/null || true
pkill -f "websockify.*:${NOVNC_PORT:-6080}" 2>/dev/null || true
pkill -f "http\\.server.*${NOVNC_PORT:-6080}" 2>/dev/null || true
pkill -f "novnc_http_ws_proxy\\.js" 2>/dev/null || true

echo "Stopped virtual remote desktop stack."
