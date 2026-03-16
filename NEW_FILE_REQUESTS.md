# New File Requests

## pkg/agent/skill_tools.go
- **Purpose:** Adapts `skills.DynamicTool` to `tools.Tool` interface; registers SKILL.md-declared tools in the tool registry.
- **Duplicate search:** Searched `pkg/agent/` (no skill-to-tool bridge), `pkg/skills/dynamic_tools.go` (has DynamicTool but returns DynamicToolResult, not tools.ToolResult), `pkg/tools/` (no skill adapter). No existing adapter found.

## pkg/agent/mcp_tools.go
- **Purpose:** Adapts MCP tools to `tools.Tool` interface for config-driven MCP server integration.
- **Duplicate search:** Searched `pkg/mcp/` (has client but no tool adapter), `pkg/agent/` (no MCP wiring), `pkg/tools/` (no MCP bridge). No existing adapter found.

## pkg/agent/orchestration_tool.go
- **Purpose:** `decompose` tool — breaks complex tasks into DAGs, executes via subagents, aggregates results.
- **Duplicate search:** Searched `pkg/orchestration/` (has DAG/roles/aggregator but no tool wrapper), `pkg/agent/` (has subagent tool but no DAG decomposition), `pkg/tools/` (no decompose tool). No existing decompose tool found.

## pkg/memory/temporal.go
- **Purpose:** Time-bucketed memory index with natural language time query ("last week", "yesterday").
- **Duplicate search:** Searched `pkg/memory/` (has semantic, hindsight, scoring — no temporal indexing), `pkg/agent/` (has epoch but no queryable timeline), `pkg/session/` (session history, not temporal index). No existing temporal memory found.

## pkg/agent2agent/discovery.go
- **Purpose:** UDP broadcast peer discovery for LAN-based Xagent mesh networking.
- **Duplicate search:** Searched `pkg/agent2agent/` (has protocol.go with peer registry but no auto-discovery), `pkg/` (no discovery package). No existing discovery found.

## pkg/agent2agent/router.go
- **Purpose:** Hardware-aware task routing — delegates to the most capable peer based on compute tier.
- **Duplicate search:** Searched `pkg/agent2agent/` (has Send/Broadcast but no routing logic), `pkg/hwprofile/` (detects tier but no routing). No existing router found.

## pkg/agent2agent/sync.go
- **Purpose:** Shared vault sync between peers — exports/imports world-facts and mental-models.
- **Duplicate search:** Searched `pkg/vault/` (writes locally but no sync), `pkg/agent2agent/` (has protocol but no data sync). No existing sync found.

## pkg/skills/fitness.go
- **Purpose:** Skill fitness scoring — tracks usage, success, recency to rank and deprecate skills.
- **Duplicate search:** Searched `pkg/skills/` (only `loader.go` exists — no fitness/scoring logic), `pkg/tools/` (no scoring), `pkg/agent/` (no skill metrics). No existing functionality found.

## pkg/skills/composer.go
- **Purpose:** Skill composition — combines multiple skills into composite skills via LLM, suggests pairings from provenance logs.
- **Duplicate search:** Searched `pkg/skills/` (loader has no composition), `pkg/agent/` (loop.go orchestrates but doesn't compose), `pkg/providers/` (types only). No existing functionality found.

## pkg/voice/loop.go
- **Purpose:** Voice conversation loop — continuous record→transcribe→agent→TTS→playback cycle.
- **Duplicate search:** Searched `pkg/voice/` (found `transcriber.go` for Groq STT, `tts.go` for Piper/espeak TTS — both reused; no loop orchestrator exists), `pkg/agent/` (has `ProcessDirect` but no voice loop), `cmd/xagent/` (sets up transcriber but no loop). No existing loop functionality found.

## pkg/dashboard/dashboard.go
- **Purpose:** Cognitive dashboard — embedded dark-themed web UI + JSON APIs for agent introspection (epochs, provenance, skills, peers).
- **Duplicate search:** Searched `pkg/health/` (has health/readyz/metrics endpoints but no dashboard UI), `pkg/` (no dashboard package), `cmd/xagent/` (no dashboard wiring). No existing dashboard found.

## pkg/memory/scoring_test.go
- **Purpose:** Tests for memory importance scoring — recency decay, salience detection, reference scaling, composite scoring, ranking.
- **Duplicate search:** Searched `pkg/memory/` (no existing `*_test.go` for scoring). No duplicates.

## pkg/memory/hindsight_test.go
- **Purpose:** Tests for biomimetic hindsight memory — Retain nil-vault, Recall no-backends, searchVaultFiles keyword search.
- **Duplicate search:** Searched `pkg/memory/` (no existing `*_test.go` for hindsight). No duplicates.

## pkg/memory/temporal_test.go
- **Purpose:** Tests for temporal memory index — add/query, topic filter, save/load round-trip, temporal prompt resolution, pruning.
- **Duplicate search:** Searched `pkg/memory/` (no existing `*_test.go` for temporal). No duplicates.

## pkg/sensors/monitor.go
- **Purpose:** Embodied cognition — sensor polling, rolling buffers, threshold alerts, system prompt formatting.
- **Duplicate search:** Searched `pkg/` (no sensors package), `pkg/devices/` (USB device events only, not sensor polling), `pkg/health/` (metrics but not sensor readings), `pkg/hwprofile/` (CPU/RAM detection, not external sensors). No existing sensor monitor found.

## pkg/agent/planner_test.go
- **Purpose:** Tests for Plan-Act-Reflect planner (parsePlanSteps, AdvanceStep, MarkCurrentFailed, IsComplete, ForSystemPrompt, Reflect).
- **Duplicate search:** Searched `pkg/agent/*_test.go` — found `loop_test.go`, `sleep_test.go`; no planner tests. No existing test file found.

## pkg/agent/compression_test.go
- **Purpose:** Tests for ContextCompressor.CompressHistory — short history passthrough, long history split, system message skipping.
- **Duplicate search:** Searched `pkg/agent/*_test.go` — no compression tests. No existing test file found.

## pkg/agent/personality_test.go
- **Purpose:** Tests for PersonalityTracker — Observe count/flush, Analyze insufficient/sufficient data, ForSystemPrompt, loadProfile/saveProfile round-trip.
- **Duplicate search:** Searched `pkg/agent/*_test.go` — no personality tests. No existing test file found.

## pkg/agent/dream_test.go
- **Purpose:** Tests for dream mode parsing — parseDreamResult, parseWorldModelUpdates, updateWorldModel temp-file round-trip.
- **Duplicate search:** Searched `pkg/agent/*_test.go` — no dream tests. No existing test file found.

## pkg/agent/skill_tools_test.go
- **Purpose:** Tests for skillToolAdapter — metadata delegation, Execute with echo command, missing required param error.
- **Duplicate search:** Searched `pkg/agent/*_test.go` — no skill tools tests. No existing test file found.

## pkg/agent/orchestration_tool_test.go
- **Purpose:** Tests for decomposeTool Name/Parameters, dagSpec JSON round-trip, dagNodeSpec fields.
- **Duplicate search:** Searched `pkg/agent/*_test.go` — no orchestration tool tests. No existing test file found.

## pkg/skills/fitness_test.go
- **Purpose:** Tests for skill fitness scoring — RecordUse success/failure, GetTopSkills ranking, GetDeprecated detection, Save/Load round-trip.
- **Duplicate search:** Searched `pkg/skills/*_test.go` — no fitness tests. No existing test file found.

## pkg/skills/composer_test.go
- **Purpose:** Tests for skill composition suggestions — SuggestCompositions from provenance JSONL, no co-pairs case.
- **Duplicate search:** Searched `pkg/skills/*_test.go` — no composer tests. No existing test file found.

## pkg/agent2agent/discovery_test.go
- **Purpose:** Tests for peer discovery — NewPeerDiscovery fields, empty peers, stale reaper logic.
- **Duplicate search:** Searched `pkg/agent2agent/*_test.go` — no discovery tests. No existing test file found.

## pkg/agent2agent/router_test.go
- **Purpose:** Tests for task routing — GPU keyword detection, local handling, BestPeerForTier selection.
- **Duplicate search:** Searched `pkg/agent2agent/*_test.go` — no router tests. No existing test file found.

## pkg/agent2agent/sync_test.go
- **Purpose:** Tests for vault sync — Export, Import, latest-writer-wins conflict resolution.
- **Duplicate search:** Searched `pkg/agent2agent/*_test.go` — no sync tests. No existing test file found.

## pkg/dashboard/dashboard_test.go
- **Purpose:** Tests for cognitive dashboard — HTML endpoint, state JSON, skills list via httptest.
- **Duplicate search:** Searched `pkg/dashboard/*_test.go` — no dashboard tests. No existing test file found.

## pkg/sensors/monitor_test.go
- **Purpose:** Tests for sensor monitor — creation, empty readings, empty system prompt.
- **Duplicate search:** Searched `pkg/sensors/*_test.go` — no monitor tests. No existing test file found.

## pkg/voice/loop_test.go
- **Purpose:** Tests for voice loop — construction, SetAgent binding.
- **Duplicate search:** Searched `pkg/voice/*_test.go` — no loop tests. No existing test file found.

## pkg/health/server_test.go
- **Purpose:** Tests for health server — healthz, readyz ready/not-ready, metricsz JSON keys, RegisterHandler.
- **Duplicate search:** Searched `pkg/health/*_test.go` — no server tests. No existing test file found.

## pkg/upgrade/upgrade_test.go
- **Purpose:** Tests for upgrade — UpgradeFromCheckpoint with unreachable URL error.
- **Duplicate search:** Searched `pkg/upgrade/*_test.go` — no upgrade tests. No existing test file found.

## pkg/agent/mcp_tools_test.go
- **Purpose:** Tests for MCP tool adapter — name prefixing, description fallback, parameter passthrough, Execute delegation with mock transport, error propagation, registerMCPTools integration.
- **Duplicate search:** Searched `pkg/agent/*mcp*_test.go` — no MCP tests. No existing test file found.

## pkg/tools/feedback_test.go
- **Purpose:** Tests for feedback tool — rating parsing, reward signal calculation (empty/mixed/all-negative), summary generation, recent feedback buffer.
- **Duplicate search:** Searched `pkg/tools/*feedback*_test.go` — no feedback tests. No existing test file found.

## pkg/phone/detect.go
- **Purpose:** Auto-detect Android (ADB) and iOS (libimobiledevice) phones connected via USB. Returns PhoneInfo with type, serial, model, connection state.
- **Duplicate search:** Searched `pkg/devices/` (USB hotplug events only, no phone detection), `pkg/sensors/` (GPIO/I2C polling, no phone), `pkg/tools/` (no phone/ADB tool), `pkg/` (no phone package). No existing phone detection found.

## pkg/phone/adb.go
- **Purpose:** ADB command wrapper — Run() for arbitrary commands, convenience methods (GetBattery, Screenshot, Shell, SendTap, Push/Pull, Install), deny-list for destructive commands.
- **Duplicate search:** Searched `pkg/tools/shell.go` (general shell exec, not ADB-specific — reused deny-pattern approach), `pkg/devices/` (USB events only, no ADB), `pkg/` (no ADB references). No existing ADB wrapper found.

## pkg/phone/ios.go
- **Purpose:** iOS phone access via libimobiledevice CLI tools — GetBattery, Screenshot, GetDeviceInfo, ListApps. Returns clear errors when tools are missing.
- **Duplicate search:** Searched `pkg/phone/` (new package, no iOS code), `pkg/devices/` (no iOS), `pkg/tools/` (no iOS). No existing libimobiledevice wrapper found.

## pkg/tools/phone.go
- **Purpose:** Agent-facing PhoneTool implementing tools.Tool — exposes 13 actions (status, screenshot, shell, app_list, app_launch, tap, swipe, text, push, pull, install, raw) with auto-detection of Android/iOS.
- **Duplicate search:** Searched `pkg/tools/` (has shell, browser, vision — no phone tool), `pkg/agent/` (no phone tool registration), `pkg/devices/` (events only, no tool interface). No existing phone tool found.
