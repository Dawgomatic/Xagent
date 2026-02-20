#!/bin/bash
# Synology Backup — Status check
set -euo pipefail

CONFIG="${SYNOLOGY_BACKUP_CONFIG:-$HOME/.openclaw/synology-backup.json}"

if [[ ! -f "$CONFIG" ]]; then
    echo "Error: Config not found at $CONFIG"
    exit 1
fi

HOST=$(jq -r '.host' "$CONFIG")
SHARE=$(jq -r '.share' "$CONFIG")
MOUNT=$(jq -r '.mountPoint // "/mnt/synology"' "$CONFIG")
CREDS=$(jq -r '.credentialsFile' "$CONFIG" | sed "s|^~|$HOME|")
SMB_VER=$(jq -r '.smbVersion // "3.0"' "$CONFIG")
RETENTION=$(jq -r '.retention // 7' "$CONFIG")

BACKUP_DIR="$MOUNT/backups"

echo "=== Synology Backup Status ==="
echo ""

# Check mount
if mountpoint -q "$MOUNT" 2>/dev/null; then
    echo "Mount:     ✅ $MOUNT → //$HOST/$SHARE"
else
    echo "Mount:     ❌ Not mounted"
    echo "           Attempting mount..."
    mkdir -p "$MOUNT"
    if mount -t cifs "//$HOST/$SHARE" "$MOUNT" -o credentials="$CREDS",vers="$SMB_VER" 2>/dev/null; then
        echo "           ✅ Mounted successfully"
    else
        echo "           ❌ Mount failed — check host, share, and credentials"
        exit 1
    fi
fi

# Check disk space
DISK_INFO=$(df -h "$MOUNT" | tail -1)
DISK_SIZE=$(echo "$DISK_INFO" | awk '{print $2}')
DISK_USED=$(echo "$DISK_INFO" | awk '{print $3}')
DISK_AVAIL=$(echo "$DISK_INFO" | awk '{print $4}')
DISK_PCT=$(echo "$DISK_INFO" | awk '{print $5}')
echo "Disk:      $DISK_AVAIL available of $DISK_SIZE ($DISK_PCT used)"

# Snapshot info
if [[ -d "$BACKUP_DIR" ]]; then
    SNAP_COUNT=$(ls -1d "$BACKUP_DIR"/20* 2>/dev/null | wc -l)
    TOTAL_SIZE=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
    
    echo "Snapshots: $SNAP_COUNT (retention: $RETENTION days)"
    echo "Total:     $TOTAL_SIZE"
    echo ""
    
    # Latest snapshot
    LATEST=$(ls -1d "$BACKUP_DIR"/20* 2>/dev/null | sort -r | head -1)
    if [[ -n "$LATEST" ]]; then
        LATEST_DATE=$(basename "$LATEST")
        LATEST_SIZE=$(du -sh "$LATEST" 2>/dev/null | cut -f1)
        
        if [[ -f "$LATEST/manifest.json" ]]; then
            LATEST_TS=$(jq -r '.timestamp' "$LATEST/manifest.json")
            echo "Latest:    $LATEST_DATE ($LATEST_SIZE) at $LATEST_TS"
        else
            echo "Latest:    $LATEST_DATE ($LATEST_SIZE)"
        fi
        
        # Age check
        LATEST_EPOCH=$(date -d "$LATEST_DATE" +%s 2>/dev/null || echo 0)
        NOW_EPOCH=$(date +%s)
        AGE_HOURS=$(( (NOW_EPOCH - LATEST_EPOCH) / 3600 ))
        
        if [[ $AGE_HOURS -gt 48 ]]; then
            echo "⚠️  WARNING: Last backup is ${AGE_HOURS}h old!"
        fi
    fi
    
    echo ""
    echo "All snapshots:"
    for snap in "$BACKUP_DIR"/20*/; do
        [[ -d "$snap" ]] || continue
        date=$(basename "$snap")
        size=$(du -sh "$snap" 2>/dev/null | cut -f1)
        echo "  $date  $size"
    done
else
    echo "Snapshots: None (no backups directory)"
fi
