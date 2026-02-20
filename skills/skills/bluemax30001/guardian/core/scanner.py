#!/usr/bin/env python3
"""Guardian reusable scanner for text and session artifacts."""

from __future__ import annotations

import hashlib
import json
import re
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional

try:
    from .guardian_db import GuardianDB
    from .settings import definitions_dir, load_config
except ImportError:
    from guardian_db import GuardianDB
    from settings import definitions_dir, load_config


class GuardianScanner:
    """Stateless scanner for threat pattern matching and optional DB recording."""

    def __init__(
        self,
        record_to_db: bool = True,
        config_path: Optional[str] = None,
        db_path: Optional[str] = None,
    ) -> None:
        self.config = load_config(config_path)
        self.patterns = self._load_patterns()
        self.threshold_score = self._threshold_score()
        self.db = GuardianDB(db_path=db_path) if record_to_db else None

    def _threshold_score(self) -> int:
        """Resolve numeric threshold score from config severity label."""
        severity = str(self.config.get("severity_threshold", "medium")).lower()
        mapping = {"low": 0, "medium": 50, "high": 80, "critical": 90}
        return mapping.get(severity, 50)

    def _load_patterns(self) -> List[Dict[str, Any]]:
        """Load all definition files and compile regex patterns."""
        compiled: List[Dict[str, Any]] = []
        defs_dir = definitions_dir(self.config)
        def_files = [
            "injection-sigs.json",
            "exfil-patterns.json",
            "tool-abuse.json",
            "social-engineering.json",
        ]

        for fname in def_files:
            fpath = defs_dir / fname
            if not fpath.exists():
                continue
            try:
                data = json.loads(fpath.read_text(encoding="utf-8"))
            except (json.JSONDecodeError, OSError):
                continue

            items = data if isinstance(data, list) else data.get("signatures", data.get("patterns", []))
            for sig in items:
                pats: List[str] = []
                det = sig.get("detection", {})
                if isinstance(det, dict):
                    pats = det.get("patterns", [])
                if not pats and isinstance(sig.get("patterns"), list):
                    pats = sig["patterns"]
                if not pats and sig.get("pattern"):
                    pats = [sig["pattern"]]

                category = sig.get("category", "")
                if not category:
                    sid = str(sig.get("id", ""))
                    if sid.startswith("INJ"):
                        category = "prompt_injection"
                    elif sid.startswith("EXF"):
                        category = "data_exfiltration"
                    elif sid.startswith("TAB"):
                        category = "tool_abuse"
                    elif sid.startswith("SOC"):
                        category = "social_engineering"
                    else:
                        category = "unknown"

                for pattern in pats:
                    try:
                        compiled.append(
                            {
                                "regex": re.compile(pattern, re.IGNORECASE),
                                "id": sig.get("id", "?"),
                                "severity": sig.get("severity", "low"),
                                "category": category,
                                "description": sig.get("description", ""),
                                "score": int(sig.get("score", 50)),
                            }
                        )
                    except re.error:
                        continue
        return compiled

    def scan(self, text: str, channel: str = "unknown") -> Dict[str, Any]:
        """Scan text and return a structured threat result without DB writes."""
        if not text or len(text) < 3:
            return {"clean": True, "score": 0, "threats": [], "channel": channel, "blocked": False}

        hits: List[Dict[str, Any]] = []
        for pattern_obj in self.patterns:
            match = pattern_obj["regex"].search(text)
            if not match:
                continue
            hits.append(
                {
                    "id": pattern_obj["id"],
                    "category": pattern_obj["category"],
                    "severity": pattern_obj["severity"],
                    "score": pattern_obj["score"],
                    "evidence": match.group(0)[:80],
                    "description": pattern_obj["description"],
                }
            )

        if not hits:
            return {"clean": True, "score": 0, "threats": [], "channel": channel, "blocked": False}

        best = max(hits, key=lambda hit: hit["score"])
        blocked = bool(best["score"] >= self.threshold_score)
        return {
            "clean": False,
            "score": best["score"],
            "blocked": blocked,
            "threats": hits,
            "top_threat": best,
            "channel": channel,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

    def scan_and_record(self, text: str, channel: str = "unknown", source_id: str = "") -> Dict[str, Any]:
        """Scan text and persist top threat to DB when a detection occurs."""
        result = self.scan(text, channel)
        if result["clean"] or not self.db:
            return result

        top = result["top_threat"]
        msg_hash = hashlib.sha256(text.encode("utf-8", errors="replace")).hexdigest()
        self.db.add_threat(
            sig_id=top["id"],
            category=top["category"],
            severity=top["severity"],
            score=top["score"],
            evidence=top["evidence"],
            description=top["description"],
            blocked=result["blocked"],
            channel=channel,
            source_file=f"{channel}:{source_id}",
            message_hash=msg_hash,
        )
        return result

    def scan_batch(self, items: List[Dict[str, str]]) -> List[Dict[str, Any]]:
        """Scan and optionally record a batch of message dictionaries."""
        return [
            self.scan_and_record(
                item.get("text", ""),
                channel=item.get("channel", "unknown"),
                source_id=item.get("source_id", ""),
            )
            for item in items
        ]

    def close(self) -> None:
        """Close DB connection if scanner was initialized with persistence."""
        if self.db:
            self.db.close()


def quick_scan(text: str, channel: str = "unknown", config_path: Optional[str] = None) -> Dict[str, Any]:
    """One-shot scan without DB recording."""
    scanner = GuardianScanner(record_to_db=False, config_path=config_path)
    return scanner.scan(text, channel)


def scan_and_record(
    text: str,
    channel: str = "unknown",
    source_id: str = "",
    config_path: Optional[str] = None,
    db_path: Optional[str] = None,
) -> Dict[str, Any]:
    """One-shot scan with DB recording enabled."""
    scanner = GuardianScanner(record_to_db=True, config_path=config_path, db_path=db_path)
    try:
        return scanner.scan_and_record(text, channel, source_id)
    finally:
        scanner.close()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Guardian one-shot scanner")
    parser.add_argument("text", help="Text to scan")
    parser.add_argument("channel", nargs="?", default="cli", help="Channel context")
    parser.add_argument("--config", dest="config_path", help="Path to config JSON")
    args = parser.parse_args()

    result = quick_scan(args.text, channel=args.channel, config_path=args.config_path)
    print(json.dumps(result, indent=2, default=str))
