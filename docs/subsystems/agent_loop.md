# Agent Core & Cognition Loop

The Agent Loop represents the reasoning layer of Xagent, managing input context, planning operations, executing tools (via the Action phase), and retaining learned insights.

## System Diagram

```mermaid
graph TD
    A[Inbound Message] -->|MessageBus| B(AgentLoop)
    
    subgraph Pre-Processing
        B --> C[ContextBuilder]
        C -->|Extract Context| C1[Cached Bootsraps]
        C -->|Recall Context| C2[Semantic Memory]
        C -->|Extract Plan| C3(Planner)
    end
    
    subgraph Execution Loop
        D[LLM Request]
        C1 --> D
        C2 --> D
        C3 --> D
        D --> E{Tool Request?}
        E -->|Yes| F[Tool Middleware]
        F -->|Return Data| D
        E -->|No| G(Final Response)
    end
    
    subgraph Post-Processing
        G --> H[Hindsight Retention]
        H --> I[Vault Writer]
    end
    
    G --> J[Outbound Message Bus]
```

## Key Mechanisms

- **ContextBuilder**: Merges static persona instructions (`AGENTS.md`) with dynamic state (Epochs, Hindsight experiences). Implements aggressive memory caching to minimize disk I/O per turn.
- **Plan-Act-Reflect**: Integrated directly into `loop.go`. If a complex prompt is detected, the `Planner` generates discrete sub-steps that are embedded in the primary system context, guiding the agent's multi-hop tool execution.
- **Token Summarization**: Implements runtime token checking (via `utf8.runeCount`) to dynamically truncate and summarize the active message history window.
