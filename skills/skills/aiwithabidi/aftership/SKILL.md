---
name: aftership
description: "AfterShip — package tracking, delivery notifications, estimated delivery dates, and courier detection."
homepage: https://www.agxntsix.ai
license: MIT
compatibility: Python 3.10+ (stdlib only — no dependencies)
metadata: {"openclaw": {"emoji": "", "requires": {"env": ["AFTERSHIP_API_KEY"]}, "primaryEnv": "AFTERSHIP_API_KEY", "homepage": "https://www.agxntsix.ai"}}
---

#  AfterShip

AfterShip — package tracking, delivery notifications, estimated delivery dates, and courier detection.

## Requirements

| Variable | Required | Description |
|----------|----------|-------------|
| `AFTERSHIP_API_KEY` |  | AfterShip API key |


## Quick Start

```bash
# List all trackings
python3 {{baseDir}}/scripts/aftership.py list-trackings --page "1" --limit "50" --keyword <value>

# Get tracking details
python3 {{baseDir}}/scripts/aftership.py get-tracking --slug <value> --tracking-number <value>

# Create tracking
python3 {{baseDir}}/scripts/aftership.py create-tracking --tracking-number <value> --slug <value> --title <value> --emails <value> --phones <value>

# Delete tracking
python3 {{baseDir}}/scripts/aftership.py delete-tracking --slug <value> --tracking-number <value>

# Retrack expired tracking
python3 {{baseDir}}/scripts/aftership.py retrack --slug <value> --tracking-number <value>

# Detect courier for tracking number
python3 {{baseDir}}/scripts/aftership.py detect-courier --tracking-number <value>

# List all supported couriers
python3 {{baseDir}}/scripts/aftership.py list-couriers

# Get last checkpoint
python3 {{baseDir}}/scripts/aftership.py get-last-checkpoint --slug <value> --tracking-number <value>

# Get notification settings
python3 {{baseDir}}/scripts/aftership.py list-notifications --slug <value> --tracking-number <value>
```

## Output Format

All commands output JSON by default.

## Script Reference

| Script | Description |
|--------|-------------|
| `{baseDir}/scripts/aftership.py` | Main CLI — all commands in one tool |

## Credits
Built by [M. Abidi](https://www.linkedin.com/in/mohammad-ali-abidi) | [agxntsix.ai](https://www.agxntsix.ai)
[YouTube](https://youtube.com/@aiwithabidi) | [GitHub](https://github.com/aiwithabidi)
Part of the **AgxntSix Skill Suite** for OpenClaw agents.

 **Need help setting up OpenClaw for your business?** [Book a free consultation](https://cal.com/agxntsix/abidi-openclaw)
