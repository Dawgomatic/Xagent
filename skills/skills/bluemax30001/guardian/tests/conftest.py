"""Pytest fixtures and import helpers for Guardian tests."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CORE_DIR = ROOT / "core"

if str(CORE_DIR) not in sys.path:
    sys.path.insert(0, str(CORE_DIR))


def load_guardian_script_module():
    """Load scripts/guardian.py as a Python module for unit testing."""
    script_path = ROOT / "scripts" / "guardian.py"
    spec = importlib.util.spec_from_file_location("guardian_cli", script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    spec.loader.exec_module(module)
    return module
