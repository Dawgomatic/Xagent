#!/usr/bin/env python3
"""
Validate and package a Xagent skill into a distributable .skill file.

Usage:
    python3 package_skill.py <path/to/skill-folder> [output-directory]
"""

import json
import os
import re
import sys
import zipfile
from pathlib import Path


SKIP_PATTERNS = {
    "node_modules", "__pycache__", ".git", ".DS_Store",
    "*.pyc", "*.pyo", "*.swp", "*.swo",
}

MAX_BUNDLE_SIZE = 50 * 1024 * 1024  # 50MB


class ValidationError:
    def __init__(self, level: str, message: str):
        self.level = level  # "error" or "warning"
        self.message = message

    def __str__(self):
        icon = "ERROR" if self.level == "error" else "WARN "
        return f"  [{icon}] {self.message}"


def parse_frontmatter(content: str) -> dict | None:
    """Extract YAML frontmatter from SKILL.md."""
    match = re.match(r"^---\n(.*?)\n---", content, re.DOTALL)
    if not match:
        return None

    fm = {}
    for line in match.group(1).split("\n"):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        colon = line.find(":")
        if colon > 0:
            key = line[:colon].strip()
            value = line[colon + 1:].strip().strip('"').strip("'")
            fm[key] = value
    return fm


def validate_skill(skill_dir: Path) -> list[ValidationError]:
    """Validate a skill directory. Returns list of errors/warnings."""
    errors = []

    skill_md = skill_dir / "SKILL.md"
    if not skill_md.exists():
        skill_md = skill_dir / "skill.md"
    if not skill_md.exists():
        errors.append(ValidationError("error", "SKILL.md not found"))
        return errors

    content = skill_md.read_text(encoding="utf-8", errors="replace")

    fm = parse_frontmatter(content)
    if fm is None:
        errors.append(ValidationError("error", "No YAML frontmatter (--- ... ---) found"))
        return errors

    if "name" not in fm or not fm["name"].strip():
        errors.append(ValidationError("error", "Frontmatter missing 'name' field"))

    if "description" not in fm or not fm["description"].strip():
        errors.append(ValidationError("error", "Frontmatter missing 'description' field"))
    elif len(fm["description"]) < 20:
        errors.append(ValidationError("warning",
            f"Description is very short ({len(fm['description'])} chars). "
            "Include what the skill does AND when to use it."))

    if fm.get("name"):
        name = fm["name"]
        if not re.match(r'^[a-z0-9][a-z0-9-]*$', name):
            errors.append(ValidationError("warning",
                f"Name '{name}' should be lowercase alphanumeric with hyphens"))
        if len(name) > 64:
            errors.append(ValidationError("error",
                f"Name too long ({len(name)} chars, max 64)"))

    body_match = re.search(r"^---\n.*?\n---\n?(.*)", content, re.DOTALL)
    body = body_match.group(1).strip() if body_match else ""
    if not body:
        errors.append(ValidationError("warning", "SKILL.md body is empty"))
    elif len(body.split("\n")) > 500:
        errors.append(ValidationError("warning",
            f"SKILL.md body is {len(body.split(chr(10)))} lines. "
            "Consider splitting into reference files to stay under 500."))

    for bad_file in ("README.md", "INSTALLATION_GUIDE.md", "CHANGELOG.md",
                      "QUICK_REFERENCE.md"):
        if (skill_dir / bad_file).exists():
            errors.append(ValidationError("warning",
                f"Unnecessary file '{bad_file}' -- skills should only contain "
                "SKILL.md and supporting resources."))

    for subdir in ("scripts", "references", "assets"):
        d = skill_dir / subdir
        if d.is_dir():
            referenced = subdir in content or any(
                f.name in content for f in d.iterdir() if f.is_file()
            )
            if not referenced:
                errors.append(ValidationError("warning",
                    f"'{subdir}/' exists but is not referenced in SKILL.md"))

    total_size = sum(
        f.stat().st_size for f in skill_dir.rglob("*")
        if f.is_file() and not should_skip(f, skill_dir)
    )
    if total_size > MAX_BUNDLE_SIZE:
        errors.append(ValidationError("error",
            f"Total size {total_size // (1024*1024)}MB exceeds {MAX_BUNDLE_SIZE // (1024*1024)}MB limit"))

    return errors


def should_skip(path: Path, root: Path) -> bool:
    """Check if a file should be excluded from packaging."""
    rel = path.relative_to(root)
    for part in rel.parts:
        if part in SKIP_PATTERNS:
            return True
        if part.startswith("."):
            return True
    for pattern in SKIP_PATTERNS:
        if pattern.startswith("*") and path.name.endswith(pattern[1:]):
            return True
    return False


def package_skill(skill_dir: Path, output_dir: Path) -> Path | None:
    """Package a skill into a .skill zip file."""
    errors = validate_skill(skill_dir)

    has_errors = any(e.level == "error" for e in errors)
    has_warnings = any(e.level == "warning" for e in errors)

    if errors:
        print(f"Validation results for '{skill_dir.name}':")
        for e in errors:
            print(str(e))
        print()

    if has_errors:
        print("Packaging FAILED -- fix errors above and retry.")
        return None

    if has_warnings:
        print("Warnings found but proceeding with packaging.\n")

    output_dir.mkdir(parents=True, exist_ok=True)
    output_file = output_dir / f"{skill_dir.name}.skill"

    file_count = 0
    with zipfile.ZipFile(output_file, "w", zipfile.ZIP_DEFLATED) as zf:
        for f in sorted(skill_dir.rglob("*")):
            if not f.is_file():
                continue
            if should_skip(f, skill_dir):
                continue
            arcname = str(f.relative_to(skill_dir.parent))
            zf.write(f, arcname)
            file_count += 1

    size_kb = output_file.stat().st_size / 1024
    print(f"Packaged: {output_file} ({file_count} files, {size_kb:.1f} KB)")
    return output_file


def main():
    if len(sys.argv) < 2:
        print("Usage: package_skill.py <path/to/skill-folder> [output-directory]")
        sys.exit(1)

    skill_dir = Path(sys.argv[1]).resolve()
    if not skill_dir.is_dir():
        print(f"Error: '{skill_dir}' is not a directory", file=sys.stderr)
        sys.exit(1)

    output_dir = Path(sys.argv[2]).resolve() if len(sys.argv) > 2 else Path(".")

    result = package_skill(skill_dir, output_dir)
    if result is None:
        sys.exit(1)


if __name__ == "__main__":
    main()
