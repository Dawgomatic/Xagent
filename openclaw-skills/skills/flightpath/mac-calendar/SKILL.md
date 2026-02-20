---
name: mac-calendar
description: Read and manage Apple Calendar on Guy's Macs (MacBook or Mac Mini) over SSH. Use when the user wants to check calendar events, create events, or manage Apple Calendar in any way.
---

# Mac Calendar

Access Apple Calendar on Guy's Macs via SSH + AppleScript.

## Script

`scripts/calendar.sh <action> [args...]`

### Actions

- `list-calendars` — List all available calendars
- `events [days] [calendar]` — List upcoming events (default: 7 days, all calendars)
- `today` — List today's events
- `tomorrow` — List tomorrow's events  
- `this-week` — List this week's events
- `add-event <calendar> <title> <start-date> [end-date] [location] [notes]` — Create new event
- `search <query> [days]` — Search events by title/notes (default: ±30 days)

### Date Formats

- `YYYY-MM-DD` for all-day events
- `YYYY-MM-DD HH:MM` for timed events (24-hour format, Central Time)

### Examples

```bash
# List all calendars
bash scripts/calendar.sh list-calendars

# Check today's events
bash scripts/calendar.sh today

# See next 14 days
bash scripts/calendar.sh events 14

# Create a new event
bash scripts/calendar.sh add-event "Work" "Team Meeting" "2026-02-18 14:00" "2026-02-18 15:00" "Conference Room A"

# Search for doctor appointments
bash scripts/calendar.sh search "doctor"
```

## Target Mac

Uses the same target selection as other Mac skills:
- Default: Mac Mini (`guym@doclib`)
- Override with `MAC_TARGET=macbook` for MacBook