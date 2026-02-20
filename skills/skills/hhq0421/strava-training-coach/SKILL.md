---
name: strava-training-coach
description: |
  Security-hardened AI training coach that prevents running injuries before they happen. 
  Monitors Strava data for dangerous mileage spikes, intensity imbalances, and recovery gaps — 
  then sends smart alerts to Discord or Slack.

  Use when:
  - Monitoring training load to prevent overtraining and injury
  - Setting up automated weekly training reports with trend analysis  
  - Receiving alerts when weekly mileage or intensity spikes dangerously
  - Tracking long-term fitness trends and recovery patterns
  - Getting notified of meaningful achievements (PRs, consistency milestones)

  Security features:
  - No hardcoded secrets - all credentials via environment variables
  - Input validation on all external data
  - Sensitive data redaction in logs
  - Rate limiting on notifications
  - Secure file permissions on token storage
  - Webhook URL validation
homepage: https://developers.strava.com/docs/reference/
metadata: {"clawdbot":{"emoji":"🏃","tags":["fitness","strava","running","injury-prevention","training","alerts","discord","slack","security"],"requires":{"env":["STRAVA_CLIENT_ID","STRAVA_CLIENT_SECRET","DISCORD_WEBHOOK_URL or SLACK_WEBHOOK_URL"]}}}
---

# Strava Training Coach

AI training partner that catches injury risk before you feel it.

## Why This Matters

Most running injuries follow the same pattern: too much, too soon. By the time you feel pain, the damage is weeks old. This coach watches your Strava data daily and alerts you **before** problems become injuries — so you stay consistent instead of sidelined.

Built on the 80/20 principle: 80% easy, 20% hard. The same approach used by elite coaches to build durable athletes.

## What You Get

- **Acute Load Alerts** — Weekly mileage up 30%+? You'll know before your knees do
- **Intensity Checks** — Too many hard days eroding recovery
- **Recovery Nudges** — Extended gaps that might need a gentle return
- **Smart PRs** — Meaningful progress, accounting for terrain and conditions
- **Weekly Reports** — Sunday trends, not just totals

## Quick Start

### 1. Connect Strava

```bash
# Set your Strava API credentials (required)
export STRAVA_CLIENT_ID=your_id
export STRAVA_CLIENT_SECRET=your_secret

# Authenticate (opens browser for OAuth)
python3 scripts/auth.py
```

Tokens are stored securely in `~/.config/strava-training-coach/` with 0600 permissions.

### 2. Set Up Notifications (Required)

**Discord:**
```bash
export DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
export NOTIFICATION_CHANNEL=discord
```

**Slack:**
```bash
export SLACK_WEBHOOK_URL=https://hooks.slack.com/...
export NOTIFICATION_CHANNEL=slack
```

⚠️ **Security:** Webhook URLs must be set via environment variables. No hardcoded URLs allowed.

### 3. Optional: Enable Oura Integration

```bash
export OURA_ENABLED=true
```

Requires Oura CLI authentication.

### 4. Run

```bash
# Daily training check + alerts
python3 scripts/coach_check.py

# Weekly summary report  
python3 scripts/weekly_report.py
```

Optional: schedule with cron for hands-off monitoring:

```json
{
  "name": "Training Coach - Daily Check",
  "schedule": {"kind": "every", "everyMs": 86400000},
  "command": "python3 scripts/coach_check.py"
}
```

## Security Features

This skill is designed with security in mind for ClawHub publication:

### Credential Handling
- **No hardcoded secrets** — All credentials via environment variables
- **Secure token storage** — Tokens saved with 0600 permissions
- **XDG compliance** — Config stored in `~/.config/strava-training-coach/`
- **Token validation** — Structure validation before use

### Input Validation
- **Date format validation** — ISO8601 format checking
- **Numeric range validation** — All thresholds bounded
- **Type checking** — Safe type conversion with defaults
- **Webhook URL validation** — Pattern matching for Discord/Slack

### Data Protection
- **Log redaction** — Sensitive data masked in logs
- **Secure temp files** — Proper permissions on state files
- **No data leakage** — Safe error messages
- **Rate limiting** — Max 1 alert per hour per type

### Network Security
- **HTTPS only** — All API calls use TLS
- **Timeout handling** — 30-second timeouts on all requests
- **Retry logic** — 3 attempts with exponential backoff
- **Certificate validation** — Standard SSL verification

## Configuration

All thresholds are optional — sensible defaults with validation.

```bash
# Training thresholds (validated ranges)
MAX_WEEKLY_MILEAGE_JUMP=30     # 5-100%, default: 30
MAX_HARD_DAY_PERCENTAGE=25     # 5-100%, default: 25
MIN_EASY_RUN_HEART_RATE=145    # 100-200 bpm, default: 145

# Feature flags
OURA_ENABLED=false             # Enable Oura integration
distVERBOse=false              # Enable debug logging
```

## Example Alerts

### Injury Risk

> "⚠️ Training Load Alert: Weekly mileage up 45% (18→26 mi). Risk of injury increases significantly above 10% weekly gains. Consider an easy week."

> "🫁 Easy Days Too Hard: 60% of this week's runs were moderate/high effort. Easy days should feel conversational (HR <145)."

> "💤 Rest Day Streak: 5 days since last activity. A gentle 20-min walk or yoga can aid recovery."

### Achievements

> "🎉 New Best Effort: 5K in 22:30 — your fastest flat road run this year!"

> "🔥 30-Day Streak: 30 days of movement. Consistency beats intensity."

> "✅ Base Building Complete: 4 weeks of 80/80 easy running. Ready to add structured workouts."

### Weekly Reports (Sunday)

- Weekly mileage vs. target
- Intensity distribution (easy/moderate/hard)
- 4-week trend
- Recommended focus for next week

## Training Philosophy

1. **The 80/20 Rule** — 80% easy, 20% hard
2. **10% Weekly Rule** — Max 10% mileage increase per week
3. **Consistency First** — Show up regularly, don't crush sporadically
4. **Listen Early** — Catch warning signs before they become injuries

See `references/training-principles.md` for the full injury prevention guide.

## Files

- `scripts/auth.py` — Strava OAuth setup
- `scripts/coach_check.py` — Daily training analysis and alerts
- `scripts/weekly_report.py` — Sunday summary reports
- `scripts/refresh_token.py` — Token refresh for expired sessions
- `references/training-principles.md` — Injury prevention guide

## Smart, Not Spammy

Alerts fire only when something matters:
- Mileage spike detected
- Intensity pattern concerning
- Meaningful PR achieved
- Weekly summary ready

Not every workout. That's what Strava is for.

## Rate Limits

- 1-2 API calls per check
- Strava allows 100 req/15 min, 1000/day
- Daily checks use ~30 requests/month
