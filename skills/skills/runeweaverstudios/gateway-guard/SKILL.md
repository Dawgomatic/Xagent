---
name: gateway-guard
description: Ensures OpenClaw gateway auth consistency and can auto-prompt "continue" when a run error (Unhandled stop reason: error) appears in gateway logs. Use when checking or fixing gateway token/password mismatch, device_token_mismatch errors, or before delegating to sub-agents.
---

# Gateway Guard

## Description

Ensures OpenClaw gateway auth consistency and can auto-prompt "continue" when a run error (Unhandled stop reason: error) appears in gateway logs. Use when checking or fixing gateway token/password mismatch, device_token_mismatch errors, or before delegating to sub-agents.

Ensures OpenClaw gateway auth consistency and can auto-prompt "continue" when a run error (Unhandled stop reason: error) appears in gateway logs. Use when checking or fixing gateway token/password mismatch, device_token_mismatch errors, or before delegating to sub-agents.

# Gateway Guard

Keeps OpenClaw gateway authentication in sync with `openclaw.json`. Use when the user or agent sees gateway auth issues, `device_token_mismatch`, or needs to ensure the gateway is running with the correct token/password before spawning sub-agents.


## Usage

- User or logs report "Gateway auth issue", "device_token_mismatch", or "unauthorized"
- Before running the router and `sessions_spawn` (orchestrator flow): check gateway status first
- After installing or updating OpenClaw: verify gateway and config match
- When the TUI disconnects or won't connect: fix auth and restart gateway
- **Run error (Unhandled stop reason: error):** run `continue-on-error --loop` (e.g. via LaunchAgent or cron) so the guard auto-sends "continue" to the agent when this appears in `gateway.log`

```bash
python3 <skill-dir>/scripts/gateway_guard.py status [--json]
python3 <skill-dir>/scripts/gateway_guard.py ensure [--apply] [--json]
python3 <skill-dir>/scripts/gateway_guard.py continue-on-error [--once] [--loop] [--interval 30] [--json]
```

- **status** — Report whether the running gateway's auth matches `openclaw.json`. Exit 0 if ok, 1 if mismatch.
- **ensure** — Same check; if mismatch and `--apply`, restart the gateway with credentials from config. Writes `gateway.auth` to `openclaw.json` **only when it is missing or wrong** (never overwrites correct config).
- **continue-on-error** — When `gateway.log` contains **Unhandled stop reason: error** (run error), send **continue** to the agent via `openclaw agent --message continue --deliver`. Use `--once` to check once and exit, or `--loop` to run every `--interval` seconds. Cooldown 90s between triggers. State: `logs/gateway-guard.continue-state.json`.
- **watch** — Single combined daemon (one LaunchAgent). Each run: (0) **token sync** — `ensure --apply` so gateway auth matches config (prevents device_token_mismatch); (1) gateway back → what-just-happened summary; (2) continue-on-error check. **Install one daemon:** `bash <skill-dir>/scripts/install_watcher.sh` (or `install_continue_on_error.sh`). This unloads the old separate what-just-happened and continue-on-error LaunchAgents and loads `com.openclaw.gateway-guard.watcher` so users only need one. For periodic gateway recovery (check every 10s, restart if not ok), use the separate **gateway-watchdog** skill.


## Behavior

- Reads `openclaw.json` → `gateway.auth` (token or password) and `gateway.port`.
- Compares with the process listening on that port (and optional guard state file).
- If `ensure --apply`: restarts gateway via `openclaw gateway stop` then `openclaw gateway --port N --auth token|password --token|--password SECRET`.
- If token is missing in config (token mode only): generates a token, writes it to config once, then proceeds. Does not overwrite config when it is already correct.
- **continue-on-error:** Tails `OPENCLAW_HOME/logs/gateway.log` for the string `Unhandled stop reason: error`. When found (and not in cooldown), runs `openclaw agent --message continue --deliver` so the agent receives "continue" and can resume. Run `install_continue_on_error.sh` to install a LaunchAgent that checks every 30s. If the error appears in the TUI but the watcher never triggers, the gateway may not be writing run errors to `gateway.log` — ensure run/stream errors are logged there.


## JSON output (for orchestration)

- **status --json** / **ensure --json**: `ok`, `secretMatchesConfig`, `running`, `pid`, `reason`, `recommendedAction`, `configPath`, `authMode`, `gatewayPort`. When not ok, `recommendedAction` is "run gateway_guard.py ensure --apply and restart client session".


## Requirements

- OpenClaw `openclaw.json` with `gateway.auth` (mode `token` or `password`) and `gateway.port`.
- **CLI / system:** `openclaw` CLI on PATH (for `ensure --apply` and continue-on-error); `lsof` and `ps` (macOS/Unix); `launchctl` on macOS when using the LaunchAgent install scripts.
- **Environment (optional):** `OPENCLAW_HOME` — OpenClaw home directory (default: `~/.openclaw`). `OPENCLAW_BIN` — Path or name of `openclaw` binary (default: `openclaw`).
