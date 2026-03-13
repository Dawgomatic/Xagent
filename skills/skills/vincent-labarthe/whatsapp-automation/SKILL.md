---
name: whatsapp-automation
description: "WhatsApp automation with intelligent Telegram alerts. Detects appointments, important messages, and suggests calendar additions. OpenClaw cron jobs + Claude analysis."
---

 **DEPRECATED - Use the new version instead!**

This skill has been replaced by a newer, improved version:

 **New Skill (Recommended):** https://www.clawhub.ai/Vincent-Labarthe/whatsapp-telegram-calendar-alert

**Improvements in the new version:**
-  Better appointment detection
-  Calendar integration with confirmation
-  Cleaner setup process
-  Better documentation
-  Zero duplicates guarantee

**Use the new skill instead.** This version is archived for reference only.

---

#  WhatsApp Automation Skill

Automatically capture WhatsApp messages and get intelligent Telegram notifications with Claude analysis.

---

## Prerequisites

Before setup, make sure you have:
-  **OpenClaw installed** (with cron + message tools)
-  **An AI agent configured** (Claude, Gemini, Anthropic, etc.)
-  **Telegram bot configured** in your OpenClaw config (for alerts)
-  **Google Calendar API** (optional, for calendar additions)

##  ONE-LINE SETUP

```bash
bash ~/.openclaw/workspace/whatsapp-automation-skill/setup.sh
```

This will:
1. Start WAHA (Docker) for WhatsApp capture
2. Create 2 OpenClaw cron jobs (Claude-powered analysis)
3. Set up message storage

Done! 

---

## What It Does

| Feature | How |
|---------|-----|
|  **Appointment Detection** | Your AI agent finds "meeting/rdv/reunion" → asks if you want to add to Google Calendar |
|  **Important Messages** | AI detects tone/keywords suggesting importance → sends Telegram alert |
|  **Message Storage** | All WhatsApp messages saved to `~/.openclaw/workspace/.whatsapp-messages/messages.jsonl` |
|  **Continuous Monitoring** | Runs every 5 minutes via OpenClaw cron jobs (not launchd/scripts) |

---

## How It Works

```
WhatsApp → WAHA (Docker) → messages.jsonl
                              ↓
                        (every 5 min)
                              ↓
                    OpenClaw Cron Jobs
                              ↓
                    Claude AI Analysis
                              ↓
                    Telegram Alerts
                              ↓
                      Your Telegram
```

### How AI Analysis Works

The 2 cron jobs run OpenClaw `agentTurn` tasks that spawn isolated AI agents:

- **Your configured agent analyzes** → no regex, pure AI understanding
- **Job 1: WhatsApp Smart Analyzer** → reads messages, detects appointments, asks for calendar additions
- **Job 2: Important Messages** → assesses message importance, sends alerts only when truly relevant
- **Each job runs every 5 minutes** using your OpenClaw instance

Your agent handles:
- Appointment detection ("meeting", "rdv", "reunion" patterns + context)
- Importance assessment (URGENT keywords + tone analysis)
- Google Calendar prompts (optional integration)

**Note:** Uses whatever agent/model you configured in OpenClaw (Claude, Gemini, or custom)

### Data Flow

```
WhatsApp → WAHA (Docker) → messages.jsonl
                              ↓
                        (every 5 min)
                              ↓
                    OpenClaw Cron Job
                              ↓
                    Your Configured AI Agent
                    (Claude, Gemini, etc)
                              ↓
                    Telegram Bot API
                              ↓
                      Your Telegram
```

---

## After Setup

### Verify It's Running
```bash
# Check OpenClaw cron jobs
cron list

# Should show:
#  WhatsApp Smart Analyzer
#  Important Messages
```

### Check Message Store
```bash
# View latest messages
tail ~/.openclaw/workspace/.whatsapp-messages/messages.jsonl | jq '.'

# Count total messages
wc -l ~/.openclaw/workspace/.whatsapp-messages/messages.jsonl
```

### View Alerts Log
```bash
# See sent alerts
tail ~/.openclaw/workspace/.whatsapp-messages/alerts.log
```

### Test It
**Send a WhatsApp message:**
```
"meeting tomorrow at 3pm"
```

**You'll get Telegram:**
```
 Meeting detected
Day: tomorrow
Time: 3:00 PM

Tu veux que j'ajoute ça à Google Calendar? (oui/non)
```

---

## Configuration

### Customize Detection

Edit the cron jobs in OpenClaw:

```bash
cron list  # Get job IDs
cron update <job-id> --patch '{"payload":{"message":"YOUR NEW PROMPT"}}'
```

### Add Contact-Specific Monitoring
You can create additional cron jobs for specific contacts:
```bash
cron add --job '{
  "name": "Monitor Contact",
  "schedule": {"kind": "every", "everyMs": 300000},
  "payload": {"kind": "agentTurn", "message": "Check messages from specific contact..."},
  "sessionTarget": "isolated"
}'
```

---

## Troubleshooting

### Not getting alerts?
```bash
# Check if WAHA is running
docker ps | grep waha

# Check messages are being stored
tail ~/.openclaw/workspace/.whatsapp-messages/messages.jsonl
```

### False positives?
Claude is smart, but you can refine by updating the cron job prompts.

### Messages not arriving in WAHA?
1. Scan QR code again in WAHA dashboard
2. Check webhook is configured correctly
3. See `references/TROUBLESHOOTING.md`

---

## Files

- `setup.sh` — Installation script (does everything)
- `scripts/` — Helper scripts (mostly deprecated, using cron now)
- `.whatsapp-messages/` — Message storage
- `references/` — Advanced docs
- `LICENSE.md` — CC BY-ND-NC 4.0

---

## What You DON'T Have

 Bash regex nightmares  
 Launchd daemons cluttering your system  
 False positives from bad patterns  
 Manual message parsing  

---

## What You DO Have

 Claude AI analyzing messages intelligently  
 Clean OpenClaw cron jobs (every 5 min)  
 Telegram alerts with full details  
 Google Calendar integration (on-demand)  
 Contact-specific filtering (Joséphine)  

---

## License

**CC BY-ND-NC 4.0** — Non-commercial, no modifications allowed

Personal use:   
Share unmodified:   
Commercial:   
Modifications:   

See `LICENSE.md` for details.

---

## Links

- WAHA: https://waha.devlike.pro/
- OpenClaw: https://docs.openclaw.ai/
- ClawhHub: https://clawhub.com/
