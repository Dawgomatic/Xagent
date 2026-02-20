#!/usr/bin/env python3
"""Convenience wrapper to list issues with common filters."""
from __future__ import annotations

import argparse
import json
from datetime import datetime, timedelta, timezone
from typing import Dict

from pylon_client import api_request


DEFAULT_WINDOW_DAYS = 30


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="List Pylon issues")
    parser.add_argument(
        "--state", help="Filter by state slug (e.g., new, waiting_on_you, closed)"
    )
    parser.add_argument("--team-id", help="Filter by team ID")
    parser.add_argument("--assignee-id", help="Filter by assignee user ID")
    parser.add_argument("--requester-id", help="Filter by requester/contact ID")
    parser.add_argument(
        "--limit",
        type=int,
        default=50,
        help="Max issues to fetch (page size). Default: 50",
    )
    parser.add_argument(
        "--start-time",
        help="ISO8601 start_time filter (default: now minus --window-days)",
    )
    parser.add_argument(
        "--end-time",
        help="ISO8601 end_time filter (default: current time)",
    )
    parser.add_argument(
        "--window-days",
        type=int,
        default=DEFAULT_WINDOW_DAYS,
        help="Window size (days) used when start/end missing. Default: 30",
    )
    parser.add_argument(
        "--page-cursor",
        help="Pagination cursor from a previous response (optional)",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    params: Dict[str, object] = {"limit": str(args.limit)}
    if args.state:
        params["state"] = args.state
    if args.team_id:
        params["team_id"] = args.team_id
    if args.assignee_id:
        params["assignee_id"] = args.assignee_id
    if args.requester_id:
        params["requester_id"] = args.requester_id
    if args.page_cursor:
        params["cursor"] = args.page_cursor

    if args.start_time:
        params["start_time"] = args.start_time
    if args.end_time:
        params["end_time"] = args.end_time

    # Some tenants require explicit start/end filters. Provide sensible defaults when absent.
    if "start_time" not in params or "end_time" not in params:
        now = datetime.now(timezone.utc)
        default_end = now.isoformat(timespec="seconds")
        default_start = (now - timedelta(days=args.window_days)).isoformat(timespec="seconds")
        params.setdefault("end_time", default_end)
        params.setdefault("start_time", default_start)

    resp = api_request("/issues", params=params)
    print(json.dumps(resp, indent=2))
    if resp.get("pagination", {}).get("has_next_page"):
        cursor = resp["pagination"].get("cursor")
        if cursor:
            print(f"\nNext page cursor: {cursor}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
