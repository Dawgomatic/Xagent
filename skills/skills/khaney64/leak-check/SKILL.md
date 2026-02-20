---
name: leak-check
description: Scan session logs for leaked credentials. Checks JSONL session files against known credential patterns and reports which AI provider received the data.
metadata: {"openclaw":{"emoji":"🔐","requires":{"bins":["node"]}}}
---

# Leak Check

Scan OpenClaw session JSONL files for leaked credentials. Reports which real AI provider (anthropic, openai, google, etc.) received the data, skipping internal delivery echoes.

## Quick Start

```bash
# Check for leaked credentials (default: discord format)
node scripts/leak-check.js

# JSON output
node scripts/leak-check.js --format json
```

## Configuration

Credentials to check are defined in `leak-check.json`:

```json
[
  { "name": "Discord", "search": "abc*xyz" },
  { "name": "Postmark", "search": "k7Qm9x" }
]
```

**Important:** Do not store full credentials in this file. Use only a partial fragment — enough to uniquely identify the credential via a contains, begins-with, or ends-with match.

**Wildcard patterns:**
- `abc*` — starts with "abc"
- `*xyz` — ends with "xyz"
- `abc*xyz` — starts with "abc" AND ends with "xyz"
- `abc` (no asterisk) — contains "abc"
- `""` (empty) — skip this credential

## Options

- `--format <type>` — Output format: `discord` (default) or `json`
- `--config <path>` — Path to credential config file (default: `leak-check.json` in skill root)
- `--help`, `-h` — Show help message

## Output

### Discord (Default)

```
🔐 **Credential Leak Check**

⚠️ **2 leaked credentials found**

**Discord Token**
• Session: `abc12345` | 2026-02-14 18:30 UTC | Provider: anthropic

**Postmark**
• Session: `def67890` | 2026-02-10 09:15 UTC | Provider: anthropic
```

Or if clean:

```
🔐 **Credential Leak Check**
✅ No leaked credentials found (checked 370 files, 7 credentials)
```

### JSON

```json
{
  "leaks": [
    {
      "credential": "Discord Token",
      "session": "abc12345",
      "timestamp": "2026-02-14T18:30:00.000Z",
      "provider": "anthropic"
    }
  ],
  "summary": {
    "filesScanned": 370,
    "credentialsChecked": 7,
    "leaksFound": 2
  }
}
```
