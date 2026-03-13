#!/bin/bash
# Demo script for Self-Healing System (safe simulation)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

clear

echo -e "${CYAN}${BOLD} OpenClaw Self-Healing System${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
sleep 1

# Step 1
echo -e "${YELLOW} Step 1: Check Gateway Status${NC}"
sleep 0.5
echo -e "$ ${BOLD}openclaw status${NC}"
sleep 0.5
echo -e "${GREEN}✓ Gateway:${NC} Running (PID 12345)"
echo -e "${GREEN}✓ Uptime:${NC} 2d 14h 32m"
echo -e "${GREEN}✓ Health:${NC} OK (HTTP 200)"
echo ""
sleep 1.5

# Step 2
echo -e "${YELLOW} Step 2: Simulate Gateway Crash${NC}"
sleep 0.5
echo -e "$ ${BOLD}kill -9 \$(pgrep openclaw)${NC}"
sleep 0.5
echo -e "${RED}✗ Gateway process terminated${NC}"
echo ""
sleep 1.5

# Step 3
echo -e "${YELLOW} Step 3: Level 1 Watchdog (180s)${NC}"
sleep 0.5
echo -e "[Watchdog] Process check... ${RED}FAILED${NC}"
echo -e "[Watchdog] Attempting restart..."
echo -e "[Watchdog] ${GREEN}Gateway restarted${NC} (PID 12346)"
echo ""
sleep 1.5

# Step 4 - Health Check
echo -e "${YELLOW} Step 4: Level 2 Health Check (300s)${NC}"
sleep 0.5
echo -e "[Health] HTTP check: localhost:18789..."
echo -e "[Health] Response: ${GREEN}200 OK${NC}"
echo -e "[Health] Gateway healthy ✓"
echo ""
sleep 1.5

# Level 3 scenario
echo -e "${YELLOW} Step 5: Level 3 Claude Doctor (if L1-L2 fail)${NC}"
sleep 0.5
echo -e "[Emergency] ${RED}Level 2 failed after 3 retries${NC}"
echo -e "[Emergency] Launching Claude Code in tmux..."
sleep 0.5
echo -e "${BLUE}┌─────────────────────────────────────────┐${NC}"
echo -e "${BLUE}│${NC}  Claude Code Diagnosis Session       ${BLUE}│${NC}"
echo -e "${BLUE}├─────────────────────────────────────────┤${NC}"
echo -e "${BLUE}│${NC} > Analyzing Gateway logs...            ${BLUE}│${NC}"
echo -e "${BLUE}│${NC} > Root cause: stale PID file           ${BLUE}│${NC}"
echo -e "${BLUE}│${NC} > Fix: rm ~/.openclaw/gateway.pid      ${BLUE}│${NC}"
echo -e "${BLUE}│${NC} > Restarting Gateway...                ${BLUE}│${NC}"
echo -e "${BLUE}│${NC} > ${GREEN}✓ Recovery successful (25s)${NC}          ${BLUE}│${NC}"
echo -e "${BLUE}└─────────────────────────────────────────┘${NC}"
echo ""
sleep 2

# Result
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD} Self-Healing Complete!${NC}"
echo ""
echo -e "${CYAN}\"AI heals AI — The system that fixes itself.\"${NC}"
echo ""
echo -e "GitHub: ${BLUE}github.com/Ramsbaby/openclaw-self-healing${NC}"
sleep 2
