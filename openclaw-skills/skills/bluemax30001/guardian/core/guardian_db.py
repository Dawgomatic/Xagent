#!/usr/bin/env python3
"""Guardian SQLite persistence layer for threats, metrics, bookmarks, and grants."""

from __future__ import annotations

import json
import sqlite3
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

try:
    from .settings import default_db_path, skill_root
except ImportError:
    from settings import default_db_path, skill_root


class GuardianDB:
    """SQLite-backed data access object for Guardian runtime state."""

    def __init__(self, db_path: Optional[str] = None) -> None:
        preferred = Path(db_path).expanduser().resolve() if db_path else default_db_path()
        candidates = [preferred, (skill_root() / "guardian.db").resolve(), Path("/tmp/guardian.db").resolve()]

        last_error: Optional[Exception] = None
        self.db_path = preferred
        for candidate in candidates:
            try:
                candidate.parent.mkdir(parents=True, exist_ok=True)
                conn = sqlite3.connect(candidate)
                conn.row_factory = sqlite3.Row
                conn.execute("PRAGMA journal_mode=WAL")
                self.db_path = candidate
                self.conn = conn
                break
            except (OSError, sqlite3.OperationalError) as exc:
                last_error = exc
        else:
            raise sqlite3.OperationalError(f"Unable to initialize Guardian DB: {last_error}")

        self._migrate()

    def _migrate(self) -> None:
        """Create required tables and indexes if missing."""
        c = self.conn
        c.executescript(
            """
            CREATE TABLE IF NOT EXISTS scan_bookmarks (
                file_path TEXT PRIMARY KEY,
                last_offset INTEGER DEFAULT 0,
                last_mtime REAL DEFAULT 0,
                updated_at TEXT
            );
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
            );
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
            );
            CREATE TABLE IF NOT EXISTS cache_grants (
                id TEXT PRIMARY KEY,
                action TEXT NOT NULL,
                channel TEXT,
                scope TEXT,
                granted_at TEXT NOT NULL,
                expires_at TEXT NOT NULL,
                granted_by TEXT,
                revoked INTEGER DEFAULT 0,
                revoked_at TEXT
            );
            CREATE TABLE IF NOT EXISTS pending_confirms (
                id TEXT PRIMARY KEY,
                action TEXT NOT NULL,
                details TEXT,
                channel TEXT,
                requested_at TEXT NOT NULL,
                expires_at TEXT NOT NULL,
                status TEXT DEFAULT 'pending',
                resolved_at TEXT,
                pin TEXT
            );
            CREATE TABLE IF NOT EXISTS config_audits (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                audited_at TEXT NOT NULL,
                score INTEGER,
                warning_count INTEGER,
                passed_count INTEGER,
                warnings TEXT,
                passed TEXT
            );
            CREATE INDEX IF NOT EXISTS idx_threats_detected ON threats(detected_at);
            CREATE INDEX IF NOT EXISTS idx_threats_category ON threats(category);
            CREATE INDEX IF NOT EXISTS idx_metrics_period ON metrics(period, period_start);
            """
        )
        c.commit()

    def _now(self) -> str:
        """Return current UTC timestamp in ISO-8601 format."""
        return datetime.now(timezone.utc).isoformat()

    def get_bookmark(self, file_path: str) -> Tuple[int, float]:
        """Return last scanned offset and mtime for a file bookmark."""
        row = self.conn.execute(
            "SELECT last_offset, last_mtime FROM scan_bookmarks WHERE file_path=?",
            (file_path,),
        ).fetchone()
        return (row["last_offset"], row["last_mtime"]) if row else (0, 0.0)

    def set_bookmark(self, file_path: str, offset: int, mtime: float) -> None:
        """Set or update bookmark for incremental scanning."""
        self.conn.execute(
            "INSERT OR REPLACE INTO scan_bookmarks (file_path, last_offset, last_mtime, updated_at) VALUES (?,?,?,?)",
            (file_path, offset, mtime, self._now()),
        )
        self.conn.commit()

    def add_threat(
        self,
        sig_id: str,
        category: str,
        severity: str,
        score: int,
        evidence: str,
        description: str,
        blocked: bool,
        channel: str,
        source_file: str,
        message_hash: str,
    ) -> Optional[int]:
        """Insert a detected threat row, deduplicated by message hash."""
        try:
            cur = self.conn.execute(
                "INSERT INTO threats (detected_at, sig_id, category, severity, score, evidence, description, blocked, channel, source_file, message_hash) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
                (
                    self._now(),
                    sig_id,
                    category,
                    severity,
                    score,
                    evidence,
                    description,
                    1 if blocked else 0,
                    channel,
                    source_file,
                    message_hash,
                ),
            )
            self.conn.commit()
            return int(cur.lastrowid)
        except sqlite3.IntegrityError:
            return None

    def get_threats(self, hours: int = 24, limit: int = 50) -> List[Dict[str, Any]]:
        """Return recent non-dismissed threats."""
        cutoff = (datetime.now(timezone.utc) - timedelta(hours=hours)).isoformat()
        rows = self.conn.execute(
            "SELECT * FROM threats WHERE detected_at >= ? AND dismissed=0 ORDER BY score DESC, detected_at DESC LIMIT ?",
            (cutoff, limit),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_threat_summary(self, hours: int = 24) -> Dict[str, Any]:
        """Return aggregate category counts for recent threats."""
        cutoff = (datetime.now(timezone.utc) - timedelta(hours=hours)).isoformat()
        rows = self.conn.execute(
            "SELECT category, COUNT(*) as cnt, SUM(blocked) as blk FROM threats WHERE detected_at >= ? AND dismissed=0 GROUP BY category",
            (cutoff,),
        ).fetchall()
        total = sum(r["cnt"] for r in rows)
        blocked = sum(r["blk"] for r in rows)
        cats = {r["category"]: r["cnt"] for r in rows}
        return {
            "total": total,
            "blocked": blocked,
            "injections": cats.get("prompt_injection", 0),
            "exfiltration": cats.get("data_exfiltration", 0),
            "toolAbuse": cats.get("tool_abuse", 0),
            "socialEng": cats.get("social_engineering", 0),
            "categories": cats,
        }

    def dismiss_threat(self, threat_id: int) -> None:
        """Mark a threat as dismissed."""
        self.conn.execute("UPDATE threats SET dismissed=1 WHERE id=?", (threat_id,))
        self.conn.commit()

    def record_scan(
        self,
        messages_scanned: int,
        files_scanned: int,
        clean: int,
        at_risk: int,
        blocked: int,
        categories: Dict[str, int],
        health_score: int,
    ) -> None:
        """Record hourly and daily scan metrics in an upsert-friendly format."""
        now = datetime.now(timezone.utc)
        hourly_start = now.replace(minute=0, second=0, microsecond=0).isoformat()
        daily_start = now.replace(hour=0, minute=0, second=0, microsecond=0).isoformat()
        cats_json = json.dumps(categories) if categories else "{}"

        for period, pstart in [("hourly", hourly_start), ("daily", daily_start)]:
            self.conn.execute(
                """
                INSERT INTO metrics (period, period_start, messages_scanned, files_scanned, clean, at_risk, blocked, categories, health_score)
                VALUES (?,?,?,?,?,?,?,?,?)
                ON CONFLICT(period, period_start) DO UPDATE SET
                    messages_scanned = messages_scanned + excluded.messages_scanned,
                    files_scanned = excluded.files_scanned,
                    clean = clean + excluded.clean,
                    at_risk = at_risk + excluded.at_risk,
                    blocked = blocked + excluded.blocked,
                    categories = excluded.categories,
                    health_score = excluded.health_score
                """,
                (period, pstart, messages_scanned, files_scanned, clean, at_risk, blocked, cats_json, health_score),
            )
        self.conn.commit()

    def get_metrics(self, period: str = "daily", days: int = 7) -> List[Dict[str, Any]]:
        """Fetch historical metrics for a period over trailing days."""
        cutoff = (datetime.now(timezone.utc) - timedelta(days=days)).isoformat()
        rows = self.conn.execute(
            "SELECT * FROM metrics WHERE period=? AND period_start >= ? ORDER BY period_start",
            (period, cutoff),
        ).fetchall()
        return [dict(r) for r in rows]

    def get_current_totals(self) -> Dict[str, Any]:
        """Return totals aggregated from hourly metrics."""
        row = self.conn.execute(
            "SELECT SUM(messages_scanned) as total_scanned, SUM(at_risk) as total_threats, SUM(blocked) as total_blocked, MIN(period_start) as first_scan FROM metrics WHERE period='hourly'"
        ).fetchone()
        if row and row["total_scanned"]:
            return {
                "totalScanned": row["total_scanned"],
                "totalThreats": row["total_threats"] or 0,
                "totalBlocked": row["total_blocked"] or 0,
                "firstScan": row["first_scan"],
            }
        return {"totalScanned": 0, "totalThreats": 0, "totalBlocked": 0, "firstScan": None}

    def add_grant(
        self,
        action: str,
        channel: str,
        scope: str,
        expires_at: str,
        granted_by: Optional[str] = None,
    ) -> str:
        """Create an approval grant and return grant ID."""
        gid = str(uuid.uuid4())
        self.conn.execute(
            "INSERT INTO cache_grants (id, action, channel, scope, granted_at, expires_at, granted_by) VALUES (?,?,?,?,?,?,?)",
            (gid, action, channel, scope, self._now(), expires_at, granted_by),
        )
        self.conn.commit()
        return gid

    def check_grant(self, action: str, channel: str) -> bool:
        """Return True when a non-revoked and non-expired grant exists."""
        now = self._now()
        row = self.conn.execute(
            "SELECT id FROM cache_grants WHERE action=? AND channel=? AND revoked=0 AND expires_at > ? LIMIT 1",
            (action, channel, now),
        ).fetchone()
        return row is not None

    def list_active_grants(self) -> List[Dict[str, Any]]:
        """Return all active grants sorted by newest first."""
        now = self._now()
        rows = self.conn.execute(
            "SELECT * FROM cache_grants WHERE revoked=0 AND expires_at > ? ORDER BY granted_at DESC",
            (now,),
        ).fetchall()
        return [dict(r) for r in rows]

    def revoke_grant(self, grant_id: str) -> None:
        """Revoke a specific grant by ID."""
        self.conn.execute("UPDATE cache_grants SET revoked=1, revoked_at=? WHERE id=?", (self._now(), grant_id))
        self.conn.commit()

    def revoke_all_grants(self) -> None:
        """Revoke all active grants."""
        self.conn.execute("UPDATE cache_grants SET revoked=1, revoked_at=? WHERE revoked=0", (self._now(),))
        self.conn.commit()

    def add_pending(self, action: str, details: Any, channel: str, pin: str, expires_at: str) -> str:
        """Insert a pending confirmation request and return pending ID."""
        pid = str(uuid.uuid4())
        details_json = json.dumps(details) if isinstance(details, dict) else str(details)
        self.conn.execute(
            "INSERT INTO pending_confirms (id, action, details, channel, requested_at, expires_at, pin) VALUES (?,?,?,?,?,?,?)",
            (pid, action, details_json, channel, self._now(), expires_at, pin),
        )
        self.conn.commit()
        return pid

    def get_pending(self) -> List[Dict[str, Any]]:
        """Return active pending confirmations."""
        now = self._now()
        rows = self.conn.execute(
            "SELECT * FROM pending_confirms WHERE status='pending' AND expires_at > ? ORDER BY requested_at DESC",
            (now,),
        ).fetchall()
        return [dict(r) for r in rows]

    def resolve_pending(self, pending_id: str, status: str) -> None:
        """Resolve a pending confirmation with a final status."""
        self.conn.execute(
            "UPDATE pending_confirms SET status=?, resolved_at=? WHERE id=?",
            (status, self._now(), pending_id),
        )
        self.conn.commit()

    def approve_all_pending(self) -> None:
        """Approve all currently pending and unexpired confirmations."""
        now = self._now()
        self.conn.execute(
            "UPDATE pending_confirms SET status='approved', resolved_at=? WHERE status='pending' AND expires_at > ?",
            (now, now),
        )
        self.conn.commit()

    def close(self) -> None:
        """Close underlying SQLite connection."""
        self.conn.close()
