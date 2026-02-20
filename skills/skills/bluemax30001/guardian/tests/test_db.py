"""GuardianDB tests for CRUD, bookmarks, deduplication, and WAL mode."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from pathlib import Path

from guardian_db import GuardianDB


def test_db_bookmarks_and_threat_dedup(tmp_path: Path) -> None:
    db_path = tmp_path / "guardian.db"
    db = GuardianDB(db_path=str(db_path))

    db.set_bookmark("/tmp/a.jsonl", 128, 1234.5)
    offset, mtime = db.get_bookmark("/tmp/a.jsonl")
    assert offset == 128
    assert mtime == 1234.5

    inserted = db.add_threat(
        sig_id="INJ-001",
        category="prompt_injection",
        severity="critical",
        score=95,
        evidence="ignore previous instructions",
        description="test",
        blocked=True,
        channel="discord",
        source_file="discord:msg1",
        message_hash="abc123",
    )
    duplicate = db.add_threat(
        sig_id="INJ-001",
        category="prompt_injection",
        severity="critical",
        score=95,
        evidence="ignore previous instructions",
        description="test",
        blocked=True,
        channel="discord",
        source_file="discord:msg1",
        message_hash="abc123",
    )

    assert inserted is not None
    assert duplicate is None
    assert len(db.get_threats(hours=24, limit=10)) == 1

    db.close()


def test_db_wal_mode_and_grants(tmp_path: Path) -> None:
    db_path = tmp_path / "guardian.db"
    db = GuardianDB(db_path=str(db_path))

    mode = db.conn.execute("PRAGMA journal_mode").fetchone()[0]
    assert str(mode).lower() == "wal"

    expiry = (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()
    grant_id = db.add_grant("send_email", "discord", "action_type", expiry)
    assert db.check_grant("send_email", "discord") is True

    active = db.list_active_grants()
    assert any(grant["id"] == grant_id for grant in active)

    db.revoke_grant(grant_id)
    assert db.check_grant("send_email", "discord") is False
    db.close()
