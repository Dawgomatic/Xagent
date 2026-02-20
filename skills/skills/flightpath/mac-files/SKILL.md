---
name: mac-files
description: Search, read, create, edit, and manage files on Guy's Macs (Mac Mini or MacBook) over SSH. Use when the user wants to find files, read documents, create or edit files, organize folders, or manage files on their Mac. Includes Spotlight search via mdfind.
---

# Mac Files (File Operations via SSH)

Manage files on remote Macs over SSH. Access is restricted to allowed directories for safety.

## Targets

- `mini` (default) — guym@doclib
- `macbook` — guymackenzie@guy-mac-m1

## Allowed Directories

- `~/Documents`
- `~/Desktop`
- `~/Downloads`
- `~/Projects`

All paths must be within these directories. Paths outside them are rejected.

## Script

`scripts/mac-files.sh <action> <target-mac> [args...]`

### Actions

**Finding files:**
- `search <query> [folder]` — Spotlight search (mdfind) across allowed dirs, or within a specific folder
- `grep <pattern> <folder>` — Search file contents for text

**Reading files:**
- `ls <path>` — List directory contents
- `tree <path> [depth]` — Tree view (default depth 2)
- `read <path>` — Read entire file
- `head <path> [lines]` — First N lines (default 20)
- `tail <path> [lines]` — Last N lines (default 20)
- `info <path>` — File size, type, modified date
- `du <path>` — Disk usage of directory contents

**Creating & editing:**
- `create <path> <content>` — Create a new file (creates parent dirs)
- `write <path> <content>` — Overwrite a file
- `append <path> <text>` — Append text to a file
- `mkdir <path>` — Create a directory

**Moving & organizing:**
- `mv <source> <dest>` — Move or rename (both paths must be in allowed dirs)
- `cp <source> <dest>` — Copy (both paths must be in allowed dirs)
- `trash <path>` — Move to Trash (safer than delete)
- `open <path>` — Open file in its default macOS app

### Examples

```bash
# Spotlight search for tax docs
bash scripts/mac-files.sh search mini "tax 2025"

# List Documents folder
bash scripts/mac-files.sh ls mini ~/Documents

# Read a file
bash scripts/mac-files.sh read mini ~/Documents/notes.txt

# Create a file
bash scripts/mac-files.sh create mini ~/Projects/ideas.md "# Ideas"

# Search file contents
bash scripts/mac-files.sh grep mini "password" ~/Documents
```

## Notes

- Always use `trash` instead of deleting — confirm with user first
- `search` uses macOS Spotlight (mdfind) which is very fast and searches file names and content
- `open` requires the Mac to be unlocked to show the app window
- Paths can use `~` which expands to the target user's home directory
