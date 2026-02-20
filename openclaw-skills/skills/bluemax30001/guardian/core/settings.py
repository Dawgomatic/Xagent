#!/usr/bin/env python3
"""Shared settings and path resolution for the Guardian skill."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any, Dict, List, Optional

DEFAULT_CONFIG: Dict[str, Any] = {
    "scan_paths": ["auto"],
    "db_path": "auto",
    "scan_interval_minutes": 2,
    "severity_threshold": "medium",
    "dismissed_signatures": [],
    "custom_definitions_dir": None,
}


def resolve_workspace(cwd: Optional[Path] = None) -> Path:
    """Resolve Guardian workspace using env vars and documented fallback order."""
    guardian_workspace = os.environ.get("GUARDIAN_WORKSPACE")
    if guardian_workspace:
        return Path(guardian_workspace).expanduser().resolve()

    openclaw_workspace = os.environ.get("OPENCLAW_WORKSPACE")
    if openclaw_workspace:
        return Path(openclaw_workspace).expanduser().resolve()

    default_workspace = Path.home() / ".openclaw" / "workspace"
    if default_workspace.exists():
        return default_workspace.resolve()

    return (cwd or Path.cwd()).resolve()


def skill_root() -> Path:
    """Return the absolute root path of this Guardian skill checkout."""
    return Path(__file__).resolve().parent.parent


def definitions_dir(config: Optional[Dict[str, Any]] = None) -> Path:
    """Return definitions directory, honoring optional custom definitions path."""
    if config and config.get("custom_definitions_dir"):
        return Path(str(config["custom_definitions_dir"])).expanduser().resolve()
    return skill_root() / "definitions"


def default_db_path(workspace: Optional[Path] = None) -> Path:
    """Return default SQLite DB path for Guardian state."""
    ws = workspace or resolve_workspace()
    return ws / "guardian.db"


def _config_path(config_path: Optional[str] = None) -> Path:
    if config_path:
        return Path(config_path).expanduser().resolve()

    env_config = os.environ.get("GUARDIAN_CONFIG")
    if env_config:
        return Path(env_config).expanduser().resolve()

    return skill_root() / "config.json"


def load_config(config_path: Optional[str] = None) -> Dict[str, Any]:
    """Load config.json and merge with defaults."""
    merged = dict(DEFAULT_CONFIG)
    cfg_path = _config_path(config_path)

    if cfg_path.exists():
        try:
            loaded = json.loads(cfg_path.read_text(encoding="utf-8"))
            if isinstance(loaded, dict):
                merged.update(loaded)
        except json.JSONDecodeError as exc:
            raise ValueError(f"Invalid Guardian config JSON: {cfg_path}") from exc

    return merged


def resolve_scan_paths(config: Dict[str, Any]) -> List[Path]:
    """Resolve scan target paths from config with auto-discovery support."""
    raw_paths = config.get("scan_paths", ["auto"])
    if not isinstance(raw_paths, list) or not raw_paths:
        raw_paths = ["auto"]

    resolved: List[Path] = []
    for raw in raw_paths:
        if raw == "auto":
            for auto_path in [
                Path.home() / ".openclaw" / "agents",
                Path.home() / ".openclaw" / "cron",
            ]:
                if auto_path.exists():
                    resolved.append(auto_path)
        else:
            candidate = Path(str(raw)).expanduser().resolve()
            if candidate.exists():
                resolved.append(candidate)

    deduped: List[Path] = []
    seen = set()
    for item in resolved:
        normalized = str(item)
        if normalized not in seen:
            seen.add(normalized)
            deduped.append(item)

    return deduped


def severity_min_score(level: str) -> int:
    """Convert severity threshold label to a minimum numeric score."""
    mapping = {
        "low": 0,
        "medium": 50,
        "high": 80,
        "critical": 90,
    }
    return mapping.get(str(level).lower(), 50)
