---
name: knowledge-management
description: Organize and classify OpenClaw knowledge entries into local folders by content type (Research, Decision, Insight, Lesson, Pattern, Project, Reference, Tutorial).
homepage: https://github.com/ClaireAICodes/openclaw-skill-knowledge-management
metadata:
  {
    "openclaw": {
      "emoji": "📚",
      "bins": ["km"]
    }
  }
---

# Knowledge Management Skill (Local Storage)

Organize your OpenClaw memory files into a structured local knowledge base. Automatically parses `MEMORY.md` and daily memory files, classifies entries by content type, and stores each as a timestamped markdown file in the appropriate folder.

## Available Tools

### Core Commands
- `km sync [options]` - Sync memory entries to local files
- `km classify [options]` - Parse and classify without storing (JSON output)
- `km summarize [options]` - Generate index files for each content type
- `km cleanup [options]` - Remove orphaned files
- `km list_types` - List all available content types

## Setup

No API keys needed! Ensure your workspace has the required folders:

```bash
mkdir -p ~/.openclaw/workspace/{Research,Decision,Insight,Lesson,Pattern,Project,Reference,Tutorial}
```

The skill will create missing folders automatically.

## Usage Examples

### Sync last 7 days with cleanup
```bash
km sync --days_back 7 --cleanup
```

### Dry run (preview only)
```bash
km sync --days_back 14 --dry_run
```

### Classify entries and export JSON
```bash
km classify --days_back 3 > entries.json
```

### Generate index files
```bash
km summarize --output_dir ~/.openclaw/workspace
```

### Preview orphan cleanup
```bash
km cleanup --dry_run
```

## Storage Structure

```
~/.openclaw/workspace/
├── Research/
│   ├── 20260215T1448_Title_Here_HASH.md
│   └── ...
├── Decision/
├── Insight/
├── Lesson/
├── Pattern/
├── Project/
├── Reference/
├── Tutorial/
├── Research_Index.md
├── Decision_Index.md
└── ... (other index files)
```

### File Naming

Format: `YYYYMMDDTHHMM_Title_With_Underscores_8CHARHASH.md`

The 8-character content hash suffix prevents filename collisions when titles are identical but content differs.

### File Content (YAML Frontmatter)

```markdown
---
title: "Protocol Name"
content_type: "Research"
domain: "OpenClaw"
certainty: "Verified"
impact: "Medium"
confidence_score: 8
tags: ["AI", "Automation"]
source: "MEMORY.md"
source_file: "MEMORY.md"
date: "2026-02-11"
content_hash: "e4b30e75d0f5a662"
---

Entry body content starts here...
```

## How It Works

1. Parses `MEMORY.md` and recent daily `memory/*.md` files
2. Classifies each entry (content type, domain, certainty, impact, tags, confidence)
3. Computes content hash for deduplication
4. Checks sync state (`memory/local-sync-state.json`) to skip already synced entries
5. Writes to appropriate folder with timestamp + hash filename
6. Updates state mapping (hash → filepath)
7. Optional cleanup removes files not in state

## Classification Logic

- **Content Type:** Keyword matching (Research, Lesson, Decision, Pattern, Tutorial, Reference, Insight)
- **Domain:** Contextual inference (AI Models, OpenClaw, Cost, Trading, etc.)
- **Certainty:** Based on language (Verified, Likely, Speculative, Opinion)
- **Impact:** Importance indicators (High, Medium, Low, Negligible)
- **Tags:** Auto-extracted from predefined keyword map
- **Confidence Score:** 1–10 heuristic (source credibility, length, data mentions)

Customize by editing the `EntryClassifier` class in `index-local.js`.

## State Management

`memory/local-sync-state.json` maps content hashes to file paths:

```json
{
  "e4b30e75d0f5a662": "/path/to/Research/202602151440_Title_e4b30e75.md"
}
```

This enables idempotent syncs and fast duplicate detection.

**Do not edit manually** unless recovering from corruption.

## Cron Integration

Automate daily syncs:

```bash
openclaw cron add \
  --name "Daily Knowledge Sync" \
  --cron "0 5 * * *" \
  --tz "Asia/Singapore" \
  --session isolated \
  --message "km sync --days_back 7"
```

## Troubleshooting

**"km: command not found"**
- Run `npm link` in the skill directory, or add `~/workspace/bin` to PATH.

**No entries found**
- Ensure `MEMORY.md` uses `##` section headers and `###` entry titles within recognized sections.

**Files not created**
- Check write permissions; run with `--verbose`.

**Old entries not syncing**
- They may already be in state. Clear `memory/local-sync-state.json` to force re-sync (caution: may duplicate files).

**Duplicate files**
- Run `km cleanup` to remove orphans, then `km sync` to create missing files.

---

**Version:** 2.0.0
**Changed:** 2026-02-15 — Switched from Notion to local storage, added hash suffixes for uniqueness.
**Author:** Claire (OpenClaw Agent)
**License:** MIT
