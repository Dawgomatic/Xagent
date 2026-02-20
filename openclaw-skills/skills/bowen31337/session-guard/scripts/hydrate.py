#!/usr/bin/env python3
"""
session-guard: Session context hydration script.
Reads recent daily notes and tiered memory to produce a rehydration summary
after a session reset is detected.

Usage:
  python3 hydrate.py                          # auto-detect workspace
  python3 hydrate.py --workspace /path/clawd  # explicit workspace
  python3 hydrate.py --days 2                 # how many days of notes to load (default: 2)
  python3 hydrate.py --memory-limit 5         # tiered memory results (default: 3)
  python3 hydrate.py --quiet                  # suppress warnings, only output content

Outputs a human-readable hydration summary to stdout.
The agent should read this output and relay key context to the user.
"""
import os
import sys
import argparse
import subprocess
import json
from datetime import datetime, timedelta
from pathlib import Path

def parse_args():
    p = argparse.ArgumentParser()
    p.add_argument("--workspace", default=None, help="Path to agent workspace (default: auto-detect)")
    p.add_argument("--days", type=int, default=2, help="Days of daily notes to load (default: 2)")
    p.add_argument("--memory-limit", type=int, default=3, help="Tiered memory results to fetch (default: 3)")
    p.add_argument("--quiet", action="store_true", help="Suppress warnings")
    return p.parse_args()

def find_workspace(args):
    if args.workspace:
        return Path(args.workspace)
    # Common locations
    candidates = [
        Path.home() / "clawd",
        Path(os.environ.get("OPENCLAW_WORKSPACE", "")),
        Path.cwd(),
    ]
    for c in candidates:
        if c and c.exists() and (c / "SOUL.md").exists():
            return c
    return None

def load_daily_notes(workspace, days, quiet):
    notes = {}
    today = datetime.now()
    memory_dir = workspace / "memory"
    for i in range(days):
        date = today - timedelta(days=i)
        date_str = date.strftime("%Y-%m-%d")
        note_file = memory_dir / f"{date_str}.md"
        if note_file.exists():
            try:
                content = note_file.read_text(encoding="utf-8")
                notes[date_str] = content[:4000]  # cap per file
            except Exception as e:
                if not quiet:
                    print(f"⚠️  Could not read {note_file}: {e}", file=sys.stderr)
        elif not quiet and i == 0:
            print(f"ℹ️  No daily note for today ({date_str})", file=sys.stderr)
    return notes

def search_tiered_memory(workspace, limit, quiet):
    cli = workspace / "skills" / "tiered-memory" / "scripts" / "memory_cli.py"
    if not cli.exists():
        if not quiet:
            print("ℹ️  Tiered memory CLI not found — skipping", file=sys.stderr)
        return None
    try:
        result = subprocess.run(
            [sys.executable, str(cli), "retrieve",
             "--query", "recent projects decisions events status",
             "--limit", str(limit)],
            capture_output=True, text=True, timeout=30, cwd=str(workspace)
        )
        if result.returncode == 0:
            return result.stdout.strip()
        elif not quiet:
            print(f"⚠️  Tiered memory search failed: {result.stderr[:200]}", file=sys.stderr)
    except Exception as e:
        if not quiet:
            print(f"⚠️  Tiered memory error: {e}", file=sys.stderr)
    return None

def load_memory_md(workspace, quiet):
    mem_file = workspace / "MEMORY.md"
    if mem_file.exists():
        try:
            content = mem_file.read_text(encoding="utf-8")
            # Just the first 2000 chars — the header/summary section
            return content[:2000]
        except Exception as e:
            if not quiet:
                print(f"⚠️  Could not read MEMORY.md: {e}", file=sys.stderr)
    return None

def main():
    args = parse_args()
    workspace = find_workspace(args)

    parts = []
    parts.append("# 🔄 SESSION HYDRATION SUMMARY")
    parts.append(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M')}\n")

    if not workspace:
        parts.append("⚠️  Workspace not found — provide --workspace path")
        print("\n".join(parts))
        return

    parts.append(f"Workspace: {workspace}\n")

    # 1. Daily notes
    daily_notes = load_daily_notes(workspace, args.days, args.quiet)
    if daily_notes:
        parts.append("## 📝 Recent Daily Notes")
        for date_str, content in sorted(daily_notes.items(), reverse=True):
            parts.append(f"\n### {date_str}")
            # Trim to most recent/relevant lines
            lines = content.strip().split("\n")
            parts.append("\n".join(lines[:60]))  # first 60 lines
    else:
        parts.append("## 📝 Daily Notes\n_(none found)_")

    # 2. Tiered memory
    tiered = search_tiered_memory(workspace, args.memory_limit, args.quiet)
    if tiered:
        parts.append("\n## 🧠 Tiered Memory (recent context)")
        parts.append(tiered)

    # 3. MEMORY.md core context
    memory_md = load_memory_md(workspace, args.quiet)
    if memory_md:
        parts.append("\n## 📚 Long-term Memory (MEMORY.md excerpt)")
        parts.append(memory_md)

    parts.append("\n---")
    parts.append("End of hydration summary. Resume from this context.")

    print("\n".join(parts))

if __name__ == "__main__":
    main()
