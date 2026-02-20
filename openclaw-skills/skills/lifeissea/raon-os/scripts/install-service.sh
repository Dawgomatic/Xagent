#!/usr/bin/env bash
# Raon OS — launchd 서비스 설치/제거 (선택적 자동시작)
#
# 이 스크립트는 사용자가 명시적으로 실행하는 선택적 설치 스크립트입니다.
# macOS launchd에 Raon OS 서버를 등록하여 부팅 시 자동으로 시작되게 합니다.
#
# Usage:
#   scripts/install-service.sh          — launchd에 등록 및 시작
#   scripts/install-service.sh uninstall — launchd에서 제거
#
# ⚠️  주의: 이 스크립트는 macOS launchctl을 사용합니다.
#   서버 자동시작이 필요할 때만 실행하세요.
#   서버를 직접 실행하려면: raon.sh serve [port]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[raon-service]${NC} $*"; }
ok()   { echo -e "${GREEN}[raon-service]${NC} $*"; }
err()  { echo -e "${RED}[raon-service]${NC} $*" >&2; }

ACTION="${1:-install}"

case "$ACTION" in
  install)
    PLIST_SRC="$BASE_DIR/com.yeomyeonggeori.raon-os-server.plist"
    PLIST_DST="$HOME/Library/LaunchAgents/com.yeomyeonggeori.raon-os-server.plist"
    if [ ! -f "$PLIST_SRC" ]; then
      err "plist not found: $PLIST_SRC"
      exit 1
    fi
    mkdir -p "$BASE_DIR/logs"
    cp "$PLIST_SRC" "$PLIST_DST"
    launchctl load "$PLIST_DST" 2>/dev/null
    launchctl start com.yeomyeonggeori.raon-os-server 2>/dev/null
    ok "✅ Raon OS server installed and started (launchd)"
    info "   Logs: $BASE_DIR/logs/"
    ;;
  uninstall)
    PLIST_DST="$HOME/Library/LaunchAgents/com.yeomyeonggeori.raon-os-server.plist"
    launchctl stop com.yeomyeonggeori.raon-os-server 2>/dev/null || true
    launchctl unload "$PLIST_DST" 2>/dev/null || true
    rm -f "$PLIST_DST"
    ok "✅ Raon OS server uninstalled (launchd)"
    ;;
  *)
    err "Usage: $0 [install|uninstall]"
    exit 1
    ;;
esac
