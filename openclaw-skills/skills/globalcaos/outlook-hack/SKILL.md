---
name: outlook-hack
version: 1.0.0
description: "Access Outlook email via the browser relay — read, search, send, reply to emails and check calendar without separate auth. Piggybacks on the existing Outlook Web session in Chrome. When IMAP is blocked and Microsoft Graph API requires admin consent you can't get, this is the third option: just open Outlook in Chrome, click the relay button, and your agent has full email access. Zero API keys, zero admin approval, zero config."
homepage: https://github.com/globalcaos/clawdbot-moltbot-openclaw
repository: https://github.com/globalcaos/clawdbot-moltbot-openclaw
metadata:
  openclaw:
    emoji: "📧"
    requires:
      tools: ["browser"]
---

# Outlook Hack — When IMAP is Blocked and APIs Need Admin Consent

*The skill born from corporate IT frustration. If you can open Outlook in Chrome, your agent can read your email.*

## The Problem

Corporate Outlook access is locked down three ways:

1. **IMAP/SMTP** — disabled by IT policy (most Microsoft 365 tenants)
2. **Microsoft Graph API** — requires Azure AD app registration + admin consent (good luck getting that approved)
3. **EWS (Exchange Web Services)** — deprecated and increasingly blocked

If you're an OpenClaw user with a corporate Microsoft 365 account, you're stuck — until now.

## The Third Option

This skill piggybacks on your existing Outlook Web session in Chrome via the OpenClaw browser relay. No API keys, no admin consent, no IMAP. If you can read email in your browser, your agent can too.

**How it works:**

1. Open Outlook Web (`https://outlook.office.com`) in Chrome
2. Click the OpenClaw Browser Relay toolbar button (badge ON)
3. Your agent executes `fetch()` calls inside the tab using the MSAL access token that's already there

The Outlook Web app stores OAuth tokens in `localStorage`. This skill extracts them and calls the Outlook REST API v2.0 directly from the browser context. No tokens leave the browser — all API calls happen within Chrome's security sandbox.

## Prerequisites

- OpenClaw Browser Relay extension installed in Chrome ([docs](https://docs.openclaw.ai/tools/chrome-extension))
- Outlook Web tab open and logged in
- Relay attached to the Outlook tab (toolbar button → badge ON)

## Capabilities

| Feature | Supported |
|---------|-----------|
| Read inbox (with preview) | ✅ |
| Read full message body (HTML) | ✅ |
| Search messages | ✅ |
| Send email | ✅ |
| Reply / Reply All / Forward | ✅ |
| List folders + unread counts | ✅ |
| Calendar events | ✅ |
| Download attachments | ✅ |
| Mark read/unread | ✅ |
| Move to folder | ✅ |
| Flag messages | ✅ |

## Usage

### Finding the Outlook Tab

First, find the attached Outlook tab:

```
browser action=tabs profile=chrome
```

Look for a tab with URL containing `outlook.office.com`. Note the `targetId`.

### Token Extraction

All API calls start by extracting the MSAL access token from localStorage:

```javascript
const tokenKey = Object.keys(localStorage).find(k =>
  k.includes('accesstoken') &&
  k.includes('outlook.office.com') &&
  k.includes('mail.readwrite')
);
const token = JSON.parse(localStorage.getItem(tokenKey)).secret;
```

Use this token as `Authorization: Bearer <token>` for all Outlook REST API calls.

### API Base URL

```
https://outlook.office.com/api/v2.0/me/
```

### List Inbox Messages

```javascript
async () => {
  // Extract token
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;

  const resp = await fetch(
    'https://outlook.office.com/api/v2.0/me/messages?' +
    '$top=20&$select=Subject,From,ReceivedDateTime,IsRead,BodyPreview,Id' +
    '&$orderby=ReceivedDateTime desc',
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  const data = await resp.json();
  return data.value?.map(m => ({
    id: m.Id,
    subject: m.Subject,
    from: m.From?.EmailAddress?.Name,
    email: m.From?.EmailAddress?.Address,
    date: m.ReceivedDateTime,
    read: m.IsRead,
    preview: m.BodyPreview?.substring(0, 150)
  }));
}
```

Add `&$filter=IsRead eq false` to list only unread messages.

### Read Full Message

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const messageId = '<MESSAGE_ID>';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages/${messageId}?` +
    '$select=Subject,From,ToRecipients,CcRecipients,Body,ReceivedDateTime,HasAttachments',
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  return await resp.json();
}
```

The `Body.Content` field contains full HTML.

### Search Messages

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const query = 'invoice January';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages?` +
    `$search="${encodeURIComponent(query)}"&$top=10` +
    '&$select=Subject,From,ReceivedDateTime,BodyPreview',
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  const data = await resp.json();
  return data.value;
}
```

### Send Email

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;

  const resp = await fetch('https://outlook.office.com/api/v2.0/me/sendmail', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer ' + token,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      Message: {
        Subject: 'Subject here',
        Body: { ContentType: 'HTML', Content: '<p>Email body here</p>' },
        ToRecipients: [{ EmailAddress: { Address: 'recipient@example.com' } }],
        CcRecipients: []  // optional
      }
    })
  });
  return { status: resp.status, ok: resp.ok };
}
```

### Reply to Message

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const messageId = '<MESSAGE_ID>';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages/${messageId}/reply`,
    {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ Comment: '<p>Reply text here</p>' })
    }
  );
  return { status: resp.status, ok: resp.ok };
}
```

Use `/replyall` instead of `/reply` for Reply All. Use `/forward` with a `ToRecipients` array for forwarding.

### Calendar Events

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const now = new Date().toISOString();
  const end = new Date(Date.now() + 7 * 86400000).toISOString();

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/calendarview?` +
    `startdatetime=${now}&enddatetime=${end}` +
    '&$select=Subject,Start,End,Location,Organizer,IsAllDay' +
    '&$orderby=Start/DateTime',
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  const data = await resp.json();
  return data.value;
}
```

### List Folders

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;

  const resp = await fetch(
    'https://outlook.office.com/api/v2.0/me/mailfolders?' +
    '$select=DisplayName,UnreadItemCount,TotalItemCount',
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  const data = await resp.json();
  return data.value;
}
```

### Download Attachment

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const messageId = '<MESSAGE_ID>';
  const attachmentId = '<ATTACHMENT_ID>';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages/${messageId}/attachments/${attachmentId}`,
    { headers: { 'Authorization': 'Bearer ' + token } }
  );
  const data = await resp.json();
  // data.ContentBytes = base64-encoded file content
  // data.Name = filename
  // data.ContentType = MIME type
  return { name: data.Name, type: data.ContentType, size: data.ContentBytes?.length };
}
```

### Mark as Read/Unread

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const messageId = '<MESSAGE_ID>';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages/${messageId}`,
    {
      method: 'PATCH',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ IsRead: true })  // false for unread
    }
  );
  return { status: resp.status, ok: resp.ok };
}
```

### Move to Folder

```javascript
async () => {
  const tk = Object.keys(localStorage).find(k => k.includes('accesstoken') && k.includes('outlook.office.com') && k.includes('mail.readwrite'));
  const token = JSON.parse(localStorage.getItem(tk)).secret;
  const messageId = '<MESSAGE_ID>';

  const resp = await fetch(
    `https://outlook.office.com/api/v2.0/me/messages/${messageId}/move`,
    {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ DestinationId: 'Archive' })  // or folder ID
    }
  );
  return { status: resp.status, ok: resp.ok };
}
```

## Implementation Pattern

Always use the browser tool with `profile="chrome"`:

```
browser action=act profile=chrome targetId=<outlook-tab-id>
  request.kind=evaluate
  request.fn=<async function>
```

## Token Refresh

MSAL auto-refreshes tokens in the background while the Outlook tab is open. If a call returns 401:

1. Wait 2 seconds (MSAL may be refreshing)
2. Re-extract the token from localStorage (it will have been updated)
3. Retry the call

Tokens expire after ~1 hour but are refreshed automatically.

## Comparison with Other Approaches

| Feature | IMAP | Graph API | EWS | **Outlook Hack** |
|---------|------|-----------|-----|------------------|
| Admin consent needed | Often blocked | ✅ Required | Often blocked | ❌ None |
| API keys needed | ✅ Credentials | ✅ App registration | ✅ Credentials | ❌ None |
| Works with MFA | ❌ Breaks IMAP | ✅ | ❌ Often breaks | ✅ Via browser session |
| Calendar access | ❌ | ✅ | ✅ | ✅ |
| Send email | ✅ | ✅ | ✅ | ✅ |
| Search | ❌ Limited | ✅ | ✅ | ✅ |
| Setup time | Medium | Hours/days | Medium | **2 minutes** |
| Requires browser open | ❌ | ❌ | ❌ | ✅ |

## Limitations

- Requires Chrome with the Outlook tab open (tab can be background)
- Token expires if Outlook tab is closed for extended periods
- Large attachments (>25MB) may be slow via base64 encoding
- Microsoft rate limits apply (~10K requests per 10 minutes)
- Cannot access other users' mailboxes (only your own)

## Security Notes

- No tokens are stored outside the browser
- All API calls happen within Chrome's security context
- The relay bridges CDP commands only — auth stays in Chrome's cookie jar
- No credentials are written to disk by this skill

---

## Credits

Created by **Oscar Serra** with the help of **Claude** (Anthropic).

*Born at 2 AM on a Sunday, because corporate IT said "no" to IMAP, "no" to Graph API, and "submit a ticket" for everything else.*
