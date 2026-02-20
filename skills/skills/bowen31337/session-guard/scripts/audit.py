#!/usr/bin/env python3
"""
session-guard: Audit OpenClaw config and session health.
Detects heartbeat-in-main-session antipattern and session file bloat.

Usage:
  python3 audit.py
  python3 audit.py --config /path/to/openclaw.json
  python3 audit.py --sessions-dir /path/to/sessions
  python3 audit.py --warn-mb 3  # alert threshold (default: 5MB)
"""
import json
import os
import sys
import glob
import argparse
from pathlib import Path

DEFAULT_CONFIG = os.path.expanduser("~/.openclaw/openclaw.json")
DEFAULT_SESSIONS_DIR = os.path.expanduser("~/.openclaw/agents/main/sessions")
WARN_MB = 5
CRIT_MB = 10

def parse_args():
    p = argparse.ArgumentParser()
    p.add_argument("--config", default=DEFAULT_CONFIG)
    p.add_argument("--sessions-dir", default=DEFAULT_SESSIONS_DIR)
    p.add_argument("--warn-mb", type=float, default=WARN_MB)
    p.add_argument("--json", action="store_true", help="Output as JSON")
    return p.parse_args()

def check_config(config_path):
    issues = []
    warnings = []
    try:
        with open(config_path) as f:
            cfg = json.load(f)
    except FileNotFoundError:
        issues.append(f"Config not found: {config_path}")
        return issues, warnings
    except json.JSONDecodeError as e:
        issues.append(f"Config JSON invalid: {e}")
        return issues, warnings

    defaults = cfg.get("agents", {}).get("defaults", {})

    # Check heartbeat
    hb = defaults.get("heartbeat", {})
    hb_every = hb.get("every", "")
    if hb_every and hb_every != "0m" and hb_every != "0":
        issues.append(
            f"HEARTBEAT IN MAIN SESSION: heartbeat.every='{hb_every}' "
            f"causes session bloat. Set to '0m' and use isolated cron instead."
        )

    # Check compaction
    comp = defaults.get("compaction", {})
    comp_mode = comp.get("mode", "")
    if comp_mode == "safeguard":
        warnings.append(
            "compaction.mode='safeguard' only triggers near 200k token limit. "
            "Switch to 'default' for more proactive compaction."
        )

    return issues, warnings

def check_sessions(sessions_dir, warn_mb):
    issues = []
    warnings = []
    if not os.path.isdir(sessions_dir):
        warnings.append(f"Sessions dir not found: {sessions_dir}")
        return issues, warnings

    pattern = os.path.join(sessions_dir, "*.jsonl")
    files = glob.glob(pattern)
    active = [f for f in files if ".reset." not in f and ".deleted." not in f]
    
    if not active:
        warnings.append("No active session files found.")
        return issues, warnings

    results = []
    for f in active:
        size_bytes = os.path.getsize(f)
        size_mb = size_bytes / (1024 * 1024)
        name = os.path.basename(f)
        results.append((size_mb, name, f))

    results.sort(reverse=True)
    
    for size_mb, name, path in results:
        if size_mb >= CRIT_MB:
            issues.append(
                f"CRITICAL BLOAT: {name} is {size_mb:.1f}MB "
                f"(>{CRIT_MB}MB). Session may corrupt soon."
            )
        elif size_mb >= warn_mb:
            warnings.append(
                f"SESSION BLOAT WARNING: {name} is {size_mb:.1f}MB "
                f"(>{warn_mb}MB). Monitor for growth."
            )

    return issues, warnings, results

def main():
    args = parse_args()
    
    config_issues, config_warnings = check_config(args.config)
    session_result = check_sessions(args.sessions_dir, args.warn_mb)
    session_issues, session_warnings = session_result[0], session_result[1]
    session_files = session_result[2] if len(session_result) > 2 else []

    all_issues = config_issues + session_issues
    all_warnings = config_warnings + session_warnings

    if args.json:
        print(json.dumps({
            "issues": all_issues,
            "warnings": all_warnings,
            "session_files": [
                {"name": n, "size_mb": round(s, 2)} for s, n, _ in session_files
            ],
            "healthy": len(all_issues) == 0
        }, indent=2))
        return

    if not all_issues and not all_warnings:
        print("✅ Session guard: No issues found.")
        return

    print("=== Session Guard Audit ===\n")

    if all_issues:
        print("🚨 ISSUES (action required):")
        for i in all_issues:
            print(f"  ❌ {i}")
        print()

    if all_warnings:
        print("⚠️  WARNINGS:")
        for w in all_warnings:
            print(f"  ⚠️  {w}")
        print()

    if session_files:
        print("📊 Active session sizes:")
        for size_mb, name, _ in session_files[:5]:
            bar = "█" * min(int(size_mb), 20)
            print(f"  {size_mb:6.1f}MB  {bar}  {name[:40]}")
        print()

    if all_issues:
        print("💡 Fix: Apply config patch via gateway tool:")
        print('  agents.defaults.heartbeat.every = "0m"')
        print('  agents.defaults.compaction.mode = "default"')
        print("  Then create isolated cron heartbeat (see session-guard SKILL.md).")
        sys.exit(1)

if __name__ == "__main__":
    main()
