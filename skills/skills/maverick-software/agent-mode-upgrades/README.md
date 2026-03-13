#  Agentic Loop Upgrade

An enhanced agentic loop for [Clawdbot](https://github.com/clawdbot/clawdbot) with planning, parallel execution, confidence gates, and semantic error recovery.

![Mode Dashboard](assets/mode-dashboard.png)

##  Features

| Feature | Core Loop | Enhanced Loop |
|---------|-----------|---------------|
| **Planning** |  Reactive |  Goal decomposition with step tracking |
| **Execution** | Sequential |  Parallel (independent tools) |
| **Error Handling** | Retry-based |  Semantic recovery with alternatives |
| **Confidence** | Implicit |  Explicit gates for risky actions |
| **Context** | Overflow-triggered |  Proactive summarization |
| **State** | Implicit |  Observable FSM with checkpointing |

##  What It Does

### Planning & Reflection
The agent decomposes complex goals into step-by-step plans, tracks progress across turns, and reflects after each action to assess if steps are complete.

### Parallel Execution
Independent tools execute concurrently for faster task completion. The orchestrator identifies which tools can run in parallel.

### Confidence Gates
Before risky operations (file deletions, external messages, etc.), the system assesses confidence and can pause for approval.

### Semantic Error Recovery
When tools fail, the system diagnoses the error type and attempts alternative approaches rather than simple retries.

### Observable State Machine
Explicit state tracking enables debugging, dashboards, and checkpointing for resuming interrupted tasks.

##  Installation

### From ClawdHub
```bash
clawdbot skill install agentic-loop-upgrade
```

### Manual Installation
1. Clone/download to your skills directory:
   ```bash
   cd ~/.clawdbot/skills
   git clone https://github.com/clawdbot/skill-agentic-loop-upgrade agentic-loop-upgrade
   ```

2. Build the TypeScript:
   ```bash
   cd agentic-loop-upgrade/src
   npm install
   npm run build
   ```

3. Restart Clawdbot:
   ```bash
   clawdbot gateway restart
   ```

##  Quick Start

### Enable via Dashboard

1. Open Clawdbot Dashboard → **Agent** → **Mode**
2. Click **Enhanced Loop** card
3. Configure settings (or use defaults)
4. Click **Save Configuration**

### Disable

- Mode tab → Click **Core Loop** → Save
- Or delete: `~/.clawdbot/agents/main/agent/enhanced-loop-config.json`

##  Configuration

All settings are available in the Mode dashboard:

### Planning & Reflection
- **Enable Planning**: Generate execution plans before complex tasks
- **Reflection After Tools**: Assess progress after each tool execution
- **Max Plan Steps**: Maximum steps in a generated plan (2-15)

### Execution
- **Parallel Tools**: Execute independent tools concurrently
- **Max Concurrent**: Maximum parallel tool executions (1-10)
- **Confidence Gates**: Assess confidence before risky actions
- **Confidence Threshold**: Minimum confidence to proceed (30-95%)

### Context Management
- **Proactive Management**: Summarize and prune before overflow
- **Summarize After N Iterations**: Trigger summarization interval
- **Context Threshold**: Context fill level to trigger management

### Error Recovery
- **Semantic Recovery**: Diagnose errors and adapt approach
- **Max Recovery Attempts**: Maximum alternative attempts (1-5)
- **Learn From Errors**: Store successful recoveries for future use

### State Machine
- **Enable State Machine**: Track agent state transitions
- **State Logging**: Log all state transitions
- **Metrics Collection**: Collect timing metrics per state

### Orchestrator Model
Select a cost-effective model for planning/reflection calls (e.g., Claude Sonnet 4.5).

##  File Structure

```
~/.clawdbot/
├── agents/main/agent/
│   └── enhanced-loop-config.json    # Configuration
├── agent-state/                      # Persistent plan state
│   └── {sessionId}.json
└── checkpoints/                      # Checkpoint files
    └── {sessionId}/
        └── ckpt_*.json
```

##  For Developers

### Programmatic Usage

```typescript
import { createOrchestrator } from "@clawdbot/enhanced-loop";

const orchestrator = createOrchestrator({
  sessionId: "session_123",
  planning: { enabled: true, maxPlanSteps: 7 },
  approvalGate: { enabled: true, timeoutMs: 15000 },
  retry: { enabled: true, maxAttempts: 3 },
  context: { enabled: true, thresholdTokens: 80000 },
  checkpoint: { enabled: true },
}, {
  onPlanCreated: (plan) => console.log("Plan:", plan.goal),
  onStepCompleted: (id, result) => console.log("✓", result),
});

await orchestrator.init();
```

### Architecture

See [SKILL.md](./SKILL.md) for full technical documentation.

##  Notes

- **Token overhead**: Planning and reflection use additional tokens (configurable via orchestrator model selection)
- **Easy rollback**: One click to switch back to Core Loop
- **Checkpoints**: Long tasks can be resumed if interrupted

##  Documentation

- [SKILL.md](./SKILL.md) - Full technical documentation
- [INSTRUCTIONS.md](./INSTRUCTIONS.md) - Integration guide for agents
- [references/](./references/) - Component documentation

##  Links

- [Clawdbot](https://github.com/clawdbot/clawdbot)
- [ClawdHub](https://clawdhub.com)
- [Documentation](https://docs.clawd.bot)
- [Discord](https://discord.com/invite/clawd)

##  License

MIT
