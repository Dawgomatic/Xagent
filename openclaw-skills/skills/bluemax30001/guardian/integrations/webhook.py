#!/usr/bin/env python3
"""Webhook notifier for Guardian threat events."""

from __future__ import annotations

import json
import urllib.request
from dataclasses import asdict, is_dataclass
from typing import Any, Dict, Optional


class GuardianWebhook:
    """Send threat notifications to HTTP webhook endpoints."""

    def __init__(self, url: str, timeout_seconds: float = 3.0) -> None:
        self.url = url
        self.timeout_seconds = timeout_seconds

    def notify(self, result: Any, extra: Optional[Dict[str, Any]] = None) -> int:
        """POST scan result to configured webhook URL and return status code."""
        payload: Dict[str, Any]
        if is_dataclass(result):
            payload = asdict(result)
        elif hasattr(result, "to_dict"):
            payload = result.to_dict()
        elif isinstance(result, dict):
            payload = result
        else:
            raise TypeError("result must be dataclass, dict, or expose to_dict()")

        body = {
            "event": "guardian.threat.detected" if payload.get("threats") else "guardian.scan.clean",
            "result": payload,
        }
        if extra:
            body["meta"] = extra

        raw = json.dumps(body).encode("utf-8")
        req = urllib.request.Request(
            self.url,
            data=raw,
            method="POST",
            headers={"Content-Type": "application/json", "User-Agent": "guardian/2.0"},
        )
        with urllib.request.urlopen(req, timeout=self.timeout_seconds) as resp:
            return int(resp.status)
