# Clawdbot Workspace

Personal workspace for Clawdbot — a space lobster AI assistant 

## Structure

```
~/clawd/
├── AGENTS.md          # Agent behavior config (symlink → NAS)
├── SOUL.md            # Personality & tone (symlink → NAS)
├── USER.md            # User profile (symlink → NAS)
├── TOOLS.md           # Local tool notes (symlink → NAS)
├── IDENTITY.md        # Agent identity (symlink → NAS)
├── HEARTBEAT.md       # Periodic check tasks (symlink → NAS)
├── PERMISSIONS.md     # Access controls (symlink → NAS)
│
├── skills/            # Installed skills (via ClawdHub)
├── scripts/           # Local utility scripts
├── docs/              # Architecture & setup docs
├── .learnings/        # Self-improvement logs
└── .clawdhub/         # ClawdHub lockfile
```

## Skills

Managed via [ClawdHub](https://clawdhub.com). See `SKILLS.md` for details.

```bash
clawdhub list              # Show installed
clawdhub search "query"    # Find skills
clawdhub install <slug>    # Install
clawdhub update --all      # Update all
```

## Config Files

Core config files are symlinked to NAS storage (`~/mnt/services/clawdbot/brain/Config/`) for persistence and mobile access via Obsidian LiveSync.

## Memory

Daily memory files live on NAS at `~/mnt/services/clawdbot/brain/memory/` (symlinked as `memory/`).

---

*Powered by [Clawdbot](https://github.com/clawdbot/clawdbot)*
