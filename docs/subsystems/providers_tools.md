# Providers & Execution Tools

Manages external model inference routing and localized tool registry mapping. 

## Execution Model

```mermaid
graph TD
    A[Agent Core] --> B((Provider Factory))
    
    subgraph Providers
        B -.-> C[OpenAI / Claude APIs]
        B -.-> D[Local / vLLM / Ollama]
        B -.-> E[BitNet 1.58b]
        B -.-> F[OpenClaw RL Cluster]
    end
    
    A --> G[Tool Registry]
    G --> H{Tool Type}
    
    H -->|Command / Hardware| I[Sub-process Exec]
    H -->|Network| J[HTTP API / WebScrape]
    H -->|Sub-agent| K[Spawn Child Agent]
```

## Optimization Mechanics

- **Cached Registry Translation**: LLM APIs require comprehensive schema definitions. The Tool Registry caches these serialized `provider` structures at memory locations invalidated only upon new `Register()` calls, reducing runtime serialization penalties significantly.
- **Fallback Execution**: Providers inherently implement variable backoff timeouts natively. 
- **BitNet Interfacing**: Provides highly specific memory-optimized quantization environments for scaling up edge deployments rapidly under tight thresholds.
