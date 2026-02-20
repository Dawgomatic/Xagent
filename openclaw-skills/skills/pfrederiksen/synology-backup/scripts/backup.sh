#!/bin/bash
# Synology Backup — Incremental daily snapshot via SMB
set -euo pipefail

CONFIG="${SYNOLOGY_BACKUP_CONFIG:-$HOME/.openclaw/synology-backup.json}"

if [[ ! -f "$CONFIG" ]]; then
    echo "Error: Config not found at $CONFIG"
    echo "Create it from the skill docs or set SYNOLOGY_BACKUP_CONFIG"
    exit 1
fi

# Parse config
HOST=$(jq -r '.host' "$CONFIG")
SHARE=$(jq -r '.share' "$CONFIG")
MOUNT=$(jq -r '.mountPoint // "/mnt/synology"' "$CONFIG")
CREDS=$(jq -r '.credentialsFile' "$CONFIG" | sed "s|^~|$HOME|")
SMB_VER=$(jq -r '.smbVersion // "3.0"' "$CONFIG")
RETENTION=$(jq -r '.retention // 7' "$CONFIG")
INCLUDE_SUBAGENT=$(jq -r '.includeSubAgentWorkspaces // true' "$CONFIG")

TIMESTAMP=$(date +%Y-%m-%d)
BACKUP_DIR="$MOUNT/backups"
SNAP_DIR="$BACKUP_DIR/$TIMESTAMP"

# Ensure mount
if ! mountpoint -q "$MOUNT"; then
    mkdir -p "$MOUNT"
    mount -t cifs "//$HOST/$SHARE" "$MOUNT" \
        -o credentials="$CREDS",vers="$SMB_VER"
    echo "Mounted //$HOST/$SHARE → $MOUNT"
fi

mkdir -p "$SNAP_DIR"

# Backup configured paths
for path_raw in $(jq -r '.backupPaths[]' "$CONFIG"); do
    path=$(echo "$path_raw" | sed "s|^~|$HOME|")
    
    if [[ ! -e "$path" ]]; then
        echo "⚠️  Skipping (not found): $path"
        continue
    fi
    
    name=$(basename "$path")
    
    if [[ -d "$path" ]]; then
        rsync -a --delete "$path/" "$SNAP_DIR/$name/"
    else
        cp "$path" "$SNAP_DIR/$name"
    fi
    echo "✓ $name"
done

# Backup sub-agent workspaces
if [[ "$INCLUDE_SUBAGENT" == "true" ]]; then
    for ws in "$HOME"/.openclaw/workspace-*/; do
        [[ -d "$ws" ]] || continue
        name=$(basename "$ws")
        rsync -a --delete "$ws" "$SNAP_DIR/$name/"
        echo "✓ $name"
    done
fi

# Prune old snapshots
cd "$BACKUP_DIR"
ls -1d 20* 2>/dev/null | sort -r | tail -n +$((RETENTION + 1)) | xargs -r rm -rf

# Write manifest
cat > "$SNAP_DIR/manifest.json" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "snapshot": "$TIMESTAMP",
  "host": "$(hostname)",
  "retention": $RETENTION
}
EOF

TOTAL_SIZE=$(du -sh "$SNAP_DIR" 2>/dev/null | cut -f1)
SNAP_COUNT=$(ls -1d "$BACKUP_DIR"/20* 2>/dev/null | wc -l)
echo ""
echo "Backup complete: $SNAP_DIR ($TOTAL_SIZE)"
echo "Snapshots: $SNAP_COUNT (keeping last $RETENTION)"
