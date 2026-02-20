#!/usr/bin/env python3
"""Generic CLI wrapper for any Pylon endpoint."""
from __future__ import annotations

import argparse
import json
from typing import Any, Dict, Optional

from pylon_client import api_request


def _parse_kv_pairs(items: Optional[list[str]]) -> Dict[str, Any]:
    result: Dict[str, Any] = {}
    if not items:
        return result
    for item in items:
        if "=" not in item:
            raise SystemExit(f"Invalid key=value pair: {item}")
        key, value = item.split("=", 1)
        result[key] = value
    return result


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Send an arbitrary request to the Pylon API."
    )
    parser.add_argument("path", help="Endpoint path, e.g. /issues or /issues/{id}")
    parser.add_argument(
        "--method", default="GET", help="HTTP method (GET, POST, PATCH, DELETE, ... )"
    )
    parser.add_argument(
        "--param",
        action="append",
        dest="params",
        help="Query parameter as key=value. Repeat for multiple params.",
    )
    parser.add_argument(
        "--data",
        help="Inline JSON payload string (e.g. '{\"state\":\"waiting_on_you\"}')",
    )
    parser.add_argument(
        "--data-file",
        help="Path to JSON file to send as the request body (overrides --data)",
    )
    args = parser.parse_args()

    params = _parse_kv_pairs(args.params)
    body: Optional[Dict[str, Any]] = None
    if args.data_file:
        with open(args.data_file, "r", encoding="utf-8") as infile:
            body = json.load(infile)
    elif args.data:
        body = json.loads(args.data)

    resp = api_request(args.path, method=args.method, params=params, data=body)
    print(json.dumps(resp, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
