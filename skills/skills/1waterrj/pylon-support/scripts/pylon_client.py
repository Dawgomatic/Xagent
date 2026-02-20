#!/usr/bin/env python3
"""Lightweight Pylon API helper shared by the other scripts."""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, Optional

BASE_URL = os.environ.get("PYLON_API_BASE", "https://api.usepylon.com")
TOKEN_ENV_VAR = "PYLON_API_TOKEN"


def _require_token() -> str:
    token = os.environ.get(TOKEN_ENV_VAR)
    if not token:
        raise RuntimeError(
            f"Set the {TOKEN_ENV_VAR} environment variable to a Pylon API token."
        )
    return token


def _build_url(path: str, params: Optional[Dict[str, Any]] = None) -> str:
    path = path if path.startswith("/") else f"/{path}"
    url = BASE_URL.rstrip("/") + path
    if params:
        encoded = urllib.parse.urlencode(params, doseq=True)
        url = f"{url}?{encoded}"
    return url


def api_request(
    path: str,
    *,
    method: str = "GET",
    params: Optional[Dict[str, Any]] = None,
    data: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    """Execute an HTTP request against the Pylon API and return JSON."""
    token = _require_token()
    url = _build_url(path, params)
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/json",
    }
    body: Optional[bytes] = None
    if data is not None:
        headers["Content-Type"] = "application/json"
        body = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(url, data=body, method=method.upper(), headers=headers)
    try:
        with urllib.request.urlopen(req) as resp:
            payload = resp.read().decode("utf-8").strip()
            return json.loads(payload) if payload else {}
    except urllib.error.HTTPError as err:  # pragma: no cover - runtime guard
        details = err.read().decode("utf-8")
        print(
            f"Pylon API request failed ({err.code} {err.reason})\n{details}",
            file=sys.stderr,
        )
        raise
