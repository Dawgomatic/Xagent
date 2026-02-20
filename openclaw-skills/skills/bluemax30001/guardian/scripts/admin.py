#!/usr/bin/env python3
"""Guardian administrative CLI for runtime controls and reporting."""

from __future__ import annotations

import argparse
import json
import os
import sqlite3
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

RED = "\033[31m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
CYAN = "\033[36m"
RESET = "\033[0m"


def skill_root() -> Path:
    """Return the Guardian skill root from the current file location."""
    return Path(__file__).resolve().parents[1]


def config_path() -> Path:
    """Return default config file path."""
    return skill_root() / "config.json"


def load_config() -> Dict[str, Any]:
    """Load the Guardian config JSON."""
    path = config_path()
    if not path.exists():
        return {}
    return json.loads(path.read_text(encoding="utf-8"))


def save_config(cfg: Dict[str, Any]) -> None:
    """Persist config JSON with stable formatting."""
    config_path().write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")


def resolve_workspace() -> Path:
    """Resolve workspace using documented env fallback chain."""
    for key in ("GUARDIAN_WORKSPACE", "OPENCLAW_WORKSPACE"):
        raw = os.environ.get(key)
        if raw:
            return Path(raw).expanduser().resolve()
    default_workspace = Path.home() / ".openclaw" / "workspace"
    if default_workspace.exists():
        return default_workspace.resolve()
    return Path.cwd().resolve()


def _writable_parent(path: Path) -> bool:
    """Return True when the database parent directory is writable."""
    parent = path.parent if path.parent.exists() else path.parent
    try:
        parent.mkdir(parents=True, exist_ok=True)
    except OSError:
        return False
    return os.access(parent, os.W_OK)


def resolve_db_path(cfg: Dict[str, Any]) -> Path:
    """Resolve DB path from config with auto behavior support."""
    raw = cfg.get("db_path", "auto")
    candidate = resolve_workspace() / "guardian.db" if raw == "auto" or not raw else Path(str(raw)).expanduser().resolve()
    if _writable_parent(candidate):
        return candidate
    return (skill_root() / "guardian.db").resolve()


def connect_db(db_path: Path) -> sqlite3.Connection:
    """Open sqlite connection, creating parent directory when needed."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    try:
        conn.execute("PRAGMA journal_mode=WAL")
    except sqlite3.OperationalError:
        conn.execute("PRAGMA journal_mode=DELETE")
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
    conn.commit()
    return conn


def parse_duration(raw: str) -> timedelta:
    """Parse duration strings like 30m, 2h, 1d."""
    raw = raw.strip().lower()
    if not raw or len(raw) < 2 or not raw[:-1].isdigit():
        raise ValueError("Duration must look like 30m, 2h, or 1d")
    count = int(raw[:-1])
    unit = raw[-1]
    if unit == "m":
        return timedelta(minutes=count)
    if unit == "h":
        return timedelta(hours=count)
    if unit == "d":
        return timedelta(days=count)
    raise ValueError("Duration unit must be m, h, or d")


def now_utc() -> datetime:
    """Return the current UTC datetime."""
    return datetime.now(timezone.utc)


def parse_iso(value: str) -> datetime:
    """Parse ISO8601 timestamps with optional trailing Z."""
    cleaned = value.replace("Z", "+00:00")
    dt = datetime.fromisoformat(cleaned)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt


def format_iso_z(value: datetime) -> str:
    """Format datetime as UTC ISO8601 with trailing Z."""
    return value.astimezone(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def maybe_auto_resume(cfg: Dict[str, Any]) -> bool:
    """Auto-resume Guardian if disable-until has elapsed."""
    admin = cfg.setdefault("admin", {})
    disable_until = admin.get("disable_until")
    if not disable_until:
        return False
    try:
        until_dt = parse_iso(str(disable_until))
    except ValueError:
        return False
    if until_dt <= now_utc():
        cfg["enabled"] = True
        admin["disable_until"] = None
        save_config(cfg)
        return True
    return False


def determine_status(cfg: Dict[str, Any]) -> str:
    """Resolve admin status value."""
    if not cfg.get("enabled", True):
        return "disabled"
    if cfg.get("admin_override", False):
        return "bypass"
    return "active"


def fetch_summary(conn: sqlite3.Connection, hours: int = 24) -> Dict[str, Any]:
    """Fetch aggregate threat stats from DB."""
    cutoff = (now_utc() - timedelta(hours=hours)).isoformat()
    rows = conn.execute(
        "SELECT COUNT(*) AS total, COALESCE(SUM(blocked),0) AS blocked FROM threats WHERE detected_at >= ? AND dismissed=0",
        (cutoff,),
    ).fetchone()
    top_rows = conn.execute(
        "SELECT category, COUNT(*) AS cnt FROM threats WHERE detected_at >= ? AND dismissed=0 GROUP BY category ORDER BY cnt DESC",
        (cutoff,),
    ).fetchall()
    return {
        "total": int(rows["total"]),
        "blocked": int(rows["blocked"]),
        "categories": {str(row["category"]): int(row["cnt"]) for row in top_rows if row["category"]},
    }


def fetch_recent_threats(conn: sqlite3.Connection, limit: int = 20) -> List[Dict[str, Any]]:
    """Fetch recent threat records."""
    rows = conn.execute(
        """
        SELECT id, detected_at, sig_id, category, severity, score, blocked, channel, dismissed
        FROM threats
        ORDER BY detected_at DESC
        LIMIT ?
        """,
        (limit,),
    ).fetchall()
    return [dict(row) for row in rows]


def render_json(payload: Dict[str, Any]) -> None:
    """Render a JSON payload to stdout."""
    print(json.dumps(payload, indent=2, default=str))


def colorize(text: str, color: str) -> str:
    """Wrap text in ANSI color sequence."""
    return f"{color}{text}{RESET}"


def print_status(cfg: Dict[str, Any], conn: sqlite3.Connection, as_json: bool) -> None:
    """Print runtime status and threat summary."""
    resumed = maybe_auto_resume(cfg)
    status = determine_status(cfg)
    summary = fetch_summary(conn)

    payload = {
        "status": status,
        "enabled": bool(cfg.get("enabled", True)),
        "bypass": bool(cfg.get("admin_override", False)),
        "disable_until": cfg.get("admin", {}).get("disable_until"),
        "summary_24h": summary,
        "auto_resumed": resumed,
    }
    if as_json:
        render_json(payload)
        return

    color = GREEN if status == "active" else (YELLOW if status == "bypass" else RED)
    print(f"Guardian status: {colorize(status.upper(), color)}")
    if payload["disable_until"]:
        print(f"Disabled until: {payload['disable_until']}")
    print(f"Threats (24h): {summary['total']} total, {summary['blocked']} blocked")
    if summary["categories"]:
        print("Top categories:")
        for category, count in summary["categories"].items():
            print(f"  - {category}: {count}")


def cmd_enable(cfg: Dict[str, Any], as_json: bool) -> None:
    """Enable Guardian in config."""
    cfg["enabled"] = True
    cfg.setdefault("admin", {})["disable_until"] = None
    save_config(cfg)
    payload = {"ok": True, "action": "enable", "status": determine_status(cfg)}
    if as_json:
        render_json(payload)
        return
    print(colorize("Guardian enabled.", GREEN))


def cmd_disable(cfg: Dict[str, Any], until: Optional[str], as_json: bool) -> None:
    """Disable Guardian with optional auto-resume duration."""
    cfg["enabled"] = False
    admin = cfg.setdefault("admin", {})
    disable_until = None
    if until:
        disable_until = format_iso_z(now_utc() + parse_duration(until))
    admin["disable_until"] = disable_until
    save_config(cfg)

    payload = {"ok": True, "action": "disable", "disable_until": disable_until}
    if as_json:
        render_json(payload)
        return
    if disable_until:
        print(colorize(f"Guardian disabled until {disable_until}", YELLOW))
    else:
        print(colorize("Guardian disabled.", YELLOW))


def cmd_bypass(cfg: Dict[str, Any], value: bool, as_json: bool) -> None:
    """Enable or disable bypass mode."""
    cfg["admin_override"] = value
    save_config(cfg)

    payload = {"ok": True, "action": "bypass", "bypass": value, "status": determine_status(cfg)}
    if as_json:
        render_json(payload)
        return
    text = "Bypass mode enabled (log-only)." if value else "Bypass mode disabled."
    print(colorize(text, CYAN if value else GREEN))


def cmd_dismiss(conn: sqlite3.Connection, sig_id: str, as_json: bool) -> None:
    """Dismiss all threats matching a signature ID."""
    cur = conn.execute("UPDATE threats SET dismissed=1 WHERE sig_id=?", (sig_id,))
    conn.commit()
    payload = {"ok": True, "action": "dismiss", "sig_id": sig_id, "updated": int(cur.rowcount)}
    if as_json:
        render_json(payload)
        return
    print(colorize(f"Dismissed {cur.rowcount} threat(s) for {sig_id}.", GREEN))


def cmd_allowlist(cfg: Dict[str, Any], action: str, pattern: str, as_json: bool) -> None:
    """Add or remove allowlist regex patterns."""
    fp = cfg.setdefault("false_positive_suppression", {})
    patterns = fp.setdefault("allowlist_patterns", [])
    if not isinstance(patterns, list):
        patterns = []
        fp["allowlist_patterns"] = patterns

    changed = False
    if action == "add":
        if pattern not in patterns:
            patterns.append(pattern)
            changed = True
    else:
        if pattern in patterns:
            patterns.remove(pattern)
            changed = True
    save_config(cfg)

    payload = {
        "ok": True,
        "action": f"allowlist_{action}",
        "pattern": pattern,
        "changed": changed,
        "allowlist_patterns": patterns,
    }
    if as_json:
        render_json(payload)
        return
    verb = "Added" if action == "add" else "Removed"
    print(colorize(f"{verb} allowlist pattern: {pattern}", GREEN if changed else YELLOW))


def cmd_threats(conn: sqlite3.Connection, clear_dismissed: bool, as_json: bool) -> None:
    """List recent threats and optionally clear dismissed entries."""
    cleared = 0
    if clear_dismissed:
        cur = conn.execute("DELETE FROM threats WHERE dismissed=1")
        conn.commit()
        cleared = int(cur.rowcount)

    threats = fetch_recent_threats(conn)
    payload = {"cleared": cleared, "threats": threats}
    if as_json:
        render_json(payload)
        return

    if clear_dismissed:
        print(colorize(f"Cleared {cleared} dismissed threat(s).", GREEN))
    if not threats:
        print("No threats found.")
        return
    for threat in threats:
        blocked_label = "blocked" if threat["blocked"] else "flagged"
        dismissed_label = " dismissed" if threat["dismissed"] else ""
        print(
            f"#{threat['id']} {threat['detected_at']} {threat['sig_id']} "
            f"[{threat['severity']}] {blocked_label}{dismissed_label}"
        )


def cmd_report(conn: sqlite3.Connection, as_json: bool) -> None:
    """Generate a 7-day security report."""
    cutoff = (now_utc() - timedelta(days=7)).isoformat()
    totals = conn.execute(
        "SELECT COUNT(*) AS total, COALESCE(SUM(blocked),0) AS blocked FROM threats WHERE detected_at >= ?",
        (cutoff,),
    ).fetchone()
    severities = conn.execute(
        "SELECT severity, COUNT(*) AS cnt FROM threats WHERE detected_at >= ? GROUP BY severity",
        (cutoff,),
    ).fetchall()
    categories = conn.execute(
        "SELECT category, COUNT(*) AS cnt FROM threats WHERE detected_at >= ? GROUP BY category ORDER BY cnt DESC",
        (cutoff,),
    ).fetchall()

    payload = {
        "window": "7d",
        "threats_total": int(totals["total"]),
        "threats_blocked": int(totals["blocked"]),
        "by_severity": {str(row["severity"]): int(row["cnt"]) for row in severities if row["severity"]},
        "by_category": {str(row["category"]): int(row["cnt"]) for row in categories if row["category"]},
    }
    if as_json:
        render_json(payload)
        return

    print(colorize("Guardian Report (7d)", CYAN))
    print(f"Threats: {payload['threats_total']} total, {payload['threats_blocked']} blocked")
    if payload["by_severity"]:
        print("By severity:")
        for severity, count in payload["by_severity"].items():
            print(f"  - {severity}: {count}")
    if payload["by_category"]:
        print("By category:")
        for category, count in payload["by_category"].items():
            print(f"  - {category}: {count}")


def cmd_update_defs(as_json: bool) -> None:
    """Run definitions updater script as a subprocess."""
    update_script = skill_root() / "definitions" / "update.py"
    proc = subprocess.run(
        [sys.executable, str(update_script)],
        capture_output=True,
        text=True,
        check=False,
    )
    payload = {
        "ok": proc.returncode == 0,
        "returncode": proc.returncode,
        "stdout": proc.stdout.strip(),
        "stderr": proc.stderr.strip(),
    }
    if as_json:
        render_json(payload)
        return

    if proc.returncode == 0:
        print(colorize("Definitions update check complete.", GREEN))
        if proc.stdout.strip():
            print(proc.stdout.strip())
    else:
        print(colorize("Definitions update check failed.", RED))
        if proc.stderr.strip():
            print(proc.stderr.strip())


def build_parser() -> argparse.ArgumentParser:
    """Build command-line argument parser."""
    parser = argparse.ArgumentParser(description="Guardian admin controls")
    parser.add_argument("--json", action="store_true", help="Output JSON")

    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("status", help="Show Guardian status")

    disable_parser = sub.add_parser("disable", help="Disable Guardian")
    disable_parser.add_argument("--until", help="Disable duration, e.g. 2h, 30m, 1d")

    sub.add_parser("enable", help="Enable Guardian")

    bypass_parser = sub.add_parser("bypass", help="Toggle bypass mode")
    mode = bypass_parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--on", action="store_true", help="Enable bypass")
    mode.add_argument("--off", action="store_true", help="Disable bypass")

    dismiss_parser = sub.add_parser("dismiss", help="Dismiss threats by signature ID")
    dismiss_parser.add_argument("sig_id", help="Signature ID")

    allowlist_parser = sub.add_parser("allowlist", help="Manage allowlist patterns")
    allow_sub = allowlist_parser.add_subparsers(dest="allow_cmd", required=True)
    add_parser = allow_sub.add_parser("add", help="Add allowlist pattern")
    add_parser.add_argument("pattern", help="Regex/text pattern")
    rm_parser = allow_sub.add_parser("remove", help="Remove allowlist pattern")
    rm_parser.add_argument("pattern", help="Regex/text pattern")

    threats_parser = sub.add_parser("threats", help="List threats")
    threats_parser.add_argument("--clear", action="store_true", help="Delete dismissed threats")

    sub.add_parser("report", help="Generate 7-day report")
    sub.add_parser("update-defs", help="Check for definition updates")

    return parser


def main(argv: Optional[List[str]] = None) -> int:
    """CLI entrypoint."""
    parser = build_parser()
    args = parser.parse_args(argv)

    cfg = load_config()
    db_path = resolve_db_path(cfg)
    conn = connect_db(db_path)

    try:
        if args.command == "status":
            print_status(cfg, conn, args.json)
        elif args.command == "disable":
            cmd_disable(cfg, args.until, args.json)
        elif args.command == "enable":
            cmd_enable(cfg, args.json)
        elif args.command == "bypass":
            cmd_bypass(cfg, args.on, args.json)
        elif args.command == "dismiss":
            cmd_dismiss(conn, args.sig_id, args.json)
        elif args.command == "allowlist":
            cmd_allowlist(cfg, args.allow_cmd, args.pattern, args.json)
        elif args.command == "threats":
            cmd_threats(conn, args.clear, args.json)
        elif args.command == "report":
            cmd_report(conn, args.json)
        elif args.command == "update-defs":
            cmd_update_defs(args.json)
        return 0
    except ValueError as exc:
        if args.json:
            render_json({"ok": False, "error": str(exc)})
        else:
            print(colorize(f"Error: {exc}", RED), file=sys.stderr)
        return 2
    finally:
        conn.close()


if __name__ == "__main__":
    raise SystemExit(main())
