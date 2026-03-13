# Cron Jobs - Automated Scanning

Three jobs run every 5 minutes and send Telegram alerts.

## Job 1: Appointment Detector 

**Looks for:** "appointment", "meeting", "scheduled" + times (2:30 PM, 10 AM, etc)

**Sends:** ` Found appointment: 2:30 PM at doctor — Add to calendar? (yes/no)`

**Status:** Already created 

ID: `89e6c0ad-ec35-403c-8697-5d2cc7e7cb43`

---

## Job 2: Important Message Alert 

**Looks for:**
- Keywords: "URGENT", "ASAP", "HELP", "SOS", "blocked", "problem", "error"
- Style: ALL CAPS, multiple `!!!` or `???`

**Sends:** ` IMPORTANT MESSAGE from John: "URGENT CALL ME BACK!!!"`

**Status:** Already created 

ID: `1eb55cf9-1776-4088-8643-81de370a2529`

---

## Job 3: Contact Handler (Joséphine) 

**Watches:** Messages from Joséphine (33662028118@c.us)

**Flow:**
1. Message arrives → Telegram alert: ` Message from Joséphine: "Hi!"`
2. AI suggests warm response: ` Suggested: "Hey! How are you?"`
3. You reply yes/no
4. If yes → sent via WhatsApp 
5. If no → ignored 

**Status:** Already created 

ID: `06758784-662f-4223-9690-cd0ea1e58037`

---

## Check Jobs Are Running

```bash
openclaw cron list
```

Should show all three with status 

---

## Disable a Job

```bash
openclaw cron remove <job-id>
```

Example:
```bash
openclaw cron remove 89e6c0ad-ec35-403c-8697-5d2cc7e7cb43
```

---

## Test Appointment Detection

Send yourself a WhatsApp message:
```
Appointment Wednesday 2:30 PM at the doctor
```

Within 5 minutes, check Telegram for alert.

---

## Customize Keywords

Want to add more keywords? Edit the job message. See `references/ADVANCED.md` for how.

---

## Troubleshooting

**Cron jobs not firing?**

1. Check jobs exist: `openclaw cron list`
2. Check Telegram is connected: `openclaw status`
3. Check messages are being stored: `tail ~/.openclaw/workspace/.whatsapp-messages/messages.jsonl`

**Still not working?** See `TROUBLESHOOTING.md`
