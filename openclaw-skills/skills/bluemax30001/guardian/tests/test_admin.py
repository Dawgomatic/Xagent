"""Admin CLI tests for status controls and threat management."""

from __future__ import annotations

import importlib.util
import json
from pathlib import Path
from typing import Any


def _load_admin_module(root: Path) -> Any:
    script = root / "scripts" / "admin.py"
    spec = importlib.util.spec_from_file_location("admin_cli", script)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    spec.loader.exec_module(module)
    return module


def _seed_config(path: Path) -> None:
    path.write_text(
        json.dumps(
            {
                "enabled": True,
                "admin_override": False,
                "db_path": "auto",
                "admin": {"disable_until": None},
                "false_positive_suppression": {"allowlist_patterns": []},
            }
        ),
        encoding="utf-8",
    )


def test_disable_enable_and_bypass_flow(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    admin = _load_admin_module(root)

    cfg_path = tmp_path / "config.json"
    db_path = tmp_path / "guardian.db"
    _seed_config(cfg_path)

    admin.config_path = lambda: cfg_path
    admin.resolve_workspace = lambda: tmp_path

    assert admin.main(["disable"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert cfg["enabled"] is False

    assert admin.main(["enable"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert cfg["enabled"] is True

    assert admin.main(["bypass", "--on"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert cfg["admin_override"] is True

    assert admin.main(["bypass", "--off"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert cfg["admin_override"] is False

    assert db_path.exists()


def test_disable_with_until_sets_timestamp(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    admin = _load_admin_module(root)

    cfg_path = tmp_path / "config.json"
    _seed_config(cfg_path)

    admin.config_path = lambda: cfg_path
    admin.resolve_workspace = lambda: tmp_path

    assert admin.main(["disable", "--until", "2h"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))

    assert cfg["enabled"] is False
    assert cfg["admin"]["disable_until"] is not None


def test_dismiss_marks_matching_threats(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    admin = _load_admin_module(root)

    cfg_path = tmp_path / "config.json"
    _seed_config(cfg_path)

    admin.config_path = lambda: cfg_path
    admin.resolve_workspace = lambda: tmp_path

    conn = admin.connect_db(tmp_path / "guardian.db")
    conn.execute(
        """
        INSERT INTO threats
        (detected_at, sig_id, category, severity, score, evidence, description, blocked, channel, source_file, message_hash, dismissed)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            "2026-02-19T00:00:00+00:00",
            "INJ-004",
            "prompt_injection",
            "critical",
            90,
            "ignore",
            "test",
            1,
            "discord",
            "x",
            "abc",
            0,
        ),
    )
    conn.commit()
    conn.close()

    assert admin.main(["dismiss", "INJ-004"]) == 0

    conn = admin.connect_db(tmp_path / "guardian.db")
    row = conn.execute("SELECT dismissed FROM threats WHERE sig_id='INJ-004'").fetchone()
    conn.close()

    assert row["dismissed"] == 1


def test_allowlist_add_and_remove(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    admin = _load_admin_module(root)

    cfg_path = tmp_path / "config.json"
    _seed_config(cfg_path)

    admin.config_path = lambda: cfg_path
    admin.resolve_workspace = lambda: tmp_path

    assert admin.main(["allowlist", "add", "benign pattern"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert "benign pattern" in cfg["false_positive_suppression"]["allowlist_patterns"]

    assert admin.main(["allowlist", "remove", "benign pattern"]) == 0
    cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
    assert "benign pattern" not in cfg["false_positive_suppression"]["allowlist_patterns"]


def test_threats_clear_removes_dismissed_rows(tmp_path: Path) -> None:
    root = Path(__file__).resolve().parents[1]
    admin = _load_admin_module(root)

    cfg_path = tmp_path / "config.json"
    _seed_config(cfg_path)

    admin.config_path = lambda: cfg_path
    admin.resolve_workspace = lambda: tmp_path

    conn = admin.connect_db(tmp_path / "guardian.db")
    conn.execute(
        """
        INSERT INTO threats
        (detected_at, sig_id, category, severity, score, evidence, description, blocked, channel, source_file, message_hash, dismissed)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            "2026-02-19T00:00:00+00:00",
            "TAB-001",
            "tool_abuse",
            "high",
            85,
            "rm -rf",
            "test",
            1,
            "discord",
            "x",
            "row1",
            1,
        ),
    )
    conn.execute(
        """
        INSERT INTO threats
        (detected_at, sig_id, category, severity, score, evidence, description, blocked, channel, source_file, message_hash, dismissed)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            "2026-02-19T00:00:01+00:00",
            "INJ-002",
            "prompt_injection",
            "critical",
            95,
            "ignore",
            "test",
            1,
            "discord",
            "x",
            "row2",
            0,
        ),
    )
    conn.commit()
    conn.close()

    assert admin.main(["threats", "--clear"]) == 0

    conn = admin.connect_db(tmp_path / "guardian.db")
    rows = conn.execute("SELECT COUNT(*) AS cnt FROM threats").fetchone()
    conn.close()

    assert rows["cnt"] == 1
