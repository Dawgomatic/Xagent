# Agents Manager

 Manage Clawdbot agents: discover, profile, track capabilities, define routing hierarchy, and assign tasks.

## Features

-  **Central Registry**: Standardized `agent-registry.md` to track all agents.
-  **Strict Hierarchy**: Define `reports_to` and `can_assign_to` rules.
-  **Approval Protocol**: Handshake mechanism for secure task delegation.
-  **Agent Cards**: Standardized JSON capability cards for discovery.
-  **Visualization**: Generate Mermaid.js graphs of your agent web.
-  **Health & Stats**: Monitor agent uptime `health_check.js` and performance `log_analyzer.js`.
-  **Zero-Config**: Use `setup_wizard.js` to get started in seconds.

## Usage

### 1. Zero-Config Setup 
The easiest way to start:

```bash
node scripts/setup_wizard.js
```

### 2. Discovery
Find out what agents are available and what they can do.

```bash
node scripts/scan_agents.js
```

### 2. Validation
Make sure your registry is valid and hierarchy is sound.

```bash
node scripts/validate_registry.js
```

### 3. Hierarchy Check
Check if `Agent A` allows tasks from `Agent B`.

```bash
node scripts/can_assign.js agentA agentB
```

## Configuration

Edit `references/agent-registry.md` to configure your agents.
See `references/agent-profile-schema.md` for the full schema options including:
- `requires_approval`: Toggle manual approval.
- `auto_accept_from`: Whitelist trusted agents.

## Visualization

Generate a visual graph of your agent hierarchy:

```bash
node scripts/visualize_agents.js
```
