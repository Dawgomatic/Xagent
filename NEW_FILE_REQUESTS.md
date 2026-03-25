# New File Requests

## cmd/xagent/workspace/GOALS.md
- **Purpose:** Seed `GOALS.md` in the default workspace so the dashboard Overview and `goals` tool have initial active goals; users edit or replace via tool or file.
- **Duplicate search:** `pkg/tools/goals.go` (`defaultGoalsMarkdown()` empty sections only), `glob **/GOALS.md` under repo (none in `cmd/xagent/workspace` before this), `IDENTITY.md` has a generic "Goals" bullet list but not the EXA checkbox format. No duplicate tracked workspace goals file.

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

## pkg/dashboard/dashboard_html.go
- **Purpose:** Interactive dashboard HTML — tabbed SPA with memory editor, vault browser, epoch/provenance detail views, config viewer, and system metrics. Extracted from dashboard.go to keep Go handlers separate from the HTML const.
- **Duplicate search:** Searched `pkg/dashboard/` (dashboard.go had inline `dashboardHTML` const — replaced and moved to separate file), `pkg/` (no other HTML templates). No duplicate found.

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

## pkg/health/watchdog.go
- **Purpose:** Subsystem watchdog — periodically checks all registered subsystems (Ollama, Qdrant, dashboard, heartbeat, agent loop), reports status, and auto-recovers external services via systemd restart. Exposes `GetStatus()` for the dashboard `/api/watchdog` endpoint.
- **Duplicate search:** Searched `pkg/health/` (has `server.go` with `/healthz`, `/readyz`, `OllamaChecker` — no watchdog or subsystem-level monitoring), `pkg/hwprofile/` (has `WatchResources` for tier changes — different purpose, not subsystem health), `pkg/sensors/` (hardware sensor polling, not service monitoring), `cmd/xagent/` (no watchdog or process monitor). No existing watchdog found.

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

## pkg/tools/skills_tool.go
- **Purpose:** Agent-facing skills tool — search the 10K+ embedded catalog and install skills at runtime. Actions: search (keyword search), install (download from GitHub), list (show installed).
- **Duplicate search:** Searched `pkg/tools/` (no skills tool), `pkg/agent/loop.go` (no skill search/install tool registered), `pkg/skills/` (has AutoDiscoverer and Installer but no tools.Tool wrapper). No existing skills tool found.

## pkg/tools/usb.go
- **Purpose:** USB device enumeration tool — lists connected USB devices via `lsusb` or `/sys/bus/usb/devices/` sysfs. Actions: list (enumerate all devices), detail (show specific device info). Enables the agent to see what's physically connected.
- **Duplicate search:** Searched `pkg/tools/` (has i2c.go, spi.go for bus tools; phone.go for ADB phone access — no generic USB enumeration), `pkg/devices/` (USB hotplug events but no enumeration tool), `pkg/` (no USB listing tool). No existing USB tool found.

## pkg/sensors/source.go
- **Purpose:** SensorSource interface — abstraction for any sensor data provider (phone, host, I2C, USB serial). Defines Poll(), Available(), Interval(), Name().
- **Duplicate search:** Searched `pkg/sensors/monitor.go` (had stub readSensor but no interface), `pkg/devices/` (USB events, not sensor abstraction), `pkg/tools/i2c.go` (I2C tool, not source interface). No existing sensor source interface found.

## pkg/sensors/phone_source.go
- **Purpose:** Phone sensor source — reads battery, temperature, light, accelerometer, screen state, pressure from Android phone via ADB shell commands. Parallel reads with graceful failure per sensor.
- **Duplicate search:** Searched `pkg/sensors/` (monitor.go had stub readSensor returning 0), `pkg/tools/phone.go` (phone tool for user-triggered actions, not continuous polling), `pkg/phone/adb.go` (ADB wrapper, no sensor polling). No existing phone sensor polling found.

## pkg/sensors/xavier_source.go
- **Purpose:** Xavier/host sensor source — reads thermal zones (CPU/GPU/board), RAM usage, disk usage, load average, Tegra GPU load from sysfs/proc.
- **Duplicate search:** Searched `pkg/hwprofile/hwprofile.go` (has readDiskFree, readMemInfo — reused syscall.Statfs pattern), `pkg/sensors/` (no host readings), `pkg/health/` (metrics counters, not hardware sensors). Reused: `pkg/hwprofile/hwprofile.go` L312-L324 for syscall.Statfs pattern.

## pkg/sensors/discovery.go
- **Purpose:** Sensor source auto-discovery — probes system for ADB phones, I2C buses, GPIO chips, USB serial devices. Returns all available SensorSources. Periodic re-discovery for hot-plug.
- **Duplicate search:** Searched `pkg/sensors/` (no discovery), `pkg/devices/service.go` (USB hotplug events, not sensor source discovery), `pkg/skills/autodiscover.go` (skill catalog discovery, different domain). No existing sensor discovery found.

## pkg/sensors/camera_source.go
- **Purpose:** Camera sensor source — captures frames from USB webcams (V4L2/ffmpeg), CSI cameras (Tegra), and phone cameras (ADB screencap/camera intent). Optionally analyzes frames via Ollama vision model (moondream) for ambient scene description.
- **Duplicate search:** Searched `pkg/tools/vision.go` (Ollama vision tool for agent-triggered analysis, not continuous polling — reused API pattern), `pkg/tools/phone.go` (phone screenshot action — reused screencap pattern), `pkg/phone/adb.go` (Screenshot method — reused as reference), `pkg/sensors/` (no camera source). Reused: Ollama /api/generate pattern from `pkg/tools/vision.go`, ADB screencap pattern from `pkg/phone/adb.go` L195-L201.

## pkg/sensors/i2c_source.go
- **Purpose:** I2C sensor source — scans I2C buses with i2cdetect, auto-identifies known sensors (BMP280, TMP102, EEPROM, MPU6050, etc.), reads data from supported sensors.
- **Duplicate search:** Searched `pkg/tools/i2c.go` (I2C tool for agent-triggered reads, not continuous polling), `pkg/sensors/` (no I2C source). Reused: known sensor address database from common I2C sensor documentation.

## pkg/tools/fetch.go
- **Purpose:** HTTP content fetching tool — agent can download web pages and extract readable text (HTML stripped) or inspect response headers. Actions: get (fetch body), headers (HEAD only).
- **Duplicate search:** Searched `pkg/tools/web.go` (DuckDuckGo search, not general URL fetch), `pkg/tools/browser.go` (headless browser automation, not simple fetch), `pkg/tools/` (no fetch tool). Reused: HTML stripping regexes from `pkg/tools/web.go` L21-L26.

## pkg/tools/goals.go
- **Purpose:** Goal/project tracking tool — reads/writes GOALS.md for lightweight goal management. Actions: list, add (with priority), update (status changes), review (summary).
- **Duplicate search:** Searched `pkg/tools/` (has filesystem, notes tools — no goal-specific tool), `pkg/agent/` (no goal tracking), `pkg/memory/` (temporal/hindsight — different purpose). No existing goal tool found.

## pkg/tools/camera_tool.go
- **Purpose:** Agent-facing camera tool — gives the agent on-demand access to capture images from any discovered camera (USB webcam, CSI, phone front/rear/screen). Actions: capture (single camera), look (all cameras), list (show available). Uses CameraSource.CaptureFrom for hardware access and Ollama vision for scene analysis.
- **Duplicate search:** Searched `pkg/tools/` (has `vision.go` for analyzing existing images, `phone.go` for phone screenshot — neither provides on-demand capture-and-describe), `pkg/sensors/camera_source.go` (background polling only, no tool interface), `pkg/agent/` (no camera tool registration). Reused: `pkg/sensors/camera_source.go` CaptureFrom/ListCameras methods for hardware access.

## pkg/tools/goals.go
- **Purpose:** Agent tool `goals` — list/read `GOALS.md`, append active goals (priority + added date), update status by goal title, review section counts. EXA markdown format (Active / Completed / Paused / Abandoned).
- **Duplicate search:** Searched `pkg/tools/` (`filesystem.go` generic read/write only; `cron.go` scheduling, not goal docs; `feedback.go` ratings), `pkg/agent/` (no GOALS.md tool). No existing goal-tracker tool.

## pkg/tools/fetch.go
- **Purpose:** Agent tool `fetch` — action `get` (HTTP GET, HTML tag strip via regexp, JSON/text handling, `max_length` truncation) and `headers` (HEAD, sorted header dump). Uses `*http.Client` on `FetchTool`.
- **Duplicate search:** `pkg/tools/web.go` has `WebFetchTool` (`web_fetch`, GET only, `maxChars` default 50000, JSON/HTML/raw in `Execute`) — overlapping fetch logic but different tool name, parameters (`action`/`max_length`), and `headers` action; no `fetch`-named tool. `pkg/tools/browser.go` — browser automation, not raw HTTP. No duplicate `fetch` tool.

## pkg/providers/picolm_provider.go
- **Purpose:** PicoLM provider — local-first LLM inference via picolm C binary subprocess. Supports --json grammar mode for structured tool calling, --cache for KV persistence (skips prompt re-processing), and ARM NEON SIMD. 45MB RAM, 80KB binary, zero network, zero Python.
- **Duplicate search:** Searched `pkg/providers/` (found `bitnet_provider.go` for local inference — different runtime, no KV cache, no JSON grammar; `http_provider.go` for API-based providers; no PicoLM). Searched `reference/picolm/` (upstream C binary exists but no Go integration). Searched `reference/picoclaw/` (has PicoLM config support but in separate project, not integrated into xagent). No existing PicoLM provider found.
