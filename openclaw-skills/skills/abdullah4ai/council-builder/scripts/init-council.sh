#!/bin/bash
# Council Builder - Directory Initializer
# Creates the base directory structure for a new agent council
# Usage: ./init-council.sh <workspace-path> <agent-names...>
# Example: ./init-council.sh ~/.openclaw/workspace r2 leia anakin

set -e

WORKSPACE="${1:?Usage: init-council.sh <workspace-path> <agent-name> [agent-name...]}"
shift

if [ $# -eq 0 ]; then
    echo "Error: At least one agent name required"
    echo "Usage: init-council.sh <workspace-path> <agent-name> [agent-name...]"
    exit 1
fi

GREEN='\033[0;32m'
NC='\033[0m'

log() { echo -e "${GREEN}[+]${NC} $1"; }

# Create shared directories
mkdir -p "$WORKSPACE/shared/reports"
mkdir -p "$WORKSPACE/shared/learnings"
log "Created shared directories"

# Initialize cross-agent learnings file
if [ ! -f "$WORKSPACE/shared/learnings/CROSS-AGENT.md" ]; then
    cat > "$WORKSPACE/shared/learnings/CROSS-AGENT.md" << 'EOF'
# Cross-Agent Learnings

Learnings that apply across multiple agents. Any agent can write here.

---

<!-- New entries go below this line -->
EOF
    log "Created shared/learnings/CROSS-AGENT.md"
fi

# Create each agent's directory structure
for AGENT in "$@"; do
    AGENT_DIR="$WORKSPACE/agents/$AGENT"
    
    mkdir -p "$AGENT_DIR/memory"
    mkdir -p "$AGENT_DIR/.learnings"
    
    # Initialize .learnings files if they don't exist
    if [ ! -f "$AGENT_DIR/.learnings/LEARNINGS.md" ]; then
        cat > "$AGENT_DIR/.learnings/LEARNINGS.md" << 'EOF'
# Learnings Log

Corrections, knowledge gaps, and best practices.

**Statuses**: pending | in_progress | resolved | wont_fix | promoted

---

<!-- New entries go below this line -->
EOF
    fi
    
    if [ ! -f "$AGENT_DIR/.learnings/ERRORS.md" ]; then
        cat > "$AGENT_DIR/.learnings/ERRORS.md" << 'EOF'
# Errors Log

Command failures, exceptions, and unexpected behaviors.

**Statuses**: pending | in_progress | resolved | wont_fix

---

<!-- New entries go below this line -->
EOF
    fi
    
    if [ ! -f "$AGENT_DIR/.learnings/FEATURE_REQUESTS.md" ]; then
        cat > "$AGENT_DIR/.learnings/FEATURE_REQUESTS.md" << 'EOF'
# Feature Requests

Capabilities requested that don't currently exist.

**Statuses**: pending | in_progress | resolved | wont_fix

---

<!-- New entries go below this line -->
EOF
    fi
    
    log "Created agent structure: agents/$AGENT/"
done

echo ""
log "Council structure initialized with ${#@} agents"
echo "  Next: Create SOUL.md and AGENTS.md for each agent"
