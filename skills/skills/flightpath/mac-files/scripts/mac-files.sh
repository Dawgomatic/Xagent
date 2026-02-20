#!/usr/bin/env bash
# Mac file operations over SSH — targets Mac via Tailscale
set -euo pipefail

ACTION="${1:?Usage: mac-files.sh <action> <target-mac> [args...]}"
TARGET="${2:-mini}"
shift 2 || true

case "$TARGET" in
  macbook) SSH_TARGET="guymackenzie@guy-mac-m1"; HOME_DIR="/Users/guymackenzie" ;;
  mini)    SSH_TARGET="guym@doclib"; HOME_DIR="/Users/guym" ;;
  *)       echo "Unknown target: $TARGET" >&2; exit 1 ;;
esac

SSH_OPTS="-o ConnectTimeout=10 -o IdentitiesOnly=yes -i $HOME/.ssh/id_ed25519"

ALLOWED_DIRS="$HOME_DIR/Documents $HOME_DIR/Desktop $HOME_DIR/Downloads $HOME_DIR/Projects"

# Validate path is within allowed directories
validate_path() {
  local path="$1"
  # Expand ~ to home dir
  path="${path/#\~/$HOME_DIR}"
  # Resolve to absolute if relative
  if [[ "$path" != /* ]]; then
    path="$HOME_DIR/$path"
  fi
  local allowed=false
  for dir in $ALLOWED_DIRS; do
    if [[ "$path" == "$dir"* ]]; then
      allowed=true
      break
    fi
  done
  if [ "$allowed" = false ]; then
    echo "ERROR: Path '$path' is outside allowed directories (Documents, Desktop, Downloads, Projects)" >&2
    exit 1
  fi
  echo "$path"
}

run_ssh() {
  ssh $SSH_OPTS "$SSH_TARGET" "$@"
}

case "$ACTION" in
  search)
    # Spotlight search. $1=query, $2=folder (optional, limits scope)
    QUERY="${1:?Usage: mac-files.sh search <target> <query> [folder]}"
    FOLDER="${2:-}"
    if [ -n "$FOLDER" ]; then
      FOLDER=$(validate_path "$FOLDER")
      run_ssh "mdfind -onlyin '$FOLDER' '$QUERY' 2>/dev/null | head -30"
    else
      # Search all allowed dirs
      run_ssh "mdfind -onlyin '$HOME_DIR/Documents' -onlyin '$HOME_DIR/Desktop' -onlyin '$HOME_DIR/Downloads' -onlyin '$HOME_DIR/Projects' '$QUERY' 2>/dev/null | head -30"
    fi
    ;;

  grep)
    # Search file contents. $1=pattern, $2=folder
    PATTERN="${1:?Usage: mac-files.sh grep <target> <pattern> <folder>}"
    FOLDER="${2:?Usage: mac-files.sh grep <target> <pattern> <folder>}"
    FOLDER=$(validate_path "$FOLDER")
    run_ssh "grep -rl '$PATTERN' '$FOLDER' 2>/dev/null | head -20"
    ;;

  ls)
    # List directory. $1=path
    DIR="${1:?Usage: mac-files.sh ls <target> <path>}"
    DIR=$(validate_path "$DIR")
    run_ssh "ls -lah '$DIR'"
    ;;

  tree)
    # Tree view. $1=path, $2=depth (default 2)
    DIR="${1:?Usage: mac-files.sh tree <target> <path> [depth]}"
    DEPTH="${2:-2}"
    DIR=$(validate_path "$DIR")
    run_ssh "find '$DIR' -maxdepth $DEPTH -print 2>/dev/null | head -50 | sed 's|$DIR/||' | sort"
    ;;

  read)
    # Read a file. $1=path
    FILE="${1:?Usage: mac-files.sh read <target> <path>}"
    FILE=$(validate_path "$FILE")
    run_ssh "cat '$FILE'"
    ;;

  head)
    # Read first N lines. $1=path, $2=lines (default 20)
    FILE="${1:?Usage: mac-files.sh head <target> <path> [lines]}"
    LINES="${2:-20}"
    FILE=$(validate_path "$FILE")
    run_ssh "head -n $LINES '$FILE'"
    ;;

  tail)
    # Read last N lines. $1=path, $2=lines (default 20)
    FILE="${1:?Usage: mac-files.sh tail <target> <path> [lines]}"
    LINES="${2:-20}"
    FILE=$(validate_path "$FILE")
    run_ssh "tail -n $LINES '$FILE'"
    ;;

  create)
    # Create a file. $1=path, $2=content
    FILE="${1:?Usage: mac-files.sh create <target> <path> <content>}"
    CONTENT="${2:-}"
    FILE=$(validate_path "$FILE")
    run_ssh "mkdir -p '$(dirname "$FILE")' && cat > '$FILE' << 'CONTENT_EOF'
$CONTENT
CONTENT_EOF"
    echo "Created: $FILE"
    ;;

  append)
    # Append to a file. $1=path, $2=text
    FILE="${1:?Usage: mac-files.sh append <target> <path> <text>}"
    TEXT="${2:?Usage: mac-files.sh append <target> <path> <text>}"
    FILE=$(validate_path "$FILE")
    run_ssh "cat >> '$FILE' << 'CONTENT_EOF'
$TEXT
CONTENT_EOF"
    echo "Appended to: $FILE"
    ;;

  write)
    # Overwrite a file. $1=path, $2=content
    FILE="${1:?Usage: mac-files.sh write <target> <path> <content>}"
    CONTENT="${2:?Usage: mac-files.sh write <target> <path> <content>}"
    FILE=$(validate_path "$FILE")
    run_ssh "cat > '$FILE' << 'CONTENT_EOF'
$CONTENT
CONTENT_EOF"
    echo "Written: $FILE"
    ;;

  mkdir)
    # Create directory. $1=path
    DIR="${1:?Usage: mac-files.sh mkdir <target> <path>}"
    DIR=$(validate_path "$DIR")
    run_ssh "mkdir -p '$DIR'"
    echo "Created directory: $DIR"
    ;;

  mv)
    # Move/rename. $1=source, $2=destination
    SRC="${1:?Usage: mac-files.sh mv <target> <source> <destination>}"
    DST="${2:?Usage: mac-files.sh mv <target> <source> <destination>}"
    SRC=$(validate_path "$SRC")
    DST=$(validate_path "$DST")
    run_ssh "mv '$SRC' '$DST'"
    echo "Moved: $SRC -> $DST"
    ;;

  cp)
    # Copy. $1=source, $2=destination
    SRC="${1:?Usage: mac-files.sh cp <target> <source> <destination>}"
    DST="${2:?Usage: mac-files.sh cp <target> <source> <destination>}"
    SRC=$(validate_path "$SRC")
    DST=$(validate_path "$DST")
    run_ssh "cp -R '$SRC' '$DST'"
    echo "Copied: $SRC -> $DST"
    ;;

  trash)
    # Move to trash. $1=path
    FILE="${1:?Usage: mac-files.sh trash <target> <path>}"
    FILE=$(validate_path "$FILE")
    run_ssh "mv '$FILE' '$HOME_DIR/.Trash/'"
    echo "Trashed: $FILE"
    ;;

  open)
    # Open file in default app. $1=path
    FILE="${1:?Usage: mac-files.sh open <target> <path>}"
    FILE=$(validate_path "$FILE")
    run_ssh "open '$FILE'"
    echo "Opened: $FILE"
    ;;

  info)
    # File info. $1=path
    FILE="${1:?Usage: mac-files.sh info <target> <path>}"
    FILE=$(validate_path "$FILE")
    run_ssh "stat -f 'Name: %N%nSize: %z bytes%nModified: %Sm%nType: %T' '$FILE' && file '$FILE'"
    ;;

  du)
    # Disk usage. $1=path
    DIR="${1:?Usage: mac-files.sh du <target> <path>}"
    DIR=$(validate_path "$DIR")
    run_ssh "du -sh '$DIR'/* 2>/dev/null | sort -rh | head -20"
    ;;

  *)
    echo "Unknown action: $ACTION"
    echo "Actions: search, grep, ls, tree, read, head, tail, create, append, write, mkdir, mv, cp, trash, open, info, du"
    exit 1
    ;;
esac
