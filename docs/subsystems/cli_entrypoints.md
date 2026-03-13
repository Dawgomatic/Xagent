# Command Line Interface (CLI)

The Xagent entrypoint adopts a decentralized file structure for high maintainability.

## Structure Mapping

The primary `cmd/xagent/` namespace separates specific sub-commands dynamically avoiding monolothic switch parsing loops:

```mermaid
graph TD
    A[main.go] -->|Route Match| B(Subcommand Exec)
    
    B --> C[cmd_agent.go : Core Agent Loop]
    B --> D[cmd_onboard.go : Config Initializer]
    B --> E[cmd_skills.go : GitHub Registry]
    B --> F[cmd_llmcheck.go : Hardware Probes]
    B --> G[cmd_auth.go : Auth Handlers]
    B --> H[cmd_migrate.go : State Migration]
```

## Initialization Handlers

1. Primary setup validates hardware using LLMCheck modules.
2. The config module reads states dynamically, resolving paths against `~/.xagent` configs locally.
3. System daemons inject global variables via init chains natively inside respective target modules rather than polluting `main()`.
