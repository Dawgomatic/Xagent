#!/usr/bin/env python3
"""Guardian standalone HTTP API server (stdlib only)."""

from __future__ import annotations

import argparse
import json
import sys
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Dict, Tuple
from urllib.parse import parse_qs, urlparse

SCRIPT_DIR = Path(__file__).resolve().parent
SKILL_ROOT = SCRIPT_DIR.parent
if str(SKILL_ROOT) not in sys.path:
    sys.path.insert(0, str(SKILL_ROOT))

from core.api import GuardianScanner


def build_status_payload(scanner: GuardianScanner) -> Dict[str, Any]:
    """Build status payload from DB summary and health estimate."""
    db = scanner._scanner.db
    summary = db.get_threat_summary(hours=24) if db else {"total": 0, "blocked": 0, "categories": {}}
    health = max(0, 100 - min(100, int(summary.get("blocked", 0) * 5)))
    return {
        "ok": True,
        "health_score": health,
        "threats_24h": summary.get("total", 0),
        "blocked_24h": summary.get("blocked", 0),
        "categories": summary.get("categories", {}),
    }


def handle_scan_payload(scanner: GuardianScanner, data: Dict[str, Any]) -> Tuple[int, Dict[str, Any]]:
    """Validate and process /scan payload."""
    text = str(data.get("text", ""))
    channel = str(data.get("channel", "api"))
    if not text.strip():
        return HTTPStatus.BAD_REQUEST, {"error": "text is required"}

    result = scanner.scan(text=text, channel=channel)
    status = HTTPStatus.FORBIDDEN if result.blocked else HTTPStatus.OK
    return status, result.to_dict()


def handle_dismiss_payload(scanner: GuardianScanner, data: Dict[str, Any]) -> Tuple[int, Dict[str, Any]]:
    """Validate and process /dismiss payload."""
    raw_id = data.get("id")
    if raw_id is None:
        return HTTPStatus.BAD_REQUEST, {"error": "id is required"}

    try:
        threat_id = int(raw_id)
    except (TypeError, ValueError):
        return HTTPStatus.BAD_REQUEST, {"error": "id must be an integer"}

    db = scanner._scanner.db
    if not db:
        return HTTPStatus.BAD_REQUEST, {"error": "DB persistence disabled"}

    db.dismiss_threat(threat_id)
    return HTTPStatus.OK, {"ok": True, "dismissed": threat_id}


def list_threats_payload(scanner: GuardianScanner, query: str) -> Dict[str, Any]:
    """Return filtered threats payload for /threats route."""
    db = scanner._scanner.db
    if not db:
        return {"threats": []}

    qs = parse_qs(query)
    hours = int((qs.get("hours", ["24"]) or ["24"])[0])
    limit = int((qs.get("limit", ["50"]) or ["50"])[0])
    rows = db.get_threats(hours=hours, limit=limit)

    channel = (qs.get("channel", [None]) or [None])[0]
    category = (qs.get("category", [None]) or [None])[0]
    if channel:
        rows = [row for row in rows if row.get("channel") == channel]
    if category:
        rows = [row for row in rows if row.get("category") == category]
    return {"threats": rows, "count": len(rows)}


class GuardianHTTPHandler(BaseHTTPRequestHandler):
    """HTTP handler exposing scan/status/health/dismiss/threat endpoints."""

    scanner: GuardianScanner

    def log_message(self, format: str, *args: Any) -> None:  # noqa: A003
        return

    def _json_response(self, status: int, payload: Dict[str, Any]) -> None:
        raw = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def _read_json_body(self) -> Tuple[Dict[str, Any], str]:
        length = int(self.headers.get("Content-Length", "0") or 0)
        if length <= 0:
            return {}, "Request body is required"
        try:
            data = json.loads(self.rfile.read(length).decode("utf-8"))
        except json.JSONDecodeError:
            return {}, "Request body must be valid JSON"
        if not isinstance(data, dict):
            return {}, "JSON body must be an object"
        return data, ""

    def do_GET(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        if parsed.path == "/health":
            self._json_response(HTTPStatus.OK, {"ok": True})
            return

        if parsed.path == "/status":
            self._json_response(HTTPStatus.OK, build_status_payload(self.scanner))
            return

        if parsed.path == "/threats":
            self._json_response(HTTPStatus.OK, list_threats_payload(self.scanner, parsed.query))
            return

        self._json_response(HTTPStatus.NOT_FOUND, {"error": "Not found"})

    def do_POST(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)

        if parsed.path == "/scan":
            data, err = self._read_json_body()
            if err:
                self._json_response(HTTPStatus.BAD_REQUEST, {"error": err})
                return

            status, payload = handle_scan_payload(self.scanner, data)
            self._json_response(status, payload)
            return

        if parsed.path == "/dismiss":
            data, err = self._read_json_body()
            if err:
                self._json_response(HTTPStatus.BAD_REQUEST, {"error": err})
                return

            status, payload = handle_dismiss_payload(self.scanner, data)
            self._json_response(status, payload)
            return

        self._json_response(HTTPStatus.NOT_FOUND, {"error": "Not found"})


def create_server(
    host: str = "127.0.0.1",
    port: int = 8080,
    severity: str = "medium",
    db_path: str | None = None,
    server_class: type[ThreadingHTTPServer] = ThreadingHTTPServer,
) -> ThreadingHTTPServer:
    """Create configured Guardian HTTP server instance."""
    effective_db = db_path or str((Path.cwd() / "guardian.db").resolve())
    scanner = GuardianScanner(severity=severity, db_path=effective_db, record_to_db=True)

    class _ConfiguredHandler(GuardianHTTPHandler):
        pass

    _ConfiguredHandler.scanner = scanner
    server = server_class((host, port), _ConfiguredHandler)
    return server


def main() -> None:
    """CLI entrypoint for guardian-serve."""
    parser = argparse.ArgumentParser(description="Guardian HTTP API server")
    parser.add_argument("--host", default="127.0.0.1", help="Bind host")
    parser.add_argument("--port", type=int, default=8080, help="Bind port")
    parser.add_argument("--severity", default="medium", help="low|medium|high|critical")
    parser.add_argument("--db", dest="db_path", help="SQLite DB path")
    args = parser.parse_args()

    try:
        server = create_server(host=args.host, port=args.port, severity=args.severity, db_path=args.db_path)
    except PermissionError as exc:
        raise SystemExit(f"Unable to bind HTTP server on {args.host}:{args.port}: {exc}") from exc
    print(f"Guardian server listening on http://{args.host}:{args.port}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.shutdown()
        server.server_close()
        server.RequestHandlerClass.scanner.close()


if __name__ == "__main__":
    main()
