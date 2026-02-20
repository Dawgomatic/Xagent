# 📅 iCalendar Sync for OpenClaw

**Professional iCloud Calendar integration with enterprise-grade security**

[![Version](https://img.shields.io/badge/version-2.3.0-blue.svg)](https://github.com/h8kxrfp68z-lgtm/iCalendar-Sync/releases)
[![Security Rating](https://img.shields.io/badge/security-A-brightgreen.svg)](SECURITY.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Python](https://img.shields.io/badge/python-3.9+-blue.svg)](https://www.python.org/downloads/)
[![Cyrillic Support](https://img.shields.io/badge/cyrillic-✓-green.svg)](CHANGELOG.md)

---

## 🎉 New in v2.3.0 (Feb 13, 2026)

- **✨ UPDATE Command**: Modify existing events without breaking UID integrity
- **🔁 Smart Recurrence Handling**: Update single instance, all instances, or future instances
- **📅 Exception Support**: RECURRENCE-ID for creating exceptions in recurring series
- **🎯 Granular Control**: `--mode single/all/future` for precise updates
- **🔧 Series Management**: Split recurring series or update master rules

## Key Features

- **🌍 Multi-language Support**: Cyrillic, Chinese, Arabic calendar names
- **🔐 Secure Credentials**: OS keyring storage (Keychain/Credential Manager/Secret Service)
- **🐳 Docker/CI Ready**: Environment variables and secrets manager support
- **📅 Full CalDAV Support**: List, get, create, delete events with recurrence

[See full CHANGELOG](CHANGELOG.md)

---

## ✨ What This Version Includes

**Version 2.3.0** is a production-ready iCloud Calendar sync skill with:

### ✅ Fully Implemented Features:

**Fully functional modules:**
- `src/icalendar_sync/calendar.py` (33 KB) - Complete CalDAV client
  - Event CRUD operations (create, read, update, delete)
  - Secure credential management via OS keyring
  - Input validation and security checks
  - Rate limiting (10 calls/60s)
  - Recurring events (RRULE support)
  - Multi-calendar support
  - Cyrillic/Unicode calendar names
  - CLI interface with headless mode

- `src/icalendar_sync/__main__.py` - Module entry point
- `src/icalendar_sync/i18n.py` (40 KB) - Internationalization for 20 languages
- `src/icalendar_sync/translations_extended.py` - Extended translations
- `src/icalendar_sync/translations_extended2.py` - Additional translations

### ❌ NOT IMPLEMENTED (Mentioned in Extended Docs Only):

**These modules do NOT exist as separate files in v2.2.6:**
- ❌ `calendar_vault.py` - Described in ARCHITECTURE.md but not implemented
- ❌ `privacy_engine.py` - Mentioned in DOCUMENTATION.md but not a separate module
- ❌ `rate_limiter.py` - Rate limiting is embedded in calendar.py, not standalone
- ❌ `connector/` directory - No separate connector modules
- ❌ Advanced multi-agent isolation system

**Why the documentation mismatch?**
- DOCUMENTATION.md and ARCHITECTURE.md were written for a future v3.0 architecture
- v2.2.6 consolidates all logic into `calendar.py` for simplicity
- Some "modules" exist as functions/classes within calendar.py, not separate files
- Extended docs are kept for reference/planning but describe future state

**What to trust:**
- ✅ **THIS README** - Accurate for v2.2.6
- ✅ **SKILL.md** - Accurate for v2.2.6  
- ✅ **skill.yaml** - Correct metadata
- ✅ **CLAWHUB_METADATA.yaml** - Explicit credential declarations (NEW v2.2.6)
- ✅ **SECURITY_SCAN_NOTICE.md** - Addresses all ClawHub concerns (NEW v2.2.6)
- ✅ **CHANGELOG.md** - Complete version history
- ⚠️ **DOCUMENTATION.md** - Mix of current + future features (read with caution)
- ⚠️ **ARCHITECTURE.md** - Describes future v3.0 architecture

### 🔒 CREDENTIALS REQUIRED:

**Required environment variables:**
- `ICLOUD_USERNAME` - Your Apple ID (e.g., user@icloud.com)
- `ICLOUD_APP_PASSWORD` - App-Specific Password from [https://appleid.apple.com](https://appleid.apple.com)

**Storage:**
- ✅ Credentials are securely stored in your operating system's keyring
- ✅ macOS: Keychain, Windows: Credential Manager, Linux: Secret Service API
- ✅ Never stored in plaintext or logged

---

## ✨ Features

### 💪 Core Capabilities

- ✅ **Full Calendar Sync** - Bidirectional sync with iCloud
- 🌐 **CalDAV Protocol** - Standard-compliant implementation
- 🗓️ **Event Management** - Create, read, update, delete events
- ✨ **Smart Updates** - Modify events without breaking UID integrity
- 🔁 **Recurring Events** - Full RRULE support with smart instance handling
- 🎯 **Granular Control** - Update single, all, or future instances of series
- ⏰ **Alarms & Reminders** - Multiple alarms per event
- 📱 **Multi-Device** - Instant sync across iPhone, iPad, Mac
- 📂 **Multiple Calendars** - Work, Personal, Custom calendars
- 🌍 **Unicode Support** - Cyrillic, CJK, and all international scripts
- ⚡ **Conflict Detection** - Automatic scheduling conflict warnings

### 🔒 Security Features (v2.2.6)

- 🔑 **Keyring Integration** - Secure credential storage in OS keychain
- 🛡️ **Input Validation** - Protection against injection attacks (Unicode-aware)
- 🚫 **Rate Limiting** - DoS protection (10 calls/60s)
- 🔐 **SSL Verification** - Enforced certificate validation
- 🧹 **Log Filtering** - Automatic credential redaction
- 🧵 **Thread Safety** - Safe concurrent access
- 📝 **Atomic Operations** - Safe file writes
- ⏱️ **Timeout Protection** - 30s timeout on interactive inputs
- ⚠️ **CLI Credential Warnings** - Documented risks of command-line password exposure
- 🐳 **Docker Secrets Support** - Secure deployment patterns

---

## 🚀 Quick Start

### Installation

```bash
# From source
git clone https://github.com/h8kxrfp68z-lgtm/OpenClaw.git
cd OpenClaw/icalendar-sync
pip install -e .

# Or via pip (when published)
pip install openclaw-icalendar-sync
Setup
Interactive Setup (Default)
bash
# Interactive setup wizard
icalendar-sync setup
You'll need:

Apple ID email (e.g., user@icloud.com)

App-Specific Password from https://appleid.apple.com

Go to: Sign-In & Security → App-Specific Passwords

Create new password for "OpenClaw iCalendar Sync"

Headless Setup (Automated)
bash
# For automation, Docker, CI/CD, OpenClaw agents
# Use environment variables for credentials
export ICLOUD_USERNAME="user@icloud.com"
export ICLOUD_APP_PASSWORD="xxxx-xxxx-xxxx-xxxx"
icalendar-sync setup --non-interactive

Credentials are securely stored in your OS keyring:
- **macOS**: Keychain
- **Windows**: Credential Manager
- **Linux**: Secret Service (GNOME Keyring/KWallet)

📖 Usage
List Calendars
bash
icalendar-sync list
Output:

text
📅 Available Calendars (5):

  -  Personal
  -  Work
  -  Family
  -  Личный      # Cyrillic support!
  -  Работа      # Russian calendar names
Get Events
bash
# Next 7 days (default)
icalendar-sync get --calendar "Work"

# Cyrillic calendar names
icalendar-sync get --calendar "Личный" --days 7
icalendar-sync get --calendar "Работа" --days 30

# English calendars still work
icalendar-sync get --calendar "Personal" --days 30
Create Event
Simple Event
bash
icalendar-sync create --calendar "Work" --json '{
  "summary": "Team Meeting",
  "dtstart": "2026-02-12T14:00:00+03:00",
  "dtend": "2026-02-12T15:00:00+03:00",
  "description": "Q1 Planning Discussion",
  "location": "Conference Room A"
}'
Cyrillic Event
bash
icalendar-sync create --calendar "Личный" --json '{
  "summary": "Встреча с врачом",
  "dtstart": "2026-02-15T10:00:00+03:00",
  "dtend": "2026-02-15T11:00:00+03:00",
  "description": "Ежегодный осмотр",
  "location": "Поликлиника №5"
}'
From JSON File
bash
# Create event.json
cat > event.json << EOF
{
  "summary": "Doctor Appointment",
  "dtstart": "2026-02-15T10:00:00+03:00",
  "dtend": "2026-02-15T11:00:00+03:00",
  "description": "Annual checkup",
  "alarms": [
    {"minutes": 60, "description": "1 hour before"},
    {"minutes": 15, "description": "15 minutes before"}
  ]
}
EOF

icalendar-sync create --calendar "Personal" --json event.json
Recurring Event
bash
icalendar-sync create --calendar "Work" --json '{
  "summary": "Weekly Standup",
  "dtstart": "2026-02-12T09:00:00+03:00",
  "dtend": "2026-02-12T09:30:00+03:00",
  "rrule": {
    "freq": "WEEKLY",
    "interval": 1,
    "byday": ["MO", "WE", "FR"],
    "count": 50
  }
}'
Update Event
Simple Update
bash
# Update non-recurring event
icalendar-sync update --calendar "Work" --uid "event-uid-here" --json '{
  "summary": "Updated Meeting Title",
  "location": "New Conference Room"
}'
Update Single Instance of Recurring Event
bash
# Change one Friday standup to Saturday without affecting other Fridays
icalendar-sync update --calendar "Work" --uid "weekly-standup-uid" \
  --recurrence-id "2026-02-14T09:00:00+03:00" \
  --mode single \
  --json '{
    "dtstart": "2026-02-15T09:00:00+03:00",
    "dtend": "2026-02-15T09:30:00+03:00",
    "summary": "Standup - Moved to Saturday"
  }'
Update All Instances
bash
# Change time for ALL future standups
icalendar-sync update --calendar "Work" --uid "weekly-standup-uid" \
  --mode all \
  --json '{
    "dtstart": "2026-02-12T10:00:00+03:00",
    "dtend": "2026-02-12T10:30:00+03:00"
  }'
Update This and Future Instances
bash
# Split series: keep old time before Feb 20, new time after
icalendar-sync update --calendar "Work" --uid "weekly-standup-uid" \
  --recurrence-id "2026-02-20T09:00:00+03:00" \
  --mode future \
  --json '{
    "dtstart": "2026-02-20T14:00:00+03:00",
    "dtend": "2026-02-20T14:30:00+03:00"
  }'
Delete Event
bash
# First, get the event UID
icalendar-sync get --calendar "Work"

# Then delete
icalendar-sync delete --calendar "Личный" --uid "event-uid-here"
Module Execution
bash
# All commands now work as Python module
python -m icalendar_sync list
python -m icalendar_sync setup
python -m icalendar_sync get --calendar "Личный" --days 7
python -m icalendar_sync create --calendar "Work" --json event.json
📚 API Usage (Python)
python
from icalendar_sync import CalendarManager
from datetime import datetime, timezone

# Initialize
manager = CalendarManager()

# List calendars (including Cyrillic names)
calendars = manager.list_calendars()
print(f"Found {len(calendars)} calendars")

# Get events from Russian calendar
events = manager.get_events("Личный", days_ahead=7)

# Create event in Cyrillic calendar
event_data = {
    "summary": "Проектный дедлайн",
    "dtstart": datetime(2026, 2, 20, 17, 0, tzinfo=timezone.utc),
    "dtend": datetime(2026, 2, 20, 18, 0, tzinfo=timezone.utc),
    "description": "Финальная сдача проекта",
    "location": "Онлайн",
    "alarms": [
        {"minutes": 1440, "description": "За 1 день"},
        {"minutes": 60, "description": "За 1 час"}
    ]
}

success = manager.create_event(
    calendar_name="Работа",
    event_data=event_data,
    check_conflicts=True,
    auto_confirm=False
)

if success:
    print("✅ Событие создано успешно")
🛠️ Configuration
Environment Variables
bash
# Required (or use keyring)
export ICLOUD_USERNAME="user@icloud.com"
export ICLOUD_APP_PASSWORD="xxxx-xxxx-xxxx-xxxx"

# Optional
export DEFAULT_CALENDAR="Personal"
export LOG_LEVEL="INFO"  # DEBUG, INFO, WARNING, ERROR
Security Limits
python
# Configurable in calendar.py
MAX_CALENDAR_NAME_LENGTH = 255
MAX_SUMMARY_LENGTH = 500
MAX_DESCRIPTION_LENGTH = 5000
MAX_LOCATION_LENGTH = 500
MAX_JSON_FILE_SIZE = 1048576  # 1MB
MAX_DAYS_AHEAD = 365
RATE_LIMIT_CALLS = 10
RATE_LIMIT_WINDOW = 60  # seconds
INPUT_TIMEOUT = 30  # seconds
📊 Event Schema
Required Fields
summary (string): Event title (Unicode supported)

dtstart (ISO 8601 datetime): Start time

dtend (ISO 8601 datetime): End time

Optional Fields
description (string): Event details (Unicode supported)

location (string): Event location (Unicode supported)

status (string): CONFIRMED, TENTATIVE, CANCELLED

priority (int): 0-9 (0=undefined, 1=highest, 9=lowest)

attendees (array): List of attendee emails

alarms (array): List of alarm objects

rrule (object): Recurrence rule

Datetime Format
Use ISO 8601 with timezone:

text
2026-02-12T14:00:00+03:00  # Moscow time
2026-02-12T11:00:00Z       # UTC
2026-02-12T06:00:00-05:00  # EST
Recurrence Rule (RRULE)
json
{
  "freq": "WEEKLY",
  "interval": 1,
  "count": 10,
  "until": "2026-12-31",
  "