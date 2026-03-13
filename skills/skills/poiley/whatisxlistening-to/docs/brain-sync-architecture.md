# Brain Sync Architecture

> Clawdbot's knowledge synchronization system connecting workspace memory, Obsidian, and mobile access.

## Overview

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  Clawdbot Workspace │     │   Obsidian Vault    │     │    CouchDB Server   │
│  ~/clawd/           │ ←→  │   (SMB on UNAS)     │ ←→  │  couchdb.rinzler    │
│                     │     │                     │     │      .cloud         │
│  • memory/*.md      │     │  ~/mnt/services/    │     │                     │
│  • .learnings/*.md  │     │  clawdbot/brain/    │     │  • obsidian db      │
│  • MEMORY.md        │     │                     │     │  • 365+ docs        │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
         │                           │                           │
         │    brain-sync.sh          │      LiveSync plugin      │
         │    (every 4 hours)        │      (real-time)          │
         │                           │                           │
         └───────────────────────────┴───────────────────────────┘
                                     │
                                     ▼
                          ┌─────────────────────┐
                          │   iOS Obsidian      │
                          │   (Ben's iPhone)    │
                          │                     │
                          │   LiveSync plugin   │
                          └─────────────────────┘
```

## Components

### 1. CouchDB Server (Kubernetes)

**Location:** couchdb.rinzler.cloud (K3s cluster on Rinzler)

**Credentials:**
- Admin User: `admin`
- Admin Password: `97gYy7MzaxFMvI0IuyOJ5khu`
- Database: `obsidian`

**Kubernetes Resources:**
- Namespace: `couchdb`
- Service: ClusterIP + Ingress via Traefik
- Storage: PVC on local-path provisioner
- Ingress: TLS via cert-manager (Let's Encrypt)

**Health Check:**
```bash
curl -s -u admin:97gYy7MzaxFMvI0IuyOJ5khu \
  "https://couchdb.rinzler.cloud/obsidian" | jq '.doc_count'
```

### 2. Obsidian Vault (UNAS SMB Mount)

**Mount Point:** `~/mnt/services/clawdbot/brain`
**NAS Location:** `smb://castor.rinzler.cloud/services/clawdbot/brain`
**Auto-mount:** Configured in macOS login items

**Structure:**
```
brain/
├── .obsidian/
│   └── plugins/
│       └── obsidian-livesync/
│           └── data.json          # LiveSync config
├── Config/
├── Daily/                         # Daily notes, memory sync target
├── Knowledge/
│   ├── Concepts/
│   ├── Learnings/                 # .learnings sync target
│   ├── Patterns/
│   └── Toolbox/
├── Meta/
├── People/
├── Projects/
├── Index.md
└── README.md
```

### 3. LiveSync Plugin Configuration

**Mac (~/mnt/services/clawdbot/brain/.obsidian/plugins/obsidian-livesync/data.json):**
```json
{
  "remoteType": "couchdb",
  "couchDB_URI": "https://couchdb.rinzler.cloud",
  "couchDB_USER": "admin",
  "couchDB_PASSWORD": "97gYy7MzaxFMvI0IuyOJ5khu",
  "couchDB_DBNAME": "obsidian",
  "liveSync": true,
  "syncOnSave": true,
  "syncOnStart": true,
  "syncOnFileOpen": true,
  "deviceAndVaultName": "mac-mini"
}
```

**iOS Configuration:**
```
URI: https://couchdb.rinzler.cloud
Username: admin
Password: 97gYy7MzaxFMvI0IuyOJ5khu
Database name: obsidian
Preset: LiveSync
Device name: iphone
```

### 4. Brain Sync Script

**Location:** `~/clawd/scripts/brain-sync.sh`

**Sync Flows:**
| Source | Destination | Direction |
|--------|-------------|-----------|
| `memory/*.md` | Obsidian `Daily/` | Workspace → Obsidian |
| `.learnings/*.md` | Obsidian `Knowledge/Learnings/` | Workspace → Obsidian |
| Ensue entries | Obsidian `Knowledge/` | Bidirectional |
| Obsidian (ensue_sync: true) | Ensue | Obsidian → Ensue |

**Manual Sync:**
```bash
~/clawd/scripts/brain-sync.sh all              # Full sync
~/clawd/scripts/brain-sync.sh memory           # Just memory files
~/clawd/scripts/brain-sync.sh ensue-to-obsidian  # Pull from Ensue
```

**Cron Job:**
- Schedule: Every 4 hours (`0 */4 * * *`)
- Job ID: `f088acd8-9180-400c-b2bb-472f9e19adfc`
- Managed by: Clawdbot Gateway cron

### 5. Ensue Integration

**Account:** clawdbot org on ensue.network
- Email: poileclawdbot@icloud.com
- API Key: Stored in Keychain as "ensue-api-key"

**Obsidian → Ensue Push:**
Add frontmatter to any note:
```yaml
---
ensue_sync: true
ensue_key: public/concepts/topic/name
description: Short description
---
```

## Troubleshooting

### LiveSync Not Syncing

1. **Check CouchDB connectivity:**
   ```bash
   curl -s -u admin:97gYy7MzaxFMvI0IuyOJ5khu \
     "https://couchdb.rinzler.cloud/_up"
   ```

2. **Check doc count:**
   ```bash
   curl -s -u admin:97gYy7MzaxFMvI0IuyOJ5khu \
     "https://couchdb.rinzler.cloud/obsidian" | jq '.doc_count'
   ```

3. **Verify LiveSync config:**
   ```bash
   cat ~/mnt/services/clawdbot/brain/.obsidian/plugins/obsidian-livesync/data.json | \
     jq '{couchDB_URI, liveSync, syncOnSave}'
   ```

4. **Restart Obsidian** to reload config

### Files Not Appearing

- **Case sensitivity:** CouchDB stores paths lowercase (`daily/` not `Daily/`)
- **Pull vs Sync:** Use "Fetch from remote" for initial sync on new devices
- **External files:** Files created outside Obsidian may need vault reload (Cmd+Opt+R)

### SMB Mount Issues

```bash
# Remount manually
umount ~/mnt/services 2>/dev/null
mount -t smbfs //poile@castor.rinzler.cloud/services ~/mnt/services
```

## Security Notes

- CouchDB credentials are stored in Obsidian config files (not ideal but required by LiveSync)
- HTTPS/TLS enforced via Traefik ingress
- Database is private (no public read access)
- Consider adding end-to-end encryption in LiveSync if storing sensitive data

## Setup Date

- **Initial Setup:** 2026-01-25
- **CouchDB Deployed:** K3s cluster on Rinzler
- **LiveSync Configured:** Mac Mini + iOS
- **Documented by:** Clawdbot 
