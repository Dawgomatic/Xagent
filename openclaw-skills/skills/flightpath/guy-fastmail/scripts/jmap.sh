#!/usr/bin/env bash
# Fastmail JMAP API client
set -euo pipefail

ACTION="${1:?Usage: jmap.sh <action> [args...]}"
shift

API_TOKEN="fmu1-fb6140a5-ead6d7914d25b4016dfbf38af438adec-0-753ad070bb4c0d390695c55ed4e77cf0"
API_URL="https://api.fastmail.com/jmap/api/"
ACCOUNT_ID="ufb6140a5"

jmap_call() {
  local body="$1"
  curl -s -X POST "$API_URL" \
    -H "Authorization: Bearer $API_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$body"
}

case "$ACTION" in

  ## ── MAIL ──────────────────────────────────────

  mailboxes)
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Mailbox/get", {"accountId": "'"$ACCOUNT_ID"'"}, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
for mb in r['methodResponses'][0][1]['list']:
    count = mb.get('totalEmails', 0)
    unread = mb.get('unreadEmails', 0)
    print(f\"{mb['name']} ({unread} unread / {count} total) [id: {mb['id']}]\")
"
    ;;

  inbox)
    # List recent inbox emails. $1=count (default 20)
    COUNT="${1:-20}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [
        ["Email/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {"inMailbox": null},
          "sort": [{"property": "receivedAt", "isAscending": false}],
          "limit": '"$COUNT"'
        }, "0"],
        ["Email/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Email/query", "path": "/ids"},
          "properties": ["id", "subject", "from", "receivedAt", "preview", "keywords"]
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
# Get inbox ID first if needed
emails = r['methodResponses'][1][1].get('list', [])
for e in emails:
    fr = e.get('from', [{}])[0]
    sender = fr.get('name') or fr.get('email', '?')
    subj = e.get('subject', '(no subject)')
    date = e.get('receivedAt', '')[:16]
    unread = '●' if '\$seen' not in e.get('keywords', {}) else ' '
    eid = e.get('id', '')
    print(f'{unread} {date} | {sender} | {subj} [{eid}]')
"
    ;;

  smart-inbox)
    # Smart folder view: recent emails from ALL mailboxes. $1=timeframe (today-yesterday|this-week|last-week), $2=count (default 50)
    TIMEFRAME="${1:-today-yesterday}"
    COUNT="${2:-50}"
    
    case "$TIMEFRAME" in
      today-yesterday)
        START_DATE=$(date -u -d "2 days ago" +"%Y-%m-%dT00:00:00Z")
        END_DATE=$(date -u -d "tomorrow" +"%Y-%m-%dT00:00:00Z")
        TITLE="📧 Smart Inbox: Today & Yesterday"
        ;;
      this-week)
        # Start of this week (Monday)
        START_DATE=$(date -u -d "last monday" +"%Y-%m-%dT00:00:00Z")
        END_DATE=$(date -u -d "tomorrow" +"%Y-%m-%dT00:00:00Z")
        TITLE="📧 Smart Inbox: This Week"
        ;;
      last-week)
        # Last Monday to Sunday
        START_DATE=$(date -u -d "2 monday ago" +"%Y-%m-%dT00:00:00Z")
        END_DATE=$(date -u -d "last monday" +"%Y-%m-%dT00:00:00Z")
        TITLE="📧 Smart Inbox: Last Week"
        ;;
      *)
        echo "Unknown timeframe: $TIMEFRAME"
        echo "Options: today-yesterday, this-week, last-week"
        exit 1
        ;;
    esac

    echo "$TITLE"
    echo "$(echo "$TITLE" | sed 's/./=/g')"
    echo ""

    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [
        ["Email/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {
            "after": "'"$START_DATE"'",
            "before": "'"$END_DATE"'"
          },
          "sort": [{"property": "receivedAt", "isAscending": false}],
          "limit": '"$COUNT"'
        }, "0"],
        ["Email/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Email/query", "path": "/ids"},
          "properties": ["id", "subject", "from", "receivedAt", "preview", "keywords", "mailboxIds"]
        }, "1"],
        ["Mailbox/get", {"accountId": "'"$ACCOUNT_ID"'"}, "2"]
      ]
    }' | python3 -c "
import sys,json
from datetime import datetime

r=json.load(sys.stdin)
emails = r['methodResponses'][1][1].get('list', [])
mailboxes = r['methodResponses'][2][1].get('list', [])

# Build mailbox ID -> name mapping
mb_names = {}
for mb in mailboxes:
    mb_names[mb['id']] = mb['name']

if not emails:
    print('No emails found in this timeframe')
else:
    for e in emails:
        fr = e.get('from', [{}])[0]
        sender = fr.get('name') or fr.get('email', '?')
        subj = e.get('subject', '(no subject)')
        date_str = e.get('receivedAt', '')[:16]
        unread = '●' if '\$seen' not in e.get('keywords', {}) else ' '
        
        # Find which mailbox this email is in
        mailbox_ids = list(e.get('mailboxIds', {}).keys())
        folder = 'Unknown'
        if mailbox_ids:
            folder = mb_names.get(mailbox_ids[0], mailbox_ids[0][:8])
        
        # Format: unread indicator, date, folder, sender, subject [email ID for linking]
        email_id = e.get('id', '')
        print(f'{unread} {date_str} | {folder:<15} | {sender:<25} | {subj} [{email_id}]')
"
    ;;

  mailbox-emails)
    # List emails from a specific mailbox. $1=mailbox-id, $2=count (default 100)
    MAILBOX_ID="${1:?Usage: jmap.sh mailbox-emails <mailbox-id> [count]}"
    COUNT="${2:-100}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [
        ["Email/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {"inMailbox": "'"$MAILBOX_ID"'"},
          "sort": [{"property": "receivedAt", "isAscending": false}],
          "limit": '"$COUNT"'
        }, "0"],
        ["Email/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Email/query", "path": "/ids"},
          "properties": ["id", "subject", "from", "receivedAt", "preview", "keywords"]
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
emails = r['methodResponses'][1][1].get('list', [])
for e in emails:
    fr = e.get('from', [{}])[0]
    sender = fr.get('name') or fr.get('email', '?')
    subj = e.get('subject', '(no subject)')
    date = e.get('receivedAt', '')[:16]
    unread = '●' if '\$seen' not in e.get('keywords', {}) else ' '
    eid = e.get('id', '')
    print(f'{unread} {date} | {sender} | {subj} [{eid}]')
"
    ;;

  unread)
    # List unread emails. $1=count (default 20)
    COUNT="${1:-20}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [
        ["Email/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {"notKeyword": "$seen"},
          "sort": [{"property": "receivedAt", "isAscending": false}],
          "limit": '"$COUNT"'
        }, "0"],
        ["Email/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Email/query", "path": "/ids"},
          "properties": ["id", "subject", "from", "receivedAt", "preview"]
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
emails = r['methodResponses'][1][1].get('list', [])
if not emails:
    print('No unread emails!')
else:
    print(f'{len(emails)} unread:')
    for e in emails:
        fr = e.get('from', [{}])[0]
        sender = fr.get('name') or fr.get('email', '?')
        subj = e.get('subject', '(no subject)')
        date = e.get('receivedAt', '')[:16]
        print(f'  {date} | {sender} | {subj} [{e[\"id\"]}]')
"
    ;;

  read)
    # Read an email by ID. $1=email ID
    EMAIL_ID="${1:?Usage: jmap.sh read <email-id>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Email/get", {
        "accountId": "'"$ACCOUNT_ID"'",
        "ids": ["'"$EMAIL_ID"'"],
        "properties": ["id", "subject", "from", "to", "cc", "receivedAt", "textBody", "bodyValues", "preview"],
        "fetchTextBodyValues": true
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
emails = r['methodResponses'][0][1].get('list', [])
if not emails:
    print('Email not found')
    sys.exit(0)
e = emails[0]
fr = e.get('from', [{}])[0]
to = ', '.join(t.get('name') or t.get('email','') for t in e.get('to', []))
cc = ', '.join(t.get('name') or t.get('email','') for t in e.get('cc', []))
print(f'From: {fr.get(\"name\",\"\")} <{fr.get(\"email\",\"\")}>')
print(f'To: {to}')
if cc: print(f'Cc: {cc}')
print(f'Date: {e.get(\"receivedAt\", \"\")}')
print(f'Subject: {e.get(\"subject\", \"\")}')
print('---')
for part in e.get('textBody', []):
    pid = part.get('partId', '')
    body = e.get('bodyValues', {}).get(pid, {}).get('value', '')
    print(body)
"
    ;;

  search)
    # Search emails. $1=query, $2=count (default 20)
    QUERY="${1:?Usage: jmap.sh search <query> [count]}"
    COUNT="${2:-20}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [
        ["Email/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {"text": "'"$QUERY"'"},
          "sort": [{"property": "receivedAt", "isAscending": false}],
          "limit": '"$COUNT"'
        }, "0"],
        ["Email/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Email/query", "path": "/ids"},
          "properties": ["id", "subject", "from", "receivedAt", "preview"]
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
emails = r['methodResponses'][1][1].get('list', [])
if not emails:
    print('No emails found')
else:
    for e in emails:
        fr = e.get('from', [{}])[0]
        sender = fr.get('name') or fr.get('email', '?')
        subj = e.get('subject', '(no subject)')
        date = e.get('receivedAt', '')[:16]
        print(f'{date} | {sender} | {subj} [{e[\"id\"]}]')
"
    ;;

  send)
    # Send an email. $1=to, $2=subject, $3=body, $4=cc (optional)
    TO="${1:?Usage: jmap.sh send <to> <subject> <body> [cc]}"
    SUBJECT="${2:?Usage: jmap.sh send <to> <subject> <body> [cc]}"
    BODY="${3:?Usage: jmap.sh send <to> <subject> <body> [cc]}"
    CC="${4:-}"

    TO_JSON="[{\"email\": \"$TO\"}]"
    CC_JSON="null"
    [ -n "$CC" ] && CC_JSON="[{\"email\": \"$CC\"}]"

    # Create draft then submit
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail", "urn:ietf:params:jmap:submission"],
      "methodCalls": [
        ["Email/set", {
          "accountId": "'"$ACCOUNT_ID"'",
          "create": {
            "draft1": {
              "from": [{"name": "Guy Mackenzie", "email": "guy.mackenzie@macfiver.com"}],
              "to": '"$TO_JSON"',
              "cc": '"$CC_JSON"',
              "subject": "'"$SUBJECT"'",
              "bodyValues": {"body": {"value": "'"$BODY"'", "charset": "utf-8"}},
              "textBody": [{"partId": "body", "type": "text/plain"}],
              "keywords": {"$draft": true}
            }
          }
        }, "0"],
        ["EmailSubmission/set", {
          "accountId": "'"$ACCOUNT_ID"'",
          "onSuccessUpdateEmail": {"#sendIt": {"keywords/$draft": null, "mailboxIds/'"$ACCOUNT_ID"'": null}},
          "create": {
            "sendIt": {
              "emailId": "#draft1",
              "identityId": null
            }
          }
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
for resp in r.get('methodResponses', []):
    if resp[0] == 'EmailSubmission/set':
        created = resp[1].get('created', {})
        errors = resp[1].get('notCreated', {})
        if created:
            print('Email sent successfully!')
        elif errors:
            for k,v in errors.items():
                print(f'Send failed: {v.get(\"description\", v)}')
        else:
            print('Unknown result:', json.dumps(resp[1]))
"
    ;;

  reply)
    # Reply to an email. $1=email-id, $2=body, $3=reply-all (true/false, default false)
    EMAIL_ID="${1:?Usage: jmap.sh reply <email-id> <body> [reply-all]}"
    BODY="${2:?Usage: jmap.sh reply <email-id> <body>}"
    REPLY_ALL="${3:-false}"

    # Fetch original email for headers
    ORIG=$(jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Email/get", {
        "accountId": "'"$ACCOUNT_ID"'",
        "ids": ["'"$EMAIL_ID"'"],
        "properties": ["subject", "from", "to", "cc", "messageId", "references", "inReplyTo"]
      }, "0"]]
    }')

    python3 -c "
import sys, json, subprocess

orig = json.loads('''$ORIG''')
email = orig['methodResponses'][0][1]['list'][0]
subject = email.get('subject', '')
if not subject.lower().startswith('re:'):
    subject = 'Re: ' + subject

reply_to = email.get('from', [])
to_json = json.dumps(reply_to)

cc_json = 'null'
if '$REPLY_ALL' == 'true':
    all_cc = email.get('to', []) + email.get('cc', [])
    all_cc = [r for r in all_cc if r.get('email','').lower() != 'guy.mackenzie@macfiver.com']
    if all_cc:
        cc_json = json.dumps(all_cc)

msg_id = email.get('messageId', [''])[0] if email.get('messageId') else ''
refs = email.get('references', []) or []
if msg_id:
    refs = refs + [msg_id]

body = {
    'using': ['urn:ietf:params:jmap:core', 'urn:ietf:params:jmap:mail', 'urn:ietf:params:jmap:submission'],
    'methodCalls': [
        ['Email/set', {
            'accountId': '$ACCOUNT_ID',
            'create': {
                'draft1': {
                    'from': [{'name': 'Guy Mackenzie', 'email': 'guy.mackenzie@macfiver.com'}],
                    'to': reply_to,
                    'cc': json.loads(cc_json) if cc_json != 'null' else None,
                    'subject': subject,
                    'inReplyTo': [msg_id] if msg_id else None,
                    'references': refs if refs else None,
                    'bodyValues': {'body': {'value': '''$BODY''', 'charset': 'utf-8'}},
                    'textBody': [{'partId': 'body', 'type': 'text/plain'}],
                    'keywords': {'\$draft': True}
                }
            }
        }, '0'],
        ['EmailSubmission/set', {
            'accountId': '$ACCOUNT_ID',
            'create': {'sendIt': {'emailId': '#draft1'}}
        }, '1']
    ]
}

result = subprocess.run(
    ['curl', '-s', '-X', 'POST', '$API_URL',
     '-H', 'Authorization: Bearer $API_TOKEN',
     '-H', 'Content-Type: application/json',
     '-d', json.dumps(body)],
    capture_output=True, text=True
)
r = json.loads(result.stdout)
for resp in r.get('methodResponses', []):
    if resp[0] == 'EmailSubmission/set':
        if resp[1].get('created'):
            print('Reply sent!')
        else:
            print('Reply failed:', json.dumps(resp[1].get('notCreated', {})))
"
    ;;

  move)
    # Move email to mailbox. $1=email-id, $2=mailbox-id
    EMAIL_ID="${1:?Usage: jmap.sh move <email-id> <mailbox-id>}"
    MAILBOX_ID="${2:?Usage: jmap.sh move <email-id> <mailbox-id>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Email/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "update": {
          "'"$EMAIL_ID"'": {
            "mailboxIds": {"'"$MAILBOX_ID"'": true}
          }
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('updated'):
    print('Email moved successfully')
else:
    print('Move failed:', json.dumps(resp.get('notUpdated', {})))
"
    ;;

  archive)
    # Archive an email (move to Archive). $1=email-id
    EMAIL_ID="${1:?Usage: jmap.sh archive <email-id>}"
    # First find Archive mailbox ID
    ARCHIVE_ID=$(jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Mailbox/get", {"accountId": "'"$ACCOUNT_ID"'"}, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
for mb in r['methodResponses'][0][1]['list']:
    if mb.get('role') == 'archive' or mb['name'] == 'Archive':
        print(mb['id'])
        break
")
    if [ -z "$ARCHIVE_ID" ]; then
      echo "Archive mailbox not found"
      exit 1
    fi
    bash "$0" move "$EMAIL_ID" "$ARCHIVE_ID"
    ;;

  batch-delete)
    # Move multiple emails to trash. Reads email IDs from stdin, one per line
    python3 -c "
import sys, json, requests

api_url = 'https://api.fastmail.com/jmap/api/'
api_token = 'fmu1-fb6140a5-ead6d7914d25b4016dfbf38af438adec-0-753ad070bb4c0d390695c55ed4e77cf0'
account_id = 'ufb6140a5'

# First get the trash mailbox ID
trash_payload = {
    'using': ['urn:ietf:params:jmap:core', 'urn:ietf:params:jmap:mail'],
    'methodCalls': [['Mailbox/get', {'accountId': account_id}, '0']]
}

response = requests.post(api_url, 
                       headers={
                           'Authorization': f'Bearer {api_token}',
                           'Content-Type': 'application/json'
                       },
                       json=trash_payload)

trash_id = None
if response.status_code == 200:
    result = response.json()
    for mb in result['methodResponses'][0][1]['list']:
        if mb.get('role') == 'trash' or mb['name'] == 'Trash':
            trash_id = mb['id']
            break

if not trash_id:
    print('Trash mailbox not found')
    sys.exit(1)

email_ids = [line.strip() for line in sys.stdin if line.strip()]
batch_size = 200
total_deleted = 0

for i in range(0, len(email_ids), batch_size):
    batch = email_ids[i:i + batch_size]
    
    # Move emails to trash by updating their mailboxIds
    updates = {}
    for email_id in batch:
        updates[email_id] = {'mailboxIds': {trash_id: True}}
    
    payload = {
        'using': ['urn:ietf:params:jmap:core', 'urn:ietf:params:jmap:mail'],
        'methodCalls': [
            ['Email/set', {
                'accountId': account_id,
                'update': updates
            }, '0']
        ]
    }
    
    response = requests.post(api_url, 
                           headers={
                               'Authorization': f'Bearer {api_token}',
                               'Content-Type': 'application/json'
                           },
                           json=payload)
    
    if response.status_code == 200:
        result = response.json()
        updated = result['methodResponses'][0][1].get('updated', {})
        deleted_count = len(updated)
        total_deleted += deleted_count
        print(f'Batch {i//batch_size + 1}: moved {deleted_count}/{len(batch)} emails to trash (total: {total_deleted})')
    else:
        print(f'Batch {i//batch_size + 1} failed: HTTP {response.status_code}')
        
print(f'Total emails moved to trash: {total_deleted}')
"
    ;;

  batch-destroy)
    # Permanently delete multiple emails. Reads email IDs from stdin, one per line
    python3 -c "
import sys, json, requests

api_url = 'https://api.fastmail.com/jmap/api/'
api_token = 'fmu1-fb6140a5-ead6d7914d25b4016dfbf38af438adec-0-753ad070bb4c0d390695c55ed4e77cf0'
account_id = 'ufb6140a5'

email_ids = [line.strip() for line in sys.stdin if line.strip()]
batch_size = 200
total_destroyed = 0

for i in range(0, len(email_ids), batch_size):
    batch = email_ids[i:i + batch_size]
    
    payload = {
        'using': ['urn:ietf:params:jmap:core', 'urn:ietf:params:jmap:mail'],
        'methodCalls': [
            ['Email/set', {
                'accountId': account_id,
                'destroy': batch
            }, '0']
        ]
    }
    
    response = requests.post(api_url, 
                           headers={
                               'Authorization': f'Bearer {api_token}',
                               'Content-Type': 'application/json'
                           },
                           json=payload)
    
    if response.status_code == 200:
        result = response.json()
        destroyed = result['methodResponses'][0][1].get('destroyed', [])
        destroyed_count = len(destroyed)
        total_destroyed += destroyed_count
        print(f'Batch {i//batch_size + 1}: permanently deleted {destroyed_count}/{len(batch)} emails (total: {total_destroyed})')
    else:
        print(f'Batch {i//batch_size + 1} failed: HTTP {response.status_code}')
        
print(f'Total emails permanently deleted: {total_destroyed}')
"
    ;;

  delete)
    # Move email to trash. $1=email-id
    EMAIL_ID="${1:?Usage: jmap.sh delete <email-id>}"
    TRASH_ID=$(jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Mailbox/get", {"accountId": "'"$ACCOUNT_ID"'"}, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
for mb in r['methodResponses'][0][1]['list']:
    if mb.get('role') == 'trash' or mb['name'] == 'Trash':
        print(mb['id'])
        break
")
    if [ -z "$TRASH_ID" ]; then
      echo "Trash mailbox not found"
      exit 1
    fi
    bash "$0" move "$EMAIL_ID" "$TRASH_ID"
    echo "Moved to trash"
    ;;

  create-mailbox)
    # Create a new mailbox. $1=name, $2=parent-id (optional)
    NAME="${1:?Usage: jmap.sh create-mailbox <name> [parent-id]}"
    PARENT="${2:-}"
    PARENT_JSON="null"
    [ -n "$PARENT" ] && PARENT_JSON="\"$PARENT\""
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Mailbox/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "create": {
          "mb1": {
            "name": "'"$NAME"'",
            "parentId": '"$PARENT_JSON"'
          }
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('created'):
    mb = resp['created']['mb1']
    print(f'Created mailbox: $NAME [id: {mb[\"id\"]}]')
else:
    print('Failed:', json.dumps(resp.get('notCreated', {})))
"
    ;;

  flag)
    # Flag/unflag an email. $1=email-id, $2=flag (seen/flagged/answered), $3=true/false
    EMAIL_ID="${1:?Usage: jmap.sh flag <email-id> <flag> <true|false>}"
    FLAG="${2:?Usage: jmap.sh flag <email-id> <flag> <true|false>}"
    VALUE="${3:-true}"
    KEYWORD="\$$FLAG"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
      "methodCalls": [["Email/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "update": {
          "'"$EMAIL_ID"'": {
            "keywords/'"$KEYWORD"'": '"$VALUE"'
          }
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('updated'):
    print('Flag updated')
else:
    print('Failed:', json.dumps(resp.get('notUpdated', {})))
"
    ;;

  ## ── MASKED EMAIL ───────────────────────────────

  masked-list)
    # List masked email addresses. $1=count (default 50)
    COUNT="${1:-50}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/get", {"accountId": "'"$ACCOUNT_ID"'"}, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
items = r['methodResponses'][0][1].get('list', [])
if not items:
    print('No masked emails')
else:
    for m in items[:$COUNT]:
        email = m.get('email', '')
        state = m.get('state', 'unknown')
        domain = m.get('forDomain', '')
        desc = m.get('description', '')
        created = m.get('createdAt', '')[:10]
        icon = '✅' if state == 'enabled' else '⏸️' if state == 'disabled' else '🗑️'
        extra = f' ({domain})' if domain else ''
        extra += f' - {desc}' if desc else ''
        print(f'{icon} {email} [{state}]{extra} ({created})')
"
    ;;

  masked-search)
    # Search masked emails. $1=query (searches email, domain, description)
    QUERY="${1:?Usage: jmap.sh masked-search <query>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/get", {"accountId": "'"$ACCOUNT_ID"'"}, "0"]]
    }' | python3 -c "
import sys,json
q = '$QUERY'.lower()
r=json.load(sys.stdin)
items = r['methodResponses'][0][1].get('list', [])
found = [m for m in items if q in m.get('email','').lower() or q in m.get('forDomain','').lower() or q in m.get('description','').lower()]
if not found:
    print('No masked emails matching: $QUERY')
else:
    for m in found:
        email = m.get('email', '')
        state = m.get('state', 'unknown')
        domain = m.get('forDomain', '')
        desc = m.get('description', '')
        icon = '✅' if state == 'enabled' else '⏸️' if state == 'disabled' else '🗑️'
        extra = f' ({domain})' if domain else ''
        extra += f' - {desc}' if desc else ''
        print(f'{icon} {email} [{state}]{extra} [id: {m[\"id\"]}]')
"
    ;;

  masked-create)
    # Create a masked email. $1=domain (optional), $2=description (optional)
    DOMAIN="${1:-}"
    DESC="${2:-}"
    DOMAIN_JSON="null"
    [ -n "$DOMAIN" ] && DOMAIN_JSON="\"$DOMAIN\""
    DESC_JSON="null"
    [ -n "$DESC" ] && DESC_JSON="\"$DESC\""
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "create": {
          "me1": {
            "state": "enabled",
            "forDomain": '"$DOMAIN_JSON"',
            "description": '"$DESC_JSON"'
          }
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('created'):
    me = resp['created']['me1']
    print(f'Created masked email: {me[\"email\"]}')
else:
    print('Failed:', json.dumps(resp.get('notCreated', {})))
"
    ;;

  masked-disable)
    # Disable a masked email. $1=masked email id
    ME_ID="${1:?Usage: jmap.sh masked-disable <masked-email-id>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "update": {
          "'"$ME_ID"'": {"state": "disabled"}
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('updated'): print('Masked email disabled')
else: print('Failed:', json.dumps(resp.get('notUpdated', {})))
"
    ;;

  masked-enable)
    # Enable a masked email. $1=masked email id
    ME_ID="${1:?Usage: jmap.sh masked-enable <masked-email-id>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "update": {
          "'"$ME_ID"'": {"state": "enabled"}
        }
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('updated'): print('Masked email enabled')
else: print('Failed:', json.dumps(resp.get('notUpdated', {})))
"
    ;;

  masked-delete)
    # Delete a masked email. $1=masked email id
    ME_ID="${1:?Usage: jmap.sh masked-delete <masked-email-id>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "https://www.fastmail.com/dev/maskedemail"],
      "methodCalls": [["MaskedEmail/set", {
        "accountId": "'"$ACCOUNT_ID"'",
        "destroy": ["'"$ME_ID"'"]
      }, "0"]]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
resp = r['methodResponses'][0][1]
if resp.get('destroyed'): print('Masked email deleted')
else: print('Failed:', json.dumps(resp.get('notDestroyed', {})))
"
    ;;

  ## ── FILES (WebDAV) ─────────────────────────────

  files-list)
    # List files. $1=path (default /)
    FPATH="${1:-/}"
    curl -s -u "guy.mackenzie@macfiver.com:4g22266w3z4g3245" \
      "https://myfiles.fastmail.com${FPATH}" \
      -X PROPFIND -H "Depth: 1" | python3 -c "
import sys, xml.etree.ElementTree as ET
data = sys.stdin.read()
root = ET.fromstring(data)
ns = {'d': 'DAV:'}
for resp in root.findall('.//d:response', ns):
    href = resp.find('d:href', ns)
    size = resp.find('.//d:getcontentlength', ns)
    name = resp.find('.//d:displayname', ns)
    restype = resp.find('.//d:collection', ns)
    if href is not None:
        h = href.text
        n = name.text if name is not None else h.split('/')[-1]
        s = size.text if size is not None else '-'
        t = '📁' if restype is not None else '📄'
        print(f'{t} {n} ({s} bytes)' if s != '-' and s != '0' else f'{t} {n}')
"
    ;;

  files-upload)
    # Upload a file. $1=local path, $2=remote path
    LOCAL="${1:?Usage: jmap.sh files-upload <local-path> <remote-path>}"
    REMOTE="${2:?Usage: jmap.sh files-upload <local-path> <remote-path>}"
    curl -s -u "guy.mackenzie@macfiver.com:4g22266w3z4g3245" \
      "https://myfiles.fastmail.com${REMOTE}" \
      -T "$LOCAL" && echo "Uploaded: $REMOTE"
    ;;

  files-download)
    # Download a file. $1=remote path, $2=local path
    REMOTE="${1:?Usage: jmap.sh files-download <remote-path> <local-path>}"
    LOCAL="${2:?Usage: jmap.sh files-download <remote-path> <local-path>}"
    curl -s -u "guy.mackenzie@macfiver.com:4g22266w3z4g3245" \
      "https://myfiles.fastmail.com${REMOTE}" \
      -o "$LOCAL" && echo "Downloaded to: $LOCAL"
    ;;

  files-mkdir)
    # Create a folder. $1=path
    FPATH="${1:?Usage: jmap.sh files-mkdir <path>}"
    curl -s -u "guy.mackenzie@macfiver.com:4g22266w3z4g3245" \
      "https://myfiles.fastmail.com${FPATH}" \
      -X MKCOL && echo "Created folder: $FPATH"
    ;;

  files-delete)
    # Delete a file or folder. $1=path
    FPATH="${1:?Usage: jmap.sh files-delete <path>}"
    curl -s -u "guy.mackenzie@macfiver.com:4g22266w3z4g3245" \
      "https://myfiles.fastmail.com${FPATH}" \
      -X DELETE && echo "Deleted: $FPATH"
    ;;

  ## ── CONTACTS ──────────────────────────────────

  contacts)
    # List contacts. $1=count (default 50)
    COUNT="${1:-50}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:contacts"],
      "methodCalls": [
        ["Contact/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "limit": '"$COUNT"'
        }, "0"],
        ["Contact/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Contact/query", "path": "/ids"}
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
contacts = r['methodResponses'][1][1].get('list', [])
if not contacts:
    print('No contacts found')
else:
    for c in contacts:
        name = ' '.join(filter(None, [c.get('firstName',''), c.get('lastName','')]))
        emails = [e.get('value','') for e in c.get('emails', [])]
        phones = [p.get('value','') for p in c.get('phones', [])]
        print(f'{name or \"(no name)\"}: {\" \".join(emails)} {\" \".join(phones)}')
"
    ;;

  contact-search)
    # Search contacts. $1=query
    QUERY="${1:?Usage: jmap.sh contact-search <query>}"
    jmap_call '{
      "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:contacts"],
      "methodCalls": [
        ["Contact/query", {
          "accountId": "'"$ACCOUNT_ID"'",
          "filter": {"text": "'"$QUERY"'"},
          "limit": 20
        }, "0"],
        ["Contact/get", {
          "accountId": "'"$ACCOUNT_ID"'",
          "#ids": {"resultOf": "0", "name": "Contact/query", "path": "/ids"}
        }, "1"]
      ]
    }' | python3 -c "
import sys,json
r=json.load(sys.stdin)
contacts = r['methodResponses'][1][1].get('list', [])
if not contacts:
    print('No contacts found matching: $QUERY')
else:
    for c in contacts:
        name = ' '.join(filter(None, [c.get('firstName',''), c.get('lastName','')]))
        emails = [e.get('value','') for e in c.get('emails', [])]
        phones = [p.get('value','') for p in c.get('phones', [])]
        company = c.get('company', '')
        extra = f' ({company})' if company else ''
        print(f'{name or \"(no name)\"}{extra}: {\" \".join(emails)} {\" \".join(phones)}')
"
    ;;

  ## ── CALENDAR (CalDAV) ──────────────────────────

  calendars)
    # List all calendars
    curl -s -L -u "guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b" \
      "https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com/" \
      -X PROPFIND -H "Depth: 1" -H "Content-Type: application/xml" \
      -d '<?xml version="1.0"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:cs="http://calendarserver.org/ns/" xmlns:x="http://apple.com/ns/ical/">
  <d:prop>
    <d:displayname/>
    <d:resourcetype/>
    <cs:getctag/>
    <x:calendar-color/>
  </d:prop>
</d:propfind>' | python3 -c "
import sys, xml.etree.ElementTree as ET
data = sys.stdin.read()
root = ET.fromstring(data)
ns = {'d': 'DAV:', 'cs': 'http://calendarserver.org/ns/', 'x': 'http://apple.com/ns/ical/'}
for resp in root.findall('.//d:response', ns):
    href = resp.find('d:href', ns)
    name = resp.find('.//d:displayname', ns)
    rt = resp.find('.//d:resourcetype', ns)
    is_cal = False
    if rt is not None:
        for c in rt:
            if 'calendar' in c.tag:
                is_cal = True
    if is_cal and href is not None and name is not None and name.text:
        h = href.text.rstrip('/')
        cal_id = h.split('/')[-1]
        print(f'📅 {name.text} [{cal_id}]')
"
    ;;

  events)
    # List upcoming events. $1=days ahead (default 14), $2=calendar path (optional, default all)
    DAYS="${1:-14}"
    START=$(date -u +"%Y%m%dT000000Z")
    END=$(date -u -d "+${DAYS} days" +"%Y%m%dT235959Z")
    CAL_PATH="${2:-}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    if [ -n "$CAL_PATH" ]; then
      URLS="${CALDAV_BASE}/${CAL_PATH}/"
    else
      # Get all calendar URLs first
      URLS=$(curl -s -L -u "$CALDAV_AUTH" \
        "${CALDAV_BASE}/" \
        -X PROPFIND -H "Depth: 1" -H "Content-Type: application/xml" \
        -d '<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:resourcetype/></d:prop></d:propfind>' | python3 -c "
import sys, xml.etree.ElementTree as ET
root = ET.fromstring(sys.stdin.read())
ns = {'d': 'DAV:'}
for resp in root.findall('.//d:response', ns):
    href = resp.find('d:href', ns)
    rt = resp.find('.//d:resourcetype', ns)
    is_cal = False
    if rt is not None:
        for child in rt:
            if 'calendar' in child.tag:
                is_cal = True
                break
    if href is not None and is_cal:
        print('https://caldav.fastmail.com' + href.text)
")
    fi

    for URL in $URLS; do
      curl -s -L -u "$CALDAV_AUTH" \
        "$URL" \
        -X REPORT -H "Depth: 1" -H "Content-Type: application/xml" \
        -d '<?xml version="1.0"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:time-range start="'"$START"'" end="'"$END"'"/>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>'
    done | python3 -c "
import sys, re
from datetime import datetime, timezone

data = sys.stdin.read()
events = []
for block in re.findall(r'BEGIN:VEVENT.*?END:VEVENT', data, re.DOTALL):
    summary = re.search(r'SUMMARY:(.*)', block)
    dtstart = re.search(r'DTSTART[^:]*:(.*)', block)
    dtend = re.search(r'DTEND[^:]*:(.*)', block)
    location = re.search(r'LOCATION:(.*)', block)
    uid = re.search(r'UID:(.*)', block)

    s = summary.group(1).strip() if summary else '(no title)'
    ds = dtstart.group(1).strip() if dtstart else ''
    de = dtend.group(1).strip() if dtend else ''
    loc = location.group(1).strip() if location else ''
    u = uid.group(1).strip() if uid else ''

    # Parse date
    for fmt in ('%Y%m%dT%H%M%SZ', '%Y%m%dT%H%M%S', '%Y%m%d'):
        try:
            dt = datetime.strptime(ds, fmt)
            break
        except:
            dt = None

    events.append((dt, s, ds, de, loc, u))

events.sort(key=lambda x: x[0] or datetime.min)
if not events:
    print('No events in the next $DAYS days')
else:
    for dt, s, ds, de, loc, u in events:
        date_str = dt.strftime('%a %b %d %H:%M') if dt else ds
        loc_str = f' 📍 {loc}' if loc else ''
        print(f'{date_str}  {s}{loc_str}')
"
    ;;

  event-search)
    # Search events by text. $1=query, $2=days to search back and forward (default 90)
    QUERY="${1:?Usage: jmap.sh event-search <query> [days]}"
    DAYS="${2:-90}"
    START=$(date -u -d "-${DAYS} days" +"%Y%m%dT000000Z")
    END=$(date -u -d "+${DAYS} days" +"%Y%m%dT235959Z")

    # Get all calendar URLs
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"
    URLS=$(curl -s -L -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/" \
      -X PROPFIND -H "Depth: 1" -H "Content-Type: application/xml" \
      -d '<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:resourcetype/></d:prop></d:propfind>' | python3 -c "
import sys, xml.etree.ElementTree as ET
root = ET.fromstring(sys.stdin.read())
ns = {'d': 'DAV:'}
for resp in root.findall('.//d:response', ns):
    href = resp.find('d:href', ns)
    rt = resp.find('.//d:resourcetype', ns)
    is_cal = False
    if rt is not None:
        for child in rt:
            if 'calendar' in child.tag:
                is_cal = True
                break
    if href is not None and is_cal:
        print('https://caldav.fastmail.com' + href.text)
")

    for URL in $URLS; do
      curl -s -L -u "$CALDAV_AUTH" \
        "$URL" \
        -X REPORT -H "Depth: 1" -H "Content-Type: application/xml" \
        -d '<?xml version="1.0"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><c:calendar-data/></d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:time-range start="'"$START"'" end="'"$END"'"/>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>'
    done | python3 -c "
import sys, re
from datetime import datetime

query = '${QUERY}'.lower()
data = sys.stdin.read()
events = []
for block in re.findall(r'BEGIN:VEVENT.*?END:VEVENT', data, re.DOTALL):
    if query not in block.lower():
        continue
    summary = re.search(r'SUMMARY:(.*)', block)
    dtstart = re.search(r'DTSTART[^:]*:(.*)', block)
    location = re.search(r'LOCATION:(.*)', block)

    s = summary.group(1).strip() if summary else '(no title)'
    ds = dtstart.group(1).strip() if dtstart else ''
    loc = location.group(1).strip() if location else ''

    for fmt in ('%Y%m%dT%H%M%SZ', '%Y%m%dT%H%M%S', '%Y%m%d'):
        try:
            dt = datetime.strptime(ds, fmt)
            break
        except:
            dt = None

    events.append((dt, s, ds, loc))

events.sort(key=lambda x: x[0] or datetime.min)
if not events:
    print('No events matching: $QUERY')
else:
    for dt, s, ds, loc in events:
        date_str = dt.strftime('%a %b %d %H:%M') if dt else ds
        loc_str = f' 📍 {loc}' if loc else ''
        print(f'{date_str}  {s}{loc_str}')
"
    ;;

  calendar-create)
    # Create a new calendar. $1=name, $2=color (optional, hex like #FF0000)
    CAL_NAME="${1:?Usage: jmap.sh calendar-create <name> [color]}"
    CAL_COLOR="${2:-}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"
    CAL_UUID="$(cat /proc/sys/kernel/random/uuid)"

    PROPS="<d:displayname>${CAL_NAME}</d:displayname>"
    [ -n "$CAL_COLOR" ] && PROPS="${PROPS}<x:calendar-color xmlns:x=\"http://apple.com/ns/ical/\">${CAL_COLOR}</x:calendar-color>"

    HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_UUID}/" \
      -X MKCALENDAR -H "Content-Type: application/xml" \
      -d "<?xml version=\"1.0\" encoding=\"utf-8\"?>
<c:mkcalendar xmlns:d=\"DAV:\" xmlns:c=\"urn:ietf:params:xml:ns:caldav\">
  <d:set>
    <d:prop>
      ${PROPS}
    </d:prop>
  </d:set>
</c:mkcalendar>")

    if [ "$HTTP_CODE" = "201" ]; then
      echo "✅ Created calendar: ${CAL_NAME} [${CAL_UUID}]"
    else
      echo "❌ Failed to create calendar (HTTP ${HTTP_CODE})"
      exit 1
    fi
    ;;

  calendar-delete)
    # Delete a calendar and ALL its events. $1=calendar-id
    CAL_ID="${1:?Usage: jmap.sh calendar-delete <calendar-id>}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_ID}/" \
      -X DELETE)

    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
      echo "✅ Deleted calendar: ${CAL_ID}"
    else
      echo "❌ Failed to delete calendar (HTTP ${HTTP_CODE})"
      exit 1
    fi
    ;;

  event-add)
    # Add an event. $1=calendar-id, $2=title, $3=start (YYYY-MM-DDTHH:MM), $4=end (YYYY-MM-DDTHH:MM), $5=location (optional), $6=description (optional)
    # For all-day events use YYYY-MM-DD format for start/end
    CAL_ID="${1:?Usage: jmap.sh event-add <calendar-id> <title> <start> <end> [location] [description]}"
    TITLE="${2:?Missing title}"
    DTSTART_RAW="${3:?Missing start datetime (YYYY-MM-DDTHH:MM or YYYY-MM-DD)}"
    DTEND_RAW="${4:?Missing end datetime (YYYY-MM-DDTHH:MM or YYYY-MM-DD)}"
    LOCATION="${5:-}"
    DESCRIPTION="${6:-}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    EVT_UUID="$(cat /proc/sys/kernel/random/uuid)"

    # Detect all-day vs timed
    if [[ "$DTSTART_RAW" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
      # All-day event
      DTSTART_FMT="DTSTART;VALUE=DATE:$(echo "$DTSTART_RAW" | tr -d '-')"
      DTEND_FMT="DTEND;VALUE=DATE:$(echo "$DTEND_RAW" | tr -d '-')"
    else
      # Timed event — convert to UTC-ish iCal format
      DTSTART_FMT="DTSTART;TZID=America/Chicago:$(echo "$DTSTART_RAW" | tr -d ':-' | sed 's/T/T/')00"
      DTEND_FMT="DTEND;TZID=America/Chicago:$(echo "$DTEND_RAW" | tr -d ':-' | sed 's/T/T/')00"
    fi

    VEVENT="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Dux//OpenClaw//EN
BEGIN:VEVENT
UID:${EVT_UUID}
DTSTAMP:$(date -u +"%Y%m%dT%H%M%SZ")
${DTSTART_FMT}
${DTEND_FMT}
SUMMARY:${TITLE}"

    [ -n "$LOCATION" ] && VEVENT="${VEVENT}
LOCATION:${LOCATION}"
    [ -n "$DESCRIPTION" ] && VEVENT="${VEVENT}
DESCRIPTION:${DESCRIPTION}"

    VEVENT="${VEVENT}
END:VEVENT
END:VCALENDAR"

    HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_ID}/${EVT_UUID}.ics" \
      -X PUT -H "Content-Type: text/calendar; charset=utf-8" \
      -d "$VEVENT")

    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
      echo "✅ Created event: ${TITLE} (UID: ${EVT_UUID})"
    else
      echo "❌ Failed to create event (HTTP ${HTTP_CODE})"
      exit 1
    fi
    ;;

  event-edit)
    # Edit an event. First fetches it, then re-PUTs with changes.
    # $1=calendar-id, $2=uid, $3=field (title|start|end|location|description), $4=new value
    CAL_ID="${1:?Usage: jmap.sh event-edit <calendar-id> <uid> <field> <value>}"
    EVT_UID="${2:?Missing event UID}"
    FIELD="${3:?Missing field (title|start|end|location|description)}"
    VALUE="${4:?Missing new value}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    # Fetch existing event
    EXISTING=$(curl -s -L -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_ID}/${EVT_UID}.ics")

    if echo "$EXISTING" | grep -q "BEGIN:VEVENT"; then
      UPDATED=$(echo "$EXISTING" | python3 -c "
import sys, re

data = sys.stdin.read()
field = '${FIELD}'
value = '''${VALUE}'''

if field == 'title':
    data = re.sub(r'SUMMARY:.*', 'SUMMARY:' + value, data)
elif field == 'location':
    if re.search(r'LOCATION:', data):
        data = re.sub(r'LOCATION:.*', 'LOCATION:' + value, data)
    else:
        data = data.replace('END:VEVENT', 'LOCATION:' + value + '\nEND:VEVENT')
elif field == 'description':
    if re.search(r'DESCRIPTION:', data):
        data = re.sub(r'DESCRIPTION:.*', 'DESCRIPTION:' + value, data)
    else:
        data = data.replace('END:VEVENT', 'DESCRIPTION:' + value + '\nEND:VEVENT')
elif field in ('start', 'end'):
    tag = 'DTSTART' if field == 'start' else 'DTEND'
    if re.match(r'^\d{4}-\d{2}-\d{2}$', value):
        new_line = tag + ';VALUE=DATE:' + value.replace('-', '')
    else:
        new_line = tag + ';TZID=America/Chicago:' + value.replace('-','').replace(':','') + '00'
    data = re.sub(tag + r'[^:]*:.*', new_line, data)

# Update DTSTAMP
import datetime
stamp = datetime.datetime.utcnow().strftime('%Y%m%dT%H%M%SZ')
data = re.sub(r'DTSTAMP:.*', 'DTSTAMP:' + stamp, data)

print(data, end='')
")

      HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "$CALDAV_AUTH" \
        "${CALDAV_BASE}/${CAL_ID}/${EVT_UID}.ics" \
        -X PUT -H "Content-Type: text/calendar; charset=utf-8" \
        -d "$UPDATED")

      if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
        echo "✅ Updated ${FIELD} to: ${VALUE}"
      else
        echo "❌ Failed to update event (HTTP ${HTTP_CODE})"
        exit 1
      fi
    else
      echo "❌ Event not found: ${EVT_UID}"
      exit 1
    fi
    ;;

  event-delete)
    # Delete an event. $1=calendar-id, $2=uid
    CAL_ID="${1:?Usage: jmap.sh event-delete <calendar-id> <uid>}"
    EVT_UID="${2:?Missing event UID}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_ID}/${EVT_UID}.ics" \
      -X DELETE)

    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
      echo "✅ Deleted event: ${EVT_UID}"
    else
      echo "❌ Failed to delete event (HTTP ${HTTP_CODE})"
      exit 1
    fi
    ;;

  event-get)
    # Get full event details by UID. $1=calendar-id, $2=uid
    CAL_ID="${1:?Usage: jmap.sh event-get <calendar-id> <uid>}"
    EVT_UID="${2:?Missing event UID}"
    CALDAV_BASE="https://caldav.fastmail.com/dav/calendars/user/guy_mackenzie@macfiver.com"
    CALDAV_AUTH="guy.mackenzie@macfiver.com:6u5v3e3z274r9x4b"

    curl -s -L -u "$CALDAV_AUTH" \
      "${CALDAV_BASE}/${CAL_ID}/${EVT_UID}.ics" | python3 -c "
import sys, re
data = sys.stdin.read()
if 'BEGIN:VEVENT' not in data:
    print('Event not found')
    sys.exit(1)
fields = {
    'SUMMARY': 'Title',
    'DTSTART': 'Start',
    'DTEND': 'End',
    'LOCATION': 'Location',
    'DESCRIPTION': 'Description',
    'UID': 'UID',
    'ORGANIZER': 'Organizer',
    'STATUS': 'Status',
}
for key, label in fields.items():
    m = re.search(key + r'[^:]*:(.*)', data)
    if m:
        print(f'{label}: {m.group(1).strip()}')
"
    ;;

  *)
    echo "Unknown action: $ACTION"
    echo ""
    echo "Mail:     mailboxes, inbox, smart-inbox, mailbox-emails, unread, read, search, send, reply, move, archive, delete, batch-delete, batch-destroy, create-mailbox, flag"
    echo "Masked:   masked-list, masked-search, masked-create, masked-enable, masked-disable, masked-delete"
    echo "Files:    files-list, files-upload, files-download, files-mkdir, files-delete"
    echo "Contacts: contacts, contact-search"
    echo "Calendar: calendars, events, event-search, event-add, event-get, event-edit, event-delete"
    exit 1
    ;;
esac
