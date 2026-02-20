# Enhanced Agentic Loop Skill

A comprehensive upgrade to Clawdbot's agentic capabilities with persistent state, automatic planning, approval gates, retry logic, context management, and checkpointing.

## Status: ✅ Active

All components are integrated and working.

| Component | Status |
|-----------|--------|
| Mode Dashboard UI | ✅ Working |
| Configuration System | ✅ Working |
| Hook/Wrapper Integration | ✅ Working |
| State Machine | ✅ Working |
| Planning Layer | ✅ Working |
| Parallel Execution | ✅ Working |
| Confidence Gates | ✅ Working |
| Error Recovery | ✅ Working |
| Checkpointing | ✅ Working |

## Features

### 1. Persistent Plan State
Plans survive across conversation turns. The agent knows where it left off.

```typescript
import { getStateManager } from "@clawdbot/enhanced-loop";

const state = getStateManager();
await state.init(sessionId);

// Plan persists in ~/.clawdbot/agent-state/{sessionId}.json
state.setPlan(plan);
state.completeStep("step_1", "Files created");
const progress = state.getProgress(); // { completed: 1, total: 5, percent: 20 }
```

### 2. Automatic Step Completion Detection
Analyzes tool results to determine if plan steps are complete.

```typescript
import { createStepTracker } from "@clawdbot/enhanced-loop";

const tracker = createStepTracker(stateManager);

// After each tool execution
const analysis = await tracker.analyzeToolResult(tool, result);
if (analysis.isComplete) {
  console.log(`Step done: ${analysis.suggestedResult}`);
}
```

### 3. Tool Approval Gates with Timeout
Risky operations pause for human approval, but auto-proceed after N seconds.

```typescript
import { getApprovalGate } from "@clawdbot/enhanced-loop";

const gate = getApprovalGate({
  enabled: true,
  timeoutMs: 15000, // 15 seconds to respond
  requireApprovalFor: ["high", "critical"],
  onApprovalNeeded: (request) => {
    // Notify user: "⚠️ Approve rm -rf? Auto-proceeding in 15s..."
  },
});

// Before risky tool execution
if (gate.requiresApproval(tool)) {
  const result = await gate.requestApproval(tool);
  if (!result.proceed) {
    return { blocked: true, reason: result.request.riskReason };
  }
}

// User can respond with:
gate.approve(requestId);  // Allow it
gate.deny(requestId);     // Block it
// Or wait for timeout → auto-proceeds
```

**Risk Levels:**
- `low`: Read operations (auto-approved)
- `medium`: Write/Edit, safe exec
- `high`: Messages, browser actions, git push
- `critical`: rm -rf, database drops, format commands

### 4. Automatic Retry with Alternatives
Failed tools get diagnosed and retried with modified approaches.

```typescript
import { createRetryEngine } from "@clawdbot/enhanced-loop";

const retry = createRetryEngine({
  enabled: true,
  maxAttempts: 3,
  retryDelayMs: 1000,
});

const result = await retry.executeWithRetry(tool, executor);
// Automatically:
// - Diagnoses errors (permission, network, not_found, etc.)
// - Applies fixes (add sudo, increase timeout, etc.)
// - Retries with exponential backoff
```

### 5. Context Summarization
Automatically summarizes old messages when context grows long.

```typescript
import { createContextSummarizer } from "@clawdbot/enhanced-loop";

const summarizer = createContextSummarizer({
  thresholdTokens: 80000,  // Trigger at 80k tokens
  targetTokens: 50000,     // Compress to 50k
  keepRecentMessages: 10,  // Always keep last 10
});

if (summarizer.needsSummarization(messages)) {
  const result = await summarizer.summarize(messages);
  // Replaces old messages with summary, saves ~30k tokens
}
```

### 6. Checkpoint/Restore
Save and resume long-running tasks across sessions.

```typescript
import { getCheckpointManager } from "@clawdbot/enhanced-loop";

const checkpoints = getCheckpointManager();

// Create checkpoint
const ckpt = await checkpoints.createCheckpoint(state, {
  description: "After step 3",
  trigger: "manual",
});

// Later: check for incomplete work
const incomplete = await checkpoints.hasIncompleteWork(sessionId);
if (incomplete.hasWork) {
  console.log(incomplete.description);
  // "Incomplete task: Build website (3/6 steps, paused 2.5h ago)"
}

// Resume
const restored = await checkpoints.restore(sessionId);
// Injects context: "Resuming from checkpoint... [plan status]"
```

## Unified Orchestrator

The recommended way to use all features together:

```typescript
import { createOrchestrator } from "@clawdbot/enhanced-loop";

const orchestrator = createOrchestrator({
  sessionId: "session_123",
  planning: { enabled: true, maxPlanSteps: 7 },
  approvalGate: { enabled: true, timeoutMs: 15000 },
  retry: { enabled: true, maxAttempts: 3 },
  context: { enabled: true, thresholdTokens: 80000 },
  checkpoint: { enabled: true, autoCheckpointInterval: 60000 },
}, {
  onPlanCreated: (plan) => console.log("Plan:", plan.goal),
  onStepCompleted: (id, result) => console.log("✓", result),
  onApprovalNeeded: (req) => notifyUser(req),
  onCheckpointCreated: (id) => console.log("📍 Checkpoint:", id),
});

// Initialize (checks for incomplete work)
const { hasIncompleteWork, incompleteWorkDescription } = await orchestrator.init();

// Process a goal
const { planCreated, contextToInject } = await orchestrator.processGoal(
  "Build a REST API with authentication"
);

// Execute tools with all enhancements
const result = await orchestrator.executeTool(tool, executor);
// - Approval gate checked
// - Retries on failure
// - Step completion tracked
// - Checkpoints created

// Get status for display
const status = orchestrator.getStatus();
// { hasPlan: true, progress: { completed: 2, total: 5, percent: 40 }, ... }
```

## Mode Dashboard Integration

The skill includes a Mode tab for the Clawdbot Dashboard:

**Location:** Agent > Mode

**Features:**
- Toggle between Core Loop and Enhanced Loop
- Configure all settings visually
- Select orchestrator model from the Clawdbot model catalog (for cost control)
- Real-time configuration preview

## Clawdbot Integration

The skill integrates via the enhanced-loop-hook in Clawdbot:

1. **Config file:** `~/.clawdbot/agents/main/agent/enhanced-loop-config.json`

2. **Automatic activation:** When enabled, the hook:
   - Detects planning intent in user messages
   - Injects plan context into system prompt (additive; does not replace or override existing system prompt policies)
   - Tracks tool executions and step progress
   - Creates checkpoints automatically
   - Offers to resume incomplete tasks

## Credentials and Security

- **No additional API keys required.** The orchestrator reuses the host Clawdbot agent's existing auth profiles (via `resolveApiKeyForProvider`). It prefers `api_key` type profiles over OAuth tokens for compatibility with direct API calls.
- **Orchestrator model is dynamically selectable** via the Mode dashboard. The dropdown is populated from the Clawdbot model catalog (`models.list`), so any model the agent can use is available. Pick a smaller model for planning/reflection calls to minimize costs.
- **No external network calls** beyond the configured LLM provider API (e.g. `api.anthropic.com`). The skill does not phone home or send telemetry.
- **Persistence is local only.** Plan state, checkpoints, and configuration are written to `~/.clawdbot/` under the agent directory. No cloud storage.
- **System prompt modification is additive.** The hook appends plan context and step progress to the agent's `extraSystemPrompt` field. It does not replace, remove, or override the core system prompt or any safety policies.
- **The wrapper is transparent.** The `wrapRun` function always calls the original agent runner. It adds orchestration (planning, context injection, tracking) around the original call but never bypasses it.

## Intent Detection

Planning automatically triggers on:

**Explicit intent:**
- "plan...", "help me...", "how should I..."
- "figure out...", "walk me through..."
- "what's the best way...", "I need to..."

**Complex tasks:**
- Complex verb + task noun: "build API", "create site"
- Sequential language: "first... then..."
- Scope words: "full", "complete", "from scratch"

## File Structure

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

## Source Structure

```
src/
├── index.ts                 # Main exports
├── orchestrator.ts          # Unified orchestrator
├── types.ts                 # Type definitions
├── clawdbot-hook.ts         # Clawdbot integration hook
├── enhanced-loop.ts         # Core loop wrapper
├── planning/
│   └── planner.ts           # Plan generation
├── execution/
│   ├── approval-gate.ts     # Approval gates
│   ├── confidence-gate.ts   # Confidence assessment
│   ├── error-recovery.ts    # Semantic error recovery
│   ├── parallel.ts          # Parallel execution
│   └── retry-engine.ts      # Retry with alternatives
├── context/
│   ├── manager.ts           # Context management
│   └── summarizer.ts        # Context summarization
├── state/
│   ├── persistence.ts       # Plan state persistence
│   ├── step-tracker.ts      # Step completion tracking
│   └── checkpoint.ts        # Checkpointing
├── state-machine/
│   └── fsm.ts               # Observable state machine
├── tasks/
│   └── task-stack.ts        # Task hierarchy
└── llm/
    └── caller.ts            # LLM abstraction for orchestrator
```

## UI Structure

```
ui/
├── views/
│   └── mode.ts              # Mode page view (Lit)
└── controllers/
    └── mode.ts              # Mode page controller
```

## Version

v1.0.0 - Full agentic loop with Mode dashboard UI
