#!/usr/bin/env python3
"""
Initialize a new Xagent skill with proper structure.

Usage:
    python3 init_skill.py <skill-name> --path <output-directory> [options]

Options:
    --resources scripts,references,assets   Create resource subdirectories
    --examples                              Add example placeholder files
"""

import argparse
import os
import sys
import re


SKILL_MD_TEMPLATE = """---
name: {name}
description: {description}
---

# {title}

{body}
"""

EXAMPLE_SCRIPT = """#!/usr/bin/env python3
\"\"\"Example script for {name} skill.\"\"\"

import sys

def main():
    print("Hello from {name} skill")

if __name__ == "__main__":
    main()
"""

EXAMPLE_REFERENCE = """# {title} Reference

## Overview

Add reference documentation here that the agent should consult when working
on tasks related to this skill.

## Key Concepts

- Concept 1
- Concept 2

## API Reference

Document relevant APIs, schemas, or data structures here.
"""


def normalize_name(name: str) -> str:
    """Normalize skill name to lowercase-hyphen format."""
    name = re.sub(r'[^a-zA-Z0-9\s-]', '', name)
    name = re.sub(r'\s+', '-', name.strip())
    name = re.sub(r'-+', '-', name)
    return name.lower()[:64]


def title_from_name(name: str) -> str:
    """Convert hyphenated name to title case."""
    return name.replace('-', ' ').title()


def create_skill(name: str, path: str, resources: list, examples: bool, description: str):
    """Create skill directory structure."""
    skill_dir = os.path.join(path, name)

    if os.path.exists(skill_dir):
        print(f"Error: {skill_dir} already exists", file=sys.stderr)
        sys.exit(1)

    os.makedirs(skill_dir, exist_ok=True)

    title = title_from_name(name)

    if not description:
        description = f"TODO: Describe what {title} does and when to use it."

    body_parts = []
    body_parts.append("## Quick Start\n")
    body_parts.append("TODO: Add quick start instructions here.\n")

    if "scripts" in resources:
        body_parts.append("## Scripts\n")
        body_parts.append(f"Run the main script:\n```bash\npython3 scripts/main.py\n```\n")

    if "references" in resources:
        body_parts.append("## References\n")
        body_parts.append("- See [reference.md](references/reference.md) for detailed documentation.\n")

    skill_md = SKILL_MD_TEMPLATE.format(
        name=name,
        description=description,
        title=title,
        body="\n".join(body_parts),
    )

    with open(os.path.join(skill_dir, "SKILL.md"), "w") as f:
        f.write(skill_md)

    for res in resources:
        res_dir = os.path.join(skill_dir, res)
        os.makedirs(res_dir, exist_ok=True)

        if examples:
            if res == "scripts":
                script_path = os.path.join(res_dir, "main.py")
                with open(script_path, "w") as f:
                    f.write(EXAMPLE_SCRIPT.format(name=name))
                os.chmod(script_path, 0o755)

            elif res == "references":
                with open(os.path.join(res_dir, "reference.md"), "w") as f:
                    f.write(EXAMPLE_REFERENCE.format(title=title))

            elif res == "assets":
                with open(os.path.join(res_dir, ".gitkeep"), "w") as f:
                    pass

    print(f"Created skill: {skill_dir}")
    print(f"  SKILL.md")
    for res in resources:
        print(f"  {res}/")
    print(f"\nNext: edit {os.path.join(skill_dir, 'SKILL.md')} to add instructions.")


def main():
    parser = argparse.ArgumentParser(description="Initialize a new Xagent skill")
    parser.add_argument("name", help="Skill name (lowercase-hyphen format)")
    parser.add_argument("--path", required=True, help="Output directory")
    parser.add_argument("--resources", default="",
                        help="Comma-separated: scripts,references,assets")
    parser.add_argument("--examples", action="store_true",
                        help="Add example placeholder files")
    parser.add_argument("--description", default="",
                        help="Skill description for frontmatter")

    args = parser.parse_args()

    name = normalize_name(args.name)
    if not name:
        print("Error: invalid skill name", file=sys.stderr)
        sys.exit(1)

    resources = [r.strip() for r in args.resources.split(",") if r.strip()]
    valid_resources = {"scripts", "references", "assets"}
    for r in resources:
        if r not in valid_resources:
            print(f"Error: unknown resource type '{r}'. Valid: {valid_resources}",
                  file=sys.stderr)
            sys.exit(1)

    os.makedirs(args.path, exist_ok=True)
    create_skill(name, args.path, resources, args.examples, args.description)


if __name__ == "__main__":
    main()
