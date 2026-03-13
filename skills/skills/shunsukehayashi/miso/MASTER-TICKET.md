# Master Ticket Auto-Management — Logic Design

## State Management

### Storage Location
```
~/.openclaw/workspace/skills/mission-control/.miso-state.json
```

### Data Structure
```json
{
  "masterTicketId": "string",
  "chatId": "string",
  "missions": [
    {
      "id": "string",
      "messageId": "string",
      "status": "pending|running|complete|error",
      "title": "string",
      "agentCount": number,
      "startedAt": "ISO8601",
      "completedAt": "ISO8601|null"
    }
  ]
}
```

## Auto-Update Triggers

| Event | Action |
|-------|--------|
| Mission start | Add new row (` #{id} {title} (pending)`) |
| Mission running | Update status (` #{id} {title} (running)`) |
| Mission complete | Update status (` #{id} {title} (complete)`) |
| Mission error | Update status (` #{id} {title} (error)`) |
| All missions complete | Show summary + unpin |

## Templates

### Master Ticket (Active)
```
 𝗠𝗜𝗦𝗦𝗜𝗢𝗡 𝗖𝗢𝗡𝗧𝗥𝗢𝗟
——————————————
 #1 Title (pending)
 #2 Title (running)
 #3 Title (complete)
——————————————
Updated: 2026-02-17 08:57:00 JST
 ᴘᴏᴡᴇʀᴇᴅ ʙʏ ᴍɪʏᴀʙɪ
```

### Master Ticket (All Complete)
```
 𝗠𝗜𝗦𝗦𝗜𝗢𝗡 𝗖𝗢𝗠𝗣𝗟𝗘𝗧𝗘 
——————————————
 #1 Title (complete)
 #2 Title (complete)
 #3 Title (complete)
——————————————
All missions complete: 3/3
Updated: 2026-02-17 08:57:00 JST
 ᴘᴏᴡᴇʀᴇᴅ ʙʏ ᴍɪʏᴀʙɪ
```

## Initial Creation Flow

1. If `.miso-state.json` doesn't exist → create new
2. If `.miso-state.json` has no `masterTicketId` → post new message
3. Pin the new message on creation
4. Save `masterTicketId` and `chatId` to `.miso-state.json`

## Daily Archive Rules

### Trigger Conditions
- Time: After 23:59
- All mission statuses are `complete` or `error`

### Actions
1. Unpin master ticket
2. Reset `.miso-state.json` to empty state (`{}`)

## API Operations

### Message Edit
```javascript
// Use message.edit() to update master ticket
// messageId = masterTicketId
```

### Pin Operations
```javascript
// Pin
message.pin({ disable_notification: true })

// Unpin
message.unpin()
```
