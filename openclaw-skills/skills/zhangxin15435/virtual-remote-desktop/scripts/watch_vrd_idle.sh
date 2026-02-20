#!/usr/bin/env bash
set -euo pipefail

PIDFILE="${1:-/root/.openclaw/vrd-data/pids.env}"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-900}"
CHECK_INTERVAL="${CHECK_INTERVAL:-15}"

if [[ ! "${IDLE_TIMEOUT}" =~ ^[0-9]+$ ]] || [[ ! "${CHECK_INTERVAL}" =~ ^[0-9]+$ ]]; then
  echo "[ERR] IDLE_TIMEOUT/CHECK_INTERVAL must be integers" >&2
  exit 1
fi
if (( IDLE_TIMEOUT <= 0 )) || (( CHECK_INTERVAL <= 0 )); then
  exit 0
fi

if [[ ! -f "${PIDFILE}" ]]; then
  echo "[WARN] PIDFILE not found: ${PIDFILE}"
  exit 0
fi

# shellcheck disable=SC1090
source "${PIDFILE}"

SELF_PID=$$
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
STOP_SCRIPT="${SCRIPT_DIR}/stop_vrd.sh"
idle_for=0

while true; do
  if [[ ! -f "${PIDFILE}" ]]; then
    exit 0
  fi

  # shellcheck disable=SC1090
  source "${PIDFILE}"

  if [[ -n "${NOVNC_PID:-}" ]] && ! kill -0 "${NOVNC_PID}" 2>/dev/null; then
    exit 0
  fi
  if [[ -n "${SELF_PID}" ]] && ! kill -0 "${SELF_PID}" 2>/dev/null; then
    exit 0
  fi

  conn_count="$(ss -Htan state established "( sport = :${NOVNC_PORT} )" 2>/dev/null | wc -l | tr -d ' ')"
  if [[ ! "${conn_count}" =~ ^[0-9]+$ ]]; then
    conn_count=0
  fi

  if (( conn_count > 0 )); then
    idle_for=0
  else
    idle_for=$((idle_for + CHECK_INTERVAL))
  fi

  if (( idle_for >= IDLE_TIMEOUT )); then
    echo "[INFO] noVNC idle for ${idle_for}s, auto stopping..."
    WORKDIR="${WORKDIR:-/root/.openclaw/vrd-data}" bash "${STOP_SCRIPT}" || true
    exit 0
  fi

  sleep "${CHECK_INTERVAL}"
done
