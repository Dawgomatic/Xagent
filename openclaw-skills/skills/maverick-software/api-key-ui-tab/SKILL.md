---
name: apikeys-ui
description: API Keys management UI tab for OpenClaw dashboard. Enter and save API keys directly in the browser without exposing them to the AI agent. Shows which keys are configured, which are missing, and provides secure input fields for each.
version: 1.1.0
author: OpenClaw Community
metadata: {"clawdbot":{"emoji":"🔑","requires":{"clawdbot":">=2026.1.0"},"category":"settings"}}
---

# API Keys UI

Adds an **API Keys** tab to the OpenClaw Control dashboard under **Settings**. Manage your API keys directly in the browser — your keys are saved to the config without ever being sent to the AI agent.

## Features

| Feature | Description |
|---------|-------------|
| **Dynamic Discovery** | Scans entire config for API keys — no hardcoded list |
| **Dashboard Tab** | "API Keys" under Settings in sidebar |
| **Key Status** | See which keys are configured (✓) or missing |
| **Secure Input** | Password fields — keys never displayed after saving |
| **Direct Save** | Keys go straight to config via `config.patch` RPC |
| **Provider Links** | "Get key ↗" buttons for known providers |
| **Clear Keys** | Remove keys from config with one click |
| **Auto-Grouping** | Keys grouped by Environment / Skills / Other |

## Dynamic Key Discovery

The UI **automatically scans your entire config** for API keys. No hardcoded list — if it looks like an API key, it shows up.

### Detection Patterns
Fields matching these patterns are discovered:
- `apiKey`, `api_key`
- `token`, `secret`
- `*_KEY`, `*_TOKEN`, `*_SECRET`

### Where It Looks
- `env.*` — Environment variables
- `skills.entries.*.apiKey` — Skill-specific keys
- `messages.tts.*.apiKey` — TTS provider keys
- Any nested config path

### Known Providers (Enhanced UX)
These get friendly names, descriptions, and "Get key" links:

| Provider | Env Key |
|----------|---------|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Brave Search | `BRAVE_API_KEY` |
| ElevenLabs | `ELEVENLABS_API_KEY` |
| Google | `GOOGLE_API_KEY` |
| Deepgram | `DEEPGRAM_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Groq | `GROQ_API_KEY` |
| Fireworks | `FIREWORKS_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| xAI | `XAI_API_KEY` |
| Perplexity | `PERPLEXITY_API_KEY` |
| GitHub | `GITHUB_TOKEN` |

Unknown keys still appear — they just get auto-generated names from their config path.

## Security Model

```
┌─────────────────────────────────────────────────────────┐
│  Browser                                                │
│  ┌─────────────────────────────────────────────────┐   │
│  │  API Keys Tab                                    │   │
│  │  ┌─────────────────────────────────────────┐    │   │
│  │  │ OpenAI: [••••••••••] [Save] [Clear]     │    │   │
│  │  │ Anthropic: [        ] [Save]            │    │   │
│  │  └─────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────┘   │
│                           │                             │
│                           ▼ (direct RPC, not via agent) │
└───────────────────────────┼─────────────────────────────┘
                            │
                    ┌───────▼───────┐
                    │   Gateway     │
                    │ config.patch  │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │ clawdbot.json │
                    └───────────────┘
```

**Key point:** The AI agent **never sees your API keys**. The browser talks directly to the gateway, which writes to the config file.

## Agent Installation Prompt

```
Install the apikeys-ui skill.

The skill is at: ~/clawd/skills/apikeys-ui/
Follow INSTALL_INSTRUCTIONS.md step by step.

Summary of changes needed:
1. Copy apikeys-views.ts to ui/src/ui/views/apikeys.ts
2. Copy apikeys-controller.ts to ui/src/ui/controllers/apikeys.ts
3. Add "apikeys" tab to navigation.ts (TAB_GROUPS, Tab type, TAB_PATHS, icon, title, subtitle)
4. Add state variables to app.ts
5. Add view rendering to app-render.ts
6. Add tab loading to app-settings.ts
7. Build and restart: pnpm ui:build && clawdbot gateway restart
```

## Files Included

```
apikeys-ui/
├── SKILL.md                      # This file
├── INSTALL_INSTRUCTIONS.md       # Step-by-step installation
└── reference/
    ├── apikeys-views.ts          # UI view (Lit templates)
    └── apikeys-controller.ts     # State management & RPC calls
```

## How It Works

1. **Controller** (`apikeys-controller.ts`):
   - `scanForKeys()` — recursively walks config, finds fields matching key patterns
   - `loadApiKeys()` — fetches config, scans it, adds common missing env keys
   - `saveApiKey()` — builds nested patch object, writes via `config.patch` RPC
   - `clearApiKey()` — patches key to `null`
   - `KNOWN_PROVIDERS` — metadata (names, descriptions, docs URLs) for recognized keys

2. **View** (`apikeys-views.ts`):
   - Groups keys by category (env / skills / other)
   - Renders password inputs with save/clear buttons
   - Shows masked preview for configured keys
   - "Get key ↗" links for known providers

3. **Security**:
   - Keys entered in browser go directly to gateway via WebSocket
   - AI agent never sees the key values
   - Only masked previews shown in UI (first 4 + last 4 chars)

## Changelog

### v1.1.0
- **Dynamic key discovery** — scans entire config instead of hardcoded list
- Auto-grouping by category (Environment / Skills / Other)
- Refresh button for full page reload
- Common env keys shown even if not configured

### v1.0.0
- Initial release
- Hardcoded provider slots
- Save/Clear functionality
