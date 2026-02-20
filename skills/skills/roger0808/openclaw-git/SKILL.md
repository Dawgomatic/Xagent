---
name: openclaw-git
description: Git automation for OpenClaw workspace. Use when user needs to commit and push changes from /home/roger/.openclaw to the remote repository. Triggers on: git commit, git push, save changes, backup openclaw.
---

# OpenClaw Git

Automated git commit and push for the OpenClaw workspace.

## Quick Start

### Commit and Push Changes

```bash
# Default commit message
~/.openclaw/workspace/skills/openclaw-git/scripts/git-push.sh

# Custom commit message
~/.openclaw/workspace/skills/openclaw-git/scripts/git-push.sh "Your commit message"
```

## What It Does

1. Changes to `/home/roger/.openclaw`
2. Stages all changes (`git add .`)
3. Commits with message (default: "update: $(date)")
4. Pushes to origin main

## Prerequisites

- Git repository initialized at `/home/roger/.openclaw`
- Remote origin configured
- Credentials configured (or use SSH key)

## Credentials

GitHub:
- Username: Roger0808
- Token: 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

## Script Reference

See [scripts/git-push.sh](scripts/git-push.sh) for the automation script.
