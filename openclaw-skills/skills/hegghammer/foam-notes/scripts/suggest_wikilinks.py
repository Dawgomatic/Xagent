#!/usr/bin/env python3
"""
Suggest wikilinks for a note based on existing notes in the archive.

Analyzes note content and identifies words/phrases that match existing note titles,
suggesting them as potential wikilinks.

Usage:
    python3 suggest_wikilinks.py <note-file>
    python3 suggest_wikilinks.py my-note.md
    python3 suggest_wikilinks.py my-note.md --min-length 4

Examples:
    python3 suggest_wikilinks.py meeting-notes.md
    python3 suggest_wikilinks.py research/paper.md --foam-root ~/notes
    python3 suggest_wikilinks.py idea.md --auto-apply  # Apply all non-conflicting
"""

import argparse
import re
import sys
from pathlib import Path
from difflib import SequenceMatcher

from foam_config import load_config, get_foam_root


def similarity(a: str, b: str) -> float:
    """Calculate string similarity (0-1)."""
    return SequenceMatcher(None, a.lower(), b.lower()).ratio()


def get_all_note_titles(foam_root: Path) -> dict:
    """Get all note titles and their file paths."""
    titles = {}

    for md_file in foam_root.rglob("*.md"):
        # Skip hidden directories and journals
        rel_path = md_file.relative_to(foam_root)
        if any(part.startswith(".") for part in rel_path.parts):
            continue

        try:
            content = md_file.read_text()
            # Get title from first h1 or filename
            title = None
            for line in content.split("\n")[:10]:
                if line.startswith("# "):
                    title = line[2:].strip()
                    break

            if not title:
                title = md_file.stem.replace("-", " ").replace("_", " ")

            # Store both filename and title as keys
            titles[md_file.stem.lower()] = {
                "path": str(rel_path),
                "title": title,
                "stem": md_file.stem,
            }
            titles[title.lower()] = {
                "path": str(rel_path),
                "title": title,
                "stem": md_file.stem,
            }
        except Exception:
            continue

    return titles


def find_wikilink_candidates(content: str, titles: dict, min_length: int = 3) -> list:
    """Find potential wikilink candidates in content."""
    candidates = []
    lines = content.split("\n")

    # Build regex patterns for multi-word titles
    title_patterns = []
    for key, info in titles.items():
        # Skip very short keys
        if len(key) < min_length:
            continue
        # Escape special regex chars but keep spaces
        pattern = re.escape(key)
        title_patterns.append((pattern, info))

    # Sort by length (longest first) to prefer multi-word matches
    title_patterns.sort(key=lambda x: len(x[0]), reverse=True)

    # Track what we've already matched to avoid duplicates
    matched_positions = set()
    in_frontmatter = False
    frontmatter_start = -1

    for line_idx, line in enumerate(lines):
        # Track YAML frontmatter boundaries
        if line.strip() == "---":
            if not in_frontmatter:
                in_frontmatter = True
                frontmatter_start = line_idx
                continue
            else:
                # End of frontmatter
                in_frontmatter = False
                continue

        # Skip if we're inside frontmatter
        if in_frontmatter:
            continue

        # Skip code blocks
        if line.startswith("```"):
            continue

        # Skip Markdown headings
        if line.lstrip().startswith("#"):
            continue

        for pattern, info in title_patterns:
            # Look for whole word matches
            # Use word boundaries for single words, allow flexible matching for multi-word
            if " " in pattern:
                regex = re.compile(pattern, re.IGNORECASE)
            else:
                regex = re.compile(r"\b" + pattern + r"\b", re.IGNORECASE)

            for match in regex.finditer(line):
                # Skip if already matched at this position
                pos = (line_idx, match.start())
                if pos in matched_positions:
                    continue

                # Skip if already a wikilink
                start = match.start()
                end = match.end()
                # Check for [[ before or ]] after
                before = line[max(0, start - 2) : start]
                after = line[end : min(len(line), end + 2)]
                if before == "[[" or after == "]]":
                    continue

                # Check for markdown link
                if "[" in line[max(0, start - 10) : start]:
                    # Simple heuristic: if there's a [ within 10 chars before, might be a link
                    if "]" in line[end : min(len(line), end + 10)]:
                        continue

                matched_positions.add(pos)

                candidates.append(
                    {
                        "line": line_idx + 1,
                        "column": start + 1,
                        "text": match.group(),
                        "target": info["stem"],
                        "target_title": info["title"],
                        "context": line.strip(),
                    }
                )

    # Deduplicate by text + line
    seen = set()
    unique = []
    for c in candidates:
        key = (c["text"].lower(), c["line"])
        if key not in seen:
            seen.add(key)
            unique.append(c)

    # Sort by line number
    unique.sort(key=lambda x: (x["line"], x["column"]))

    return unique


def format_suggestions(candidates: list) -> str:
    """Format candidates as numbered list."""
    if not candidates:
        return "No wikilink suggestions found."

    lines = [f"Found {len(candidates)} potential wikilink(s):\n"]

    for i, c in enumerate(candidates, 1):
        lines.append(f"{i}. Line {c['line']}, col {c['column']}")
        lines.append(f'   Text: "{c["text"]}"')
        lines.append(f"   Link to: [[{c['target']}]]")
        lines.append(f"   Context: ...{c['context'][:60]}...")
        lines.append("")

    lines.append("Respond with numbers to implement (e.g., '1 3 5'), 'all', or 'none'")
    return "\n".join(lines)


def apply_wikilinks(
    note_path: Path, candidates: list, selections: list, with_aliases: bool = False
) -> bool:
    """Apply selected wikilinks to the note."""
    if not selections:
        return False

    content = note_path.read_text()
    lines = content.split("\n")

    # Sort selections in reverse order (by line, then column) to apply from end to start
    # This preserves line/column positions as we modify
    to_apply = sorted(
        [(i, candidates[i - 1]) for i in selections if 1 <= i <= len(candidates)],
        key=lambda x: (-x[1]["line"], -x[1]["column"]),
    )

    for _, candidate in to_apply:
        line_idx = candidate["line"] - 1
        col_idx = candidate["column"] - 1
        text = candidate["text"]
        target = candidate["target"]

        line = lines[line_idx]
        # Replace the text with wikilink
        # Find the exact match at that position
        before = line[:col_idx]
        after = line[col_idx + len(text) :]

        # Create wikilink with or without alias
        if with_aliases:
            lines[line_idx] = before + f"[[{target}|{text}]]" + after
        else:
            lines[line_idx] = before + f"[[{target}]]" + after

    # Write back
    note_path.write_text("\n".join(lines))
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Suggest wikilinks for a note based on existing notes"
    )
    parser.add_argument(
        "note", help="Path to note file (relative to foam_root or absolute)"
    )
    parser.add_argument(
        "--foam-root", help="Foam workspace root directory (overrides config)"
    )
    parser.add_argument(
        "--min-length",
        "-m",
        type=int,
        default=3,
        help="Minimum word length to consider (default: 3)",
    )
    parser.add_argument(
        "--apply", "-a", help='Comma-separated list of numbers to apply (e.g., "1,3,5")'
    )
    parser.add_argument(
        "--auto-apply",
        action="store_true",
        help="Apply all non-conflicting suggestions without prompting",
    )
    parser.add_argument(
        "--dry-run", "-n", action="store_true", help="Show suggestions without applying"
    )
    parser.add_argument(
        "--with-aliases",
        action="store_true",
        help="Create wikilinks with aliases (e.g., [[target|text]]) instead of plain links",
    )

    args = parser.parse_args()

    # Get foam root
    config = load_config()
    if args.foam_root:
        foam_root = Path(args.foam_root).expanduser().resolve()
    else:
        foam_root = get_foam_root(config=config)

    if foam_root is None:
        print("Error: Not in a Foam workspace.", file=sys.stderr)
        print("Set foam_root in config.json or use --foam-root", file=sys.stderr)
        sys.exit(1)

    # Resolve note path
    note_path = Path(args.note)
    if not note_path.is_absolute():
        note_path = foam_root / note_path

    if not note_path.exists():
        print(f"Error: Note not found: {note_path}", file=sys.stderr)
        sys.exit(1)

    # Get all note titles
    print("Scanning archive for note titles...")
    titles = get_all_note_titles(foam_root)
    print(f"Found {len(titles) // 2} unique notes.")

    # Read note content
    content = note_path.read_text()

    # Find candidates
    candidates = find_wikilink_candidates(content, titles, args.min_length)

    if not candidates:
        print("\nNo wikilink suggestions found.")
        print("The note doesn't contain text matching other note titles.")
        return

    # Show suggestions
    print(format_suggestions(candidates))

    # Dry run - stop here
    if args.dry_run:
        return

    # Auto-apply
    if args.auto_apply:
        selections = list(range(1, len(candidates) + 1))
        apply_wikilinks(note_path, candidates, selections, args.with_aliases)
        print(
            f"\nApplied {len(selections)} wikilink(s) to {note_path.relative_to(foam_root)}"
        )
        return

    # Apply specific selections
    if args.apply:
        try:
            selections = [int(x.strip()) for x in args.apply.split(",")]
            apply_wikilinks(note_path, candidates, selections, args.with_aliases)
            print(
                f"\nApplied {len(selections)} wikilink(s) to {note_path.relative_to(foam_root)}"
            )
        except ValueError:
            print("Error: --apply should be comma-separated numbers", file=sys.stderr)
            sys.exit(1)
        return

    # Interactive mode - prompt for input
    print("\nEnter selections (or 'all' for all, 'none' to cancel):")
    try:
        response = input("> ").strip()

        if response.lower() in ("none", "n", ""):
            print("No changes made.")
            return

        if response.lower() == "all":
            selections = list(range(1, len(candidates) + 1))
        else:
            selections = [int(x.strip()) for x in response.split()]

        apply_wikilinks(note_path, candidates, selections, args.with_aliases)
        print(
            f"\nApplied {len(selections)} wikilink(s) to {note_path.relative_to(foam_root)}"
        )

    except (ValueError, KeyboardInterrupt):
        print("\nNo changes made.")
        return


if __name__ == "__main__":
    main()
