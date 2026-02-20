#!/bin/bash
# Synology Backup — Restore from snapshot
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

BACKUP_DIR="$MOUNT/backups"

# Ensure mount
if ! mountpoint -q "$MOUNT"; then
    mkdir -p "$MOUNT"
    mount -t cifs "//$HOST/$SHARE" "$MOUNT" \
        -o credentials="$CREDS",vers="$SMB_VER"
fi

# No date given — list available snapshots
if [[ -z "${1:-}" ]]; then
    echo "Available snapshots:"
    echo ""
    for snap in "$BACKUP_DIR"/20*/; do
        [[ -d "$snap" ]] || continue
        date=$(basename "$snap")
        size=$(du -sh "$snap" 2>/dev/null | cut -f1)
        manifest="$snap/manifest.json"
        if [[ -f "$manifest" ]]; then
            ts=$(jq -r '.timestamp' "$manifest")
            echo "  $date  ($size)  backed up $ts"
        else
            echo "  $date  ($size)"
        fi
    done
    echo ""
    echo "Usage: $0 <date>  (e.g., $0 2026-02-20)"
    exit 0
fi

DATE="$1"
SNAP_DIR="$BACKUP_DIR/$DATE"

if [[ ! -d "$SNAP_DIR" ]]; then
    echo "Error: No snapshot found for $DATE"
    echo "Run without arguments to list available snapshots."
    exit 1
fi

echo "⚠️  This will overwrite current files with snapshot from $DATE"
echo "   Source: $SNAP_DIR"
echo ""
read -p "Continue? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

# Restore workspace
if [[ -d "$SNAP_DIR/workspace" ]]; then
    rsync -a --delete "$SNAP_DIR/workspace/" "$HOME/.openclaw/workspace/"
    echo "✓ workspace"
fi

# Restore sub-agent workspaces
for ws in "$SNAP_DIR"/workspace-*/; do
    [[ -d "$ws" ]] || continue
    name=$(basename "$ws")
    rsync -a --delete "$ws" "$HOME/.openclaw/$name/"
    echo "✓ $name"
done

# Restore config files
for file in openclaw.json .env; do
    if [[ -f "$SNAP_DIR/$file" ]]; then
        cp "$SNAP_DIR/$file" "$HOME/.openclaw/$file"
        echo "✓ $file"
    fi
done

# Restore directories
for dir in cron agents; do
    if [[ -d "$SNAP_DIR/$dir" ]]; then
        rsync -a --delete "$SNAP_DIR/$dir/" "$HOME/.openclaw/$dir/"
        echo "✓ $dir"
    fi
done

echo ""
echo "Restore complete from $DATE. Restart OpenClaw to apply config changes."
