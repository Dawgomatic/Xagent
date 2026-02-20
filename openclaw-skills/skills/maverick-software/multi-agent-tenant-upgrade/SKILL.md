# agent-chat-ux

**name:** agent-chat-ux  
**version:** 1.1.0  
**author:** Charles Sears  
**description:** Upgrades the OpenClaw Control UI chat and agent experience — agent selector dropdown in chat, per-agent session filtering, new session button, Create Agent wizard (manual + AI), emoji picker, model selector improvements, and backend agent CRUD methods.

---

## What This Skill Adds

### 1. Agent Selector Dropdown in Chat Header
When multiple agents are configured, a dropdown appears **left of the session dropdown** in the chat header. Selecting an agent switches to that agent's most recent session (or falls back to a fresh webchat key for that agent). The session dropdown automatically filters to show **only sessions belonging to the selected agent**.

### 2. Per-Agent Session Filtering (Sorted Newest First)
Sessions are now scoped to the active agent and sorted newest-first. No more mixing other agents' cron jobs and subagent sessions into the current chat's session picker.

### 3. + New Session Button in Chat Header
A `+` icon button sits right of the session dropdown, allowing new sessions to be started without typing `/new`.

### 4. Create Agent Panel (Manual + AI Wizard)
The Agents tab gains a **+ Create Agent** button that expands a panel with two modes:

**Manual mode:**
- Agent name
- Workspace path (auto-generated from name if left blank)
- Emoji picker (see below)

**AI Wizard mode:**
- Describe the agent in plain English
- Click "Generate Agent" — AI generates name, emoji, and full SOUL.md
- Review the preview, then click "✅ Create This Agent"

After creation, the agents list **and** config form are both refreshed automatically — no "not found in config" error, no manual reload needed.

### 5. Emoji Picker Dropdown
The emoji field in the Create Agent form is now a **dropdown with 60 curated emojis** grouped into categories (Tech & AI, People & Roles, Animals, Nature & Elements, Objects & Symbols), each showing the emoji and its name. A large live preview shows the selected emoji next to the dropdown.

### 6. Agents Tab — Model Selector Cleanup
- Removed the redundant read-only "Primary Model" row from the Overview grid (it's already editable in the Model Selection section below)
- **Fallback models** converted from a free-text comma-separated input to a proper **`<select multiple>`** using the same full model catalog as the primary selector
- Added spacing and clear labels between primary and fallback fields
- Small hint "(hold Ctrl/⌘ to select multiple)" on the fallback selector

### 7. Backend — `agents.create` / `agents.update` / `agents.delete` / `agents.wizard`
New RPC handlers wired into the gateway:

| Method | Description |
|--------|-------------|
| `agents.create` | Provisions a new agent entry in config + scaffolds workspace (SOUL.md, AGENTS.md, USER.md) |
| `agents.update` | Patches agent config (name, workspace, model, identity, etc.) |
| `agents.delete` | Removes agent from config |
| `agents.wizard` | Calls the configured LLM to generate name, emoji, and SOUL.md from a plain-text description |

**Auth fix in `agents.wizard`:** Raw HTTP calls to the model API require an `api_key` token, not an OAuth/bearer token. The wizard now falls back to an explicit `api_key` profile (or `ANTHROPIC_API_KEY` env var) when the default resolved auth mode is `oauth` or `token`.

---

## Files Changed

| File | Change |
|------|--------|
| `ui/src/ui/app-render.helpers.ts` | Agent dropdown, new session button, per-agent session filter |
| `ui/src/ui/views/agents.ts` | Create Agent panel (manual + wizard), emoji picker, model display cleanup |
| `ui/src/ui/views/agents-utils.ts` | `buildModelOptionsMulti()` for multi-select fallback dropdown |
| `ui/src/ui/app-render.ts` | Wires create/wizard props; reloads configForm after agent creation |
| `ui/src/ui/app.ts` | State fields for create-agent form (name, workspace, emoji, wizard) |
| `src/gateway/server-methods/agents.ts` | Implementation of all agent CRUD + wizard methods, with OAuth→api_key auth fallback |

---

## Installation

This skill requires patching OpenClaw source files and a UI + gateway rebuild.

### Prerequisites
- OpenClaw source at `~/openclaw` (fork or local clone)
- `pnpm` installed (`npm install -g pnpm`)
- Node.js 20+

### Step 1: Apply patches

```bash
cd ~/openclaw

git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/agents-view.txt
git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/agents-utils.txt
git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/app-render-helpers.txt
git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/app-render.txt
git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/app-main.txt
git apply ~/.openclaw/workspace/skills/agent-chat-ux/references/server-agents.txt
```

If any patch fails due to upstream drift, apply manually using the patch file as a line-by-line reference.

### Step 2: Rebuild UI

```bash
cd ~/openclaw
pnpm ui:build
```

### Step 3: Rebuild gateway (for backend agent methods)

```bash
cd ~/openclaw
pnpm build
```

### Step 4: Restart gateway

```bash
openclaw gateway restart
```

### Step 5: Verify

1. Open Control UI at `http://localhost:18789`
2. **Chat tab** — agent dropdown appears left of session dropdown (if >1 agent configured); `+` button appears right of session dropdown
3. **Agents tab** — "+ Create Agent" button with Manual and AI Wizard modes
4. **Agents → Overview → Model Selection** — fallback is now a multi-select dropdown
5. Create an agent with the AI Wizard — should generate cleanly and appear in the list with no "not found" error

---

## Usage

### Chat: Switching Agents & Sessions
- **Agent dropdown** (left of session): picks the agent; session list updates to show only that agent's sessions
- **Session dropdown**: switches between existing conversations for the selected agent, newest first
- **`+` button**: starts a new session for the current agent (same as `/new`)

### Agents: Create Agent
1. Click **+ Create Agent**
2. **Manual:** enter name, workspace, pick emoji → "Create Agent"
3. **AI Wizard:** describe the agent → "Generate Agent" → review preview → "✅ Create This Agent"

### Agents: Fallback Models
In Model Selection:
- **Primary model** — single dropdown
- **Fallback models** — multi-select (`Ctrl`/`⌘` + click for multiple); these are retried in order when the primary model fails (rate limit, context overflow, etc.)

---

## Architecture Notes

### Session Key Format
`agent:<agentId>:<rest>` — the agent selector reads `parseAgentSessionKey(state.sessionKey).agentId` to determine the active agent and filters the session list accordingly.

### Config Refresh After Creation
After `agents.create` succeeds, the UI calls both `agents.list` (to update the sidebar) and `loadConfig` (to refresh `configForm`). Without the `loadConfig` call, selecting the new agent would show "not found in config" because the config form was stale.

### Wizard Auth Resolution
`agents.wizard` makes a direct HTTP call to the model provider API. Raw HTTP calls require an `api_key` type credential — not an OAuth bearer token. The resolution order is:
1. Default `resolveApiKeyForProvider` result (used if mode is `api-key`)
2. First `api_key`-type profile in the auth store for the provider
3. `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` env var directly

This mirrors the same pattern used in `enhanced-loop-hook.ts`.

### Model Fallbacks
Stored as `model.fallbacks[]` in the agent config. The runtime tries them via `runWithModelFallback()` when the primary model returns an error.

---

## Changelog

### 1.1.0 (2026-02-18)
- **Fix:** AI Wizard 401 error — OAuth token was being passed as `x-api-key`; now falls back to `api_key` profile or env var
- **Fix:** "Agent not found in config" after creation — `loadConfig` now called after `agents.create` in both Manual and Wizard paths
- **New:** Emoji picker dropdown (60 emojis, 5 categories, live preview) replaces free-text emoji input
- Patches refreshed with all fixes included

### 1.0.0 (2026-02-18)
- Initial release
- Agent selector dropdown in chat header
- Per-agent session filtering (newest-first)
- New session button (`+`) in chat header
- Create Agent panel — Manual + AI Wizard modes
- Fallback model multi-select dropdown
- Removed duplicate "Primary Model" display from Agents overview
- `agents.create` / `agents.update` / `agents.delete` / `agents.wizard` backend methods
