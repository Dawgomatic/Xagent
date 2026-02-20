"""Dashboard export tests for JSON payload fields and metrics."""

from __future__ import annotations

import importlib.util
import json
import sqlite3
from pathlib import Path
from typing import Any


def _load_dashboard_module(root: Path) -> Any:
    script = root / "scripts" / "dashboard_export.py"
    spec = importlib.util.spec_from_file_location("dashboard_export", script)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    spec.loader.exec_module(module)
    return module


def _write_config(path: Path, db_path: Path, enabled: bool = True, bypass: bool = False) -> None:
    path.write_text(
        json.dumps(
            {
                "enabled": enabled,
                "admin_override": bypass,
                "db_path": str(db_path),
                "channels": {"monitor_all": True},
                "alerts": {"notify_on_critical": True},
            }
        ),
        encoding="utf-8",
    )


def _init_db(path: Path) -> sqlite3.Connection:
    conn = sqlite3.connect(path)
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS metrics (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            period TEXT NOT NULL,
            period_start TEXT NOT NULL,
            messages_scanned INTEGER DEFAULT 0,
            files_scanned INTEGER DEFAULT 0,
            clean INTEGER DEFAULT 0,
            at_risk INTEGER DEFAULT 0,
            blocked INTEGER DEFAULT 0,
            categories TEXT,
            health_score INTEGER,
            UNIQUE(period, period_start)
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS threats (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            detected_at TEXT NOT NULL,
            sig_id TEXT,
            category TEXT,
            severity TEXT,
            score INTEGER,
            evidence TEXT,
            description TEXT,
            blocked INTEGER DEFAULT 0,
            channel TEXT,
            source_file TEXT,
            message_hash TEXT UNIQUE,
            dismissed INTEGER DEFAULT 0
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS config_audits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            audited_at TEXT NOT NULL,
            score INTEGER,
            warning_count INTEGER,
            passed_count INTEGER,
            warnings TEXT,
            passed TEXT
        )
        """
    )
    conn.commit()
    return conn


def test_dashboard_export_structure_when_db_missing(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    module = _load_dashboard_module(root)

    config_path = tmp_path / "config.json"
    db_path = tmp_path / "guardian.db"
    _write_config(config_path, db_path)

    payload = module.export_dashboard(config_path=str(config_path), db_path=str(db_path))

    expected_keys = {
        "status",
        "health",
        "scanned",
        "threats_24h",
        "blocked_24h",
        "last_scan",
        "top_categories",
        "critical_pending",
        "config_score",
        "recent_threats",
        "admin_mode",
    }
    assert set(payload.keys()) == expected_keys
    assert payload["scanned"] == 0


def test_dashboard_export_aggregates_metrics_and_threats(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    module = _load_dashboard_module(root)

    config_path = tmp_path / "config.json"
    db_path = tmp_path / "guardian.db"
    _write_config(config_path, db_path, enabled=True, bypass=False)

    conn = _init_db(db_path)
    conn.execute(
        "INSERT INTO metrics (period, period_start, messages_scanned, health_score) VALUES (?, ?, ?, ?)",
        ("hourly", "2026-02-19T15:00:00+00:00", 42, 72),
    )
    conn.execute(
        "INSERT INTO threats (detected_at, sig_id, category, severity, score, blocked, channel, message_hash, dismissed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
        ("2026-02-19T15:10:00+00:00", "INJ-004", "prompt_injection", "critical", 92, 1, "discord", "hash1", 0),
    )
    conn.execute(
        "INSERT INTO threats (detected_at, sig_id, category, severity, score, blocked, channel, message_hash, dismissed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
        ("2026-02-19T15:11:00+00:00", "TAB-001", "tool_abuse", "critical", 91, 0, "discord", "hash2", 0),
    )
    conn.execute(
        "INSERT INTO config_audits (audited_at, score, warning_count, passed_count, warnings, passed) VALUES (?, ?, ?, ?, ?, ?)",
        ("2026-02-19T15:12:00+00:00", 88, 1, 3, "[]", "[]"),
    )
    conn.commit()
    conn.close()

    payload = module.export_dashboard(config_path=str(config_path), db_path=str(db_path))

    assert payload["status"] == "active"
    assert payload["health"] == 72
    assert payload["scanned"] == 42
    assert payload["threats_24h"] == 2
    assert payload["blocked_24h"] == 1
    assert payload["critical_pending"] == 1
    assert payload["config_score"] == 88
    assert payload["top_categories"]["prompt_injection"] == 1
    assert len(payload["recent_threats"]) == 2


def test_dashboard_export_status_reflects_bypass_or_disabled(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    module = _load_dashboard_module(root)

    config_path = tmp_path / "config.json"
    db_path = tmp_path / "guardian.db"

    _write_config(config_path, db_path, enabled=True, bypass=True)
    payload = module.export_dashboard(config_path=str(config_path), db_path=str(db_path))
    assert payload["status"] == "bypass"

    _write_config(config_path, db_path, enabled=False, bypass=False)
    payload = module.export_dashboard(config_path=str(config_path), db_path=str(db_path))
    assert payload["status"] == "disabled"
