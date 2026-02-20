"""Definition file validation tests."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Dict, List

from conftest import ROOT


def _definition_items(payload: Dict) -> List[Dict]:
    if "signatures" in payload:
        return payload["signatures"]
    if "checks" in payload:
        return payload["checks"]
    return []


def test_definition_files_load_and_patterns_compile() -> None:
    defs_dir = ROOT / "definitions"
    definition_files = sorted(defs_dir.glob("*.json"))
    assert definition_files, "No definition JSON files found"

    for path in definition_files:
        data = json.loads(path.read_text(encoding="utf-8"))
        assert isinstance(data, dict)
        items = _definition_items(data)
        for item in items:
            pattern = item.get("pattern")
            if pattern:
                re.compile(pattern)


def test_no_duplicate_signature_ids() -> None:
    defs_dir = ROOT / "definitions"
    seen = set()
    duplicates = []

    for path in sorted(defs_dir.glob("*.json")):
        if path.name == "manifest.json":
            continue
        data = json.loads(path.read_text(encoding="utf-8"))
        items = _definition_items(data)
        for item in items:
            sig_id = item.get("id")
            if not sig_id:
                continue
            if sig_id in seen:
                duplicates.append(sig_id)
            seen.add(sig_id)

    assert not duplicates, f"Duplicate signature IDs: {duplicates}"
