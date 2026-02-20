# OpenClaw Full CLI Reference

Reference normalized against `https://docs.openclaw.ai/cli`.
Last verified: 2026-02-17.

Safety note:
- Commands listed here reflect upstream CLI surface.
- Local wrapper policy may block high-risk groups unless `OPENCLAW_WRAPPER_ALLOW_RISKY=1`.
- See `references/security-policy.md`.

## Core Commands
- `openclaw onboard`: Run onboarding wizard.
- `openclaw doctor`: Validate local install and health.
- `openclaw status`: Show global OpenClaw status.
- `openclaw version`: Print CLI version.
- `openclaw dashboard [--no-open]`: Open or print dashboard URL.
- `openclaw config [--list|--get <key>|--set <key=value>]`: Inspect or modify config values.
- `openclaw update`: Update CLI to latest stable build.
- `openclaw uninstall`: Remove CLI and optionally data.

## Gateway Commands
- `openclaw gateway`: Start gateway in foreground.
- `openclaw gateway status`: Show gateway runtime status.
- `openclaw gateway health`: Check gateway health endpoint.
- `openclaw gateway restart`: Restart managed gateway process.
- `openclaw gateway stop`: Stop managed gateway process.
- `openclaw gateway install`: Install gateway background service.
- `openclaw gateway uninstall`: Remove gateway background service.
- `openclaw gateway call <path> [--method ...] [--data ...]`: Call gateway APIs.

## Channels and Pairing
- `openclaw channels list`: List configured channels.
- `openclaw channels login --channel <name>`: Authenticate a channel.
- `openclaw channels logout --channel <name>`: Disconnect a channel.
- `openclaw pairing`: Show pairing flow for mobile nodes (high-risk).

## Models
- `openclaw models list`: Show available models.
- `openclaw models set <model>`: Set default model.
- `openclaw models auth add --provider <openai|anthropic>`: Add model provider auth.
- `openclaw models auth list`: Show provider auth entries.
- `openclaw models auth remove --provider <name>`: Remove provider auth.
- `openclaw models aliases list`: List alias map.
- `openclaw models aliases add <alias> <model>`: Add alias.
- `openclaw models aliases remove <alias>`: Remove alias.
- `openclaw models scan`: Discover local models.

## Automation
- `openclaw cron list`: List cron jobs.
- `openclaw cron add ...`: Create cron job (high-risk).
- `openclaw cron run <jobId>`: Run job immediately (high-risk).
- `openclaw cron remove <jobId>`: Delete job (high-risk).

## Browser and Plugins
- `openclaw browser status|start|stop`: Manage browser runtime (high-risk).
- `openclaw browser open <url>`: Navigate browser (high-risk).
- `openclaw browser screenshot [--full-page]`: Capture screenshot.
- `openclaw browser snapshot --format aria`: Capture structured page snapshot.
- `openclaw plugins list`: List plugins.
- `openclaw plugins install <path-or-url>`: Install plugin (high-risk).
- `openclaw plugins enable <name>`: Enable plugin (high-risk).
- `openclaw plugins disable <name>`: Disable plugin.
