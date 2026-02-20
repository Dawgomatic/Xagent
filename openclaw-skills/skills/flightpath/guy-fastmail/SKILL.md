---
name: guy-fastmail
description: Access Guy's Fastmail email, contacts, and file storage via JMAP API and WebDAV. Use when the user wants to read, send, search, reply to, or manage email. Also use for contact lookup, searching contacts, or managing files in Fastmail file storage. Supports mailbox management, archiving, flagging, and more.
---

# Fastmail (JMAP API)

Full email and contacts access via Fastmail's JMAP API.

## Script

`scripts/jmap.sh <action> [args...]`

### Mail Actions

- `mailboxes` — List all mailboxes with unread/total counts
- `inbox [count]` — List recent inbox emails (default 20)
- `smart-inbox [timeframe] [count]` — Smart folder: recent emails from ALL mailboxes (timeframe: today-yesterday|this-week|last-week, default: today-yesterday, count: default 50)
- `mailbox-emails <mailbox-id> [count]` — List emails from a specific mailbox (default 100)
- `unread [count]` — List unread emails (default 20)
- `read <email-id>` — Read full email by ID
- `search <query> [count]` — Search emails by text (default 20 results)
- `send <to> <subject> <body> [cc]` — Send a new email
- `reply <email-id> <body> [reply-all]` — Reply to an email (reply-all: true/false)
- `move <email-id> <mailbox-id>` — Move email to a mailbox (use `mailboxes` to get IDs)
- `archive <email-id>` — Archive an email
- `delete <email-id>` — Move email to trash
- `batch-delete` — Move multiple emails to trash (reads email IDs from stdin, one per line)
- `batch-destroy` — Permanently delete multiple emails (reads email IDs from stdin, one per line)
- `create-mailbox <name> [parent-id]` — Create a new mailbox/folder
- `flag <email-id> <flag> <true|false>` — Set flag (seen/flagged/answered)

### Masked Email Actions

- `masked-list [count]` — List all masked email addresses (default 50)
- `masked-search <query>` — Search masked emails by address, domain, or description
- `masked-create [domain] [description]` — Create a new masked email address
- `masked-enable <id>` — Re-enable a disabled masked email
- `masked-disable <id>` — Disable a masked email (stops forwarding)
- `masked-delete <id>` — Delete a masked email (confirm with user first!)

### File Actions (WebDAV)

- `files-list [path]` — List files/folders (default /)
- `files-upload <local-path> <remote-path>` — Upload a file
- `files-download <remote-path> <local-path>` — Download a file
- `files-mkdir <path>` — Create a folder
- `files-delete <path>` — Delete a file or folder (confirm with user first!)

### Contact Actions

- `contacts [count]` — List contacts (default 50)
- `contact-search <query>` — Search contacts by name, email, or phone

### Calendar Actions (CalDAV)

- `calendars` — List all calendars
- `calendar-create <name> [color]` — Create a new calendar (color: hex like #FF0000)
- `calendar-delete <calendar-id>` — Delete a calendar and ALL its events (confirm with user first!)
- `events [days] [calendar-id]` — List upcoming events (default 14 days, all calendars)
- `event-search <query> [days]` — Search events by text (default ±90 days)
- `event-add <calendar-id> <title> <start> <end> [location] [description]` — Create event (start/end: `YYYY-MM-DDTHH:MM` for timed, `YYYY-MM-DD` for all-day; times are CT)
- `event-get <calendar-id> <uid>` — Get full event details
- `event-edit <calendar-id> <uid> <field> <value>` — Edit event field (title|start|end|location|description)
- `event-delete <calendar-id> <uid>` — Delete an event (confirm with user first!)

### Examples

```bash
# Check unread emails
bash scripts/jmap.sh unread

# Smart inbox: see what arrived today/yesterday across all folders
bash scripts/jmap.sh smart-inbox

# Smart inbox: see what arrived this week
bash scripts/jmap.sh smart-inbox this-week

# Read a specific email
bash scripts/jmap.sh read "M1234abc"

# Search for emails about invoices
bash scripts/jmap.sh search "invoice"

# Send an email
bash scripts/jmap.sh send "someone@example.com" "Hello" "Hey, how are you?"

# Reply to an email
bash scripts/jmap.sh reply "M1234abc" "Thanks, got it!"

# List emails from a specific mailbox
bash scripts/jmap.sh mailbox-emails "POV" 50

# Batch delete emails (from a file with email IDs)
cat email_ids.txt | bash scripts/jmap.sh batch-delete

# Permanently delete emails from trash
cat trash_ids.txt | bash scripts/jmap.sh batch-destroy

# Find a contact
bash scripts/jmap.sh contact-search "Vivien"
```

## Notes

- Email IDs are shown in brackets like `[M1234abc]` in list output — use those for read/reply/move/etc.
- Always confirm with Guy before sending emails
- For replies, quote relevant context in the body if appropriate
- The `search` action searches subject, body, from, and to fields
- Calendar access uses CalDAV with a separate app password
- Himalaya CLI is installed as a fallback for IMAP access if needed
