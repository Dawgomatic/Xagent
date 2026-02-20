#!/usr/bin/env python3
"""
Intelligent Router - Spawn Helper (Enforced Core Skill)

MANDATORY: Call this before ANY sessions_spawn or cron job creation.
It classifies the task and outputs the exact model to use.

Usage (show command):
    python3 skills/intelligent-router/scripts/spawn_helper.py "task description"

Usage (just get model id):
    python3 skills/intelligent-router/scripts/spawn_helper.py --model-only "task description"

Usage (validate payload has model set):
    python3 skills/intelligent-router/scripts/spawn_helper.py --validate '{"kind":"agentTurn","message":"..."}'
"""

import sys
import json
import subprocess
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CONFIG_FILE = SCRIPT_DIR.parent / "config.json"

TIER_COLORS = {
    "SIMPLE": "🟢",
    "MEDIUM": "🟡",
    "COMPLEX": "🟠",
    "REASONING": "🔵",
    "CRITICAL": "🔴",
}


def load_config():
    if not CONFIG_FILE.exists():
        raise FileNotFoundError(f"Router config not found: {CONFIG_FILE}")
    with open(CONFIG_FILE) as f:
        return json.load(f)


def classify_task(task_description):
    """Run router.py classify and return (tier, model_id, confidence)."""
    result = subprocess.run(
        [sys.executable, str(SCRIPT_DIR / "router.py"), "classify", task_description],
        capture_output=True, text=True, check=True
    )
    lines = result.stdout.strip().split('\n')
    tier = None
    model_id = None
    confidence = None

    for line in lines:
        if line.startswith("Classification:"):
            tier = line.split(":", 1)[1].strip()
        elif "  ID:" in line:
            model_id = line.split(":", 1)[1].strip()
        elif line.startswith("Confidence:"):
            confidence = line.split(":", 1)[1].strip()

    return tier, model_id, confidence


def validate_payload(payload_json):
    """
    Validate a cron job payload has the model field set.
    Returns (ok: bool, message: str)
    """
    try:
        payload = json.loads(payload_json) if isinstance(payload_json, str) else payload_json
    except json.JSONDecodeError as e:
        return False, f"Invalid JSON payload: {e}"

    if payload.get("kind") != "agentTurn":
        return True, "Non-agentTurn payload — model not required"

    model = payload.get("model")
    if not model:
        return False, (
            "❌ VIOLATION: agentTurn payload missing 'model' field!\n"
            "   Without model, OpenClaw defaults to Sonnet = expensive waste.\n"
            "   Fix: add \"model\": \"<model-id>\" to the payload.\n"
            "   Run: python3 skills/intelligent-router/scripts/spawn_helper.py \"<task>\" to get the right model."
        )

    # Check if Sonnet/Opus is used for a non-critical payload
    expensive = ["claude-sonnet", "claude-opus", "claude-3"]
    for keyword in expensive:
        if keyword in model.lower():
            msg = payload.get("message", "")[:80]
            return None, (
                f"⚠️  WARNING: Expensive model '{model}' set for potentially simple task.\n"
                f"   Task preview: {msg}...\n"
                f"   Consider: python3 skills/intelligent-router/scripts/spawn_helper.py \"{msg}\""
            )

    return True, f"✅ Model set: {model}"


def main():
    args = sys.argv[1:]

    if not args:
        print(__doc__)
        sys.exit(1)

    # --validate mode
    if args[0] == "--validate":
        if len(args) < 2:
            print("Usage: spawn_helper.py --validate '<payload_json>'")
            sys.exit(1)
        ok, msg = validate_payload(args[1])
        print(msg)
        sys.exit(0 if ok else 1)

    # --model-only mode (just print the model id)
    if args[0] == "--model-only":
        if len(args) < 2:
            print("Usage: spawn_helper.py --model-only 'task description'")
            sys.exit(1)
        task = " ".join(args[1:])
        config = load_config()
        tier, model_id, _ = classify_task(task)
        if not model_id:
            rules = config.get("routing_rules", {}).get(tier, {})
            model_id = rules.get("primary", "anthropic-proxy-4/glm-4.7")
        print(model_id)
        sys.exit(0)

    # Default: classify and show spawn command
    task = " ".join(args)
    config = load_config()
    tier, model_id, confidence = classify_task(task)

    if not model_id:
        rules = config.get("routing_rules", {}).get(tier, {})
        model_id = rules.get("primary", "anthropic-proxy-4/glm-4.7")

    icon = TIER_COLORS.get(tier, "⚪")
    fallback_chain = config.get("routing_rules", {}).get(tier, {}).get("fallback_chain", [])

    print(f"\n{icon} Task classified as: {tier} (confidence: {confidence})")
    print(f"💰 Recommended model: {model_id}")
    if fallback_chain:
        print(f"🔄 Fallbacks: {' → '.join(fallback_chain[:2])}")
    print(f"\n📋 Use in sessions_spawn:")
    print(f"""   sessions_spawn(
       task=\"{task[:60]}{'...' if len(task)>60 else ''}\",
       model=\"{model_id}\",
       label=\"<label>\"
   )""")
    print(f"\n📋 Use in cron job payload:")
    print(f"""   {{
       "kind": "agentTurn",
       "message": "...",
       "model": "{model_id}"
   }}""")


if __name__ == "__main__":
    main()
