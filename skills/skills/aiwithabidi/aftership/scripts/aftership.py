#!/usr/bin/env python3
"""AfterShip CLI — AfterShip — package tracking, delivery notifications, estimated delivery dates, and courier detection.

Zero dependencies beyond Python stdlib.
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
import urllib.parse

API_BASE = "https://api.aftership.com/v4"


def get_env(name):
    val = os.environ.get(name, "")
    if not val:
        env_path = os.path.join(os.environ.get("WORKSPACE", os.path.expanduser("~/.openclaw/workspace")), ".env")
        if os.path.exists(env_path):
            with open(env_path) as f:
                for line in f:
                    line = line.strip()
                    if line.startswith(name + "="):
                        val = line.split("=", 1)[1].strip().strip('"').strip("'")
                        break
    return val


def req(method, url, data=None, headers=None, timeout=30):
    body = json.dumps(data).encode() if data else None
    r = urllib.request.Request(url, data=body, method=method)
    r.add_header("Content-Type", "application/json")
    if headers:
        for k, v in headers.items():
            r.add_header(k, v)
    try:
        resp = urllib.request.urlopen(r, timeout=timeout)
        raw = resp.read().decode()
        return json.loads(raw) if raw.strip() else {}
    except urllib.error.HTTPError as e:
        err = e.read().decode()
        print(json.dumps({"error": True, "code": e.code, "message": err}), file=sys.stderr)
        sys.exit(1)


def api(method, path, data=None, params=None):
    """Make authenticated API request."""
    base = API_BASE
    token = get_env("AFTERSHIP_API_KEY")
    if not token:
        print("Error: AFTERSHIP_API_KEY not set", file=sys.stderr)
        sys.exit(1)
    headers = {"Authorization": f"Bearer {token}"}
    url = f"{base}{path}"
    if params:
        qs = urllib.parse.urlencode({k: v for k, v in params.items() if v}, doseq=True)
        url = f"{url}{'&' if '?' in url else '?'}{qs}"
    return req(method, url, data=data, headers=headers)


def out(data):
    print(json.dumps(data, indent=2, default=str))


def cmd_list_trackings(args):
    """List all trackings"""
    path = "/trackings"
    params = {}
    if args.page:
        params["page"] = args.page
    if args.limit:
        params["limit"] = args.limit
    if args.keyword:
        params["keyword"] = args.keyword
    result = api("GET", path, params=params)
    out(result)

def cmd_get_tracking(args):
    """Get tracking details"""
    path = "/trackings/{slug}/{tracking_number}"
    path = path.replace("{slug}", str(args.slug or ""))
    path = path.replace("{tracking-number}", str(args.tracking_number or ""))
    params = {}
    if args.tracking_number:
        params["tracking-number"] = args.tracking_number
    result = api("GET", path, params=params)
    out(result)

def cmd_create_tracking(args):
    """Create tracking"""
    path = "/trackings"
    data = {}
    if args.tracking_number:
        data["tracking-number"] = args.tracking_number
    if args.slug:
        data["slug"] = args.slug
    if args.title:
        data["title"] = args.title
    if args.emails:
        data["emails"] = args.emails
    if args.phones:
        data["phones"] = args.phones
    result = api("POST", path, data=data)
    out(result)

def cmd_delete_tracking(args):
    """Delete tracking"""
    path = "/trackings/{slug}/{tracking_number}"
    path = path.replace("{slug}", str(args.slug or ""))
    path = path.replace("{tracking-number}", str(args.tracking_number or ""))
    params = {}
    if args.tracking_number:
        params["tracking-number"] = args.tracking_number
    result = api("DELETE", path, params=params)
    out(result)

def cmd_retrack(args):
    """Retrack expired tracking"""
    path = "/trackings/{slug}/{tracking_number}/retrack"
    path = path.replace("{slug}", str(args.slug or ""))
    path = path.replace("{tracking-number}", str(args.tracking_number or ""))
    data = {}
    if args.tracking_number:
        data["tracking-number"] = args.tracking_number
    result = api("POST", path, data=data)
    out(result)

def cmd_detect_courier(args):
    """Detect courier for tracking number"""
    path = "/couriers/detect"
    data = {}
    if args.tracking_number:
        data["tracking-number"] = args.tracking_number
    result = api("POST", path, data=data)
    out(result)

def cmd_list_couriers(args):
    """List all supported couriers"""
    path = "/couriers/all"
    result = api("GET", path)
    out(result)

def cmd_get_last_checkpoint(args):
    """Get last checkpoint"""
    path = "/last_checkpoint/{slug}/{tracking_number}"
    path = path.replace("{slug}", str(args.slug or ""))
    path = path.replace("{tracking-number}", str(args.tracking_number or ""))
    params = {}
    if args.tracking_number:
        params["tracking-number"] = args.tracking_number
    result = api("GET", path, params=params)
    out(result)

def cmd_list_notifications(args):
    """Get notification settings"""
    path = "/notifications/{slug}/{tracking_number}"
    path = path.replace("{slug}", str(args.slug or ""))
    path = path.replace("{tracking-number}", str(args.tracking_number or ""))
    params = {}
    if args.tracking_number:
        params["tracking-number"] = args.tracking_number
    result = api("GET", path, params=params)
    out(result)


def main():
    parser = argparse.ArgumentParser(description="AfterShip CLI")
    sub = parser.add_subparsers(dest="command")
    sub.required = True

    p_list_trackings = sub.add_parser("list-trackings", help="List all trackings")
    p_list_trackings.add_argument("--page", default="1")
    p_list_trackings.add_argument("--limit", default="50")
    p_list_trackings.add_argument("--keyword", required=True)
    p_list_trackings.set_defaults(func=cmd_list_trackings)

    p_get_tracking = sub.add_parser("get-tracking", help="Get tracking details")
    p_get_tracking.add_argument("--slug", required=True)
    p_get_tracking.add_argument("--tracking-number", required=True)
    p_get_tracking.set_defaults(func=cmd_get_tracking)

    p_create_tracking = sub.add_parser("create-tracking", help="Create tracking")
    p_create_tracking.add_argument("--tracking-number", required=True)
    p_create_tracking.add_argument("--slug", required=True)
    p_create_tracking.add_argument("--title", required=True)
    p_create_tracking.add_argument("--emails", required=True)
    p_create_tracking.add_argument("--phones", required=True)
    p_create_tracking.set_defaults(func=cmd_create_tracking)

    p_delete_tracking = sub.add_parser("delete-tracking", help="Delete tracking")
    p_delete_tracking.add_argument("--slug", required=True)
    p_delete_tracking.add_argument("--tracking-number", required=True)
    p_delete_tracking.set_defaults(func=cmd_delete_tracking)

    p_retrack = sub.add_parser("retrack", help="Retrack expired tracking")
    p_retrack.add_argument("--slug", required=True)
    p_retrack.add_argument("--tracking-number", required=True)
    p_retrack.set_defaults(func=cmd_retrack)

    p_detect_courier = sub.add_parser("detect-courier", help="Detect courier for tracking number")
    p_detect_courier.add_argument("--tracking-number", required=True)
    p_detect_courier.set_defaults(func=cmd_detect_courier)

    p_list_couriers = sub.add_parser("list-couriers", help="List all supported couriers")
    p_list_couriers.set_defaults(func=cmd_list_couriers)

    p_get_last_checkpoint = sub.add_parser("get-last-checkpoint", help="Get last checkpoint")
    p_get_last_checkpoint.add_argument("--slug", required=True)
    p_get_last_checkpoint.add_argument("--tracking-number", required=True)
    p_get_last_checkpoint.set_defaults(func=cmd_get_last_checkpoint)

    p_list_notifications = sub.add_parser("list-notifications", help="Get notification settings")
    p_list_notifications.add_argument("--slug", required=True)
    p_list_notifications.add_argument("--tracking-number", required=True)
    p_list_notifications.set_defaults(func=cmd_list_notifications)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
