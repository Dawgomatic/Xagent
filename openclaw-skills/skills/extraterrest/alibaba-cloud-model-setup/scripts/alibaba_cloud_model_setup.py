#!/usr/bin/env python3
"""Interactive OpenClaw Alibaba Cloud provider configurator.

Safely updates OpenClaw-style JSON config with a `balian` provider entry.
"""

from __future__ import annotations

import argparse
import datetime as dt
import getpass
import json
import os
import re
import shutil
import subprocess
from pathlib import Path
from typing import Any, Dict
from urllib import error, request

CN_BASE_URL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
INTL_BASE_URL = "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
US_BASE_URL = "https://dashscope-us.aliyuncs.com/compatible-mode/v1"
DEFAULT_MODEL = "qwen3-coder-plus"
DEFAULT_PRESET_MODELS = ["qwen-max", "qwen-flash", "qwen3-coder-plus"]
PROVIDER_NAME = "balian"
DEFAULT_MODEL_SPEC = {
    "reasoning": False,
    "input": ["text"],
    "cost": {"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0},
    "contextWindow": 131072,
    "maxTokens": 8192,
}
KNOWN_MODEL_OVERRIDES = {
    "qwen-max": {"name": "Qwen Max", "contextWindow": 100000, "maxTokens": 13000},
    "qwen-flash": {"name": "Qwen Flash", "contextWindow": 1000000, "maxTokens": 32000},
    "qwen3-coder-plus": {
        "name": "Qwen3 Coder Plus",
        "contextWindow": 1000000,
        "maxTokens": 100000,
    },
    "qwen3-14b": {"name": "Qwen3 14B"},
}

SITE_TO_BASE_URL = {
    "cn": CN_BASE_URL,
    "intl": INTL_BASE_URL,
    "us": US_BASE_URL,
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Configure OpenClaw to use Alibaba Cloud Model Studio provider."
    )
    parser.add_argument(
        "--config",
        type=Path,
        help="Path to OpenClaw JSON config. Auto-detected if omitted.",
    )
    parser.add_argument(
        "--site",
        choices=["cn", "intl", "us"],
        help="Site region: cn, intl, or us.",
    )
    parser.add_argument("--api-key", help="DashScope API key.")
    parser.add_argument(
        "--api-key-source",
        choices=["env", "inline"],
        default=None,
        help="Where to store provider apiKey: env (recommended) or inline.",
    )
    parser.add_argument(
        "--env-var",
        default="DASHSCOPE_API_KEY",
        help="Environment variable name used when --api-key-source=env.",
    )
    parser.add_argument(
        "--persist-env-shell",
        action="store_true",
        help="Write export line to shell profile (default ~/.bashrc) when using env key mode.",
    )
    parser.add_argument(
        "--shell-profile",
        type=Path,
        default=Path.home() / ".bashrc",
        help="Shell profile file for --persist-env-shell.",
    )
    parser.add_argument(
        "--persist-env-systemd",
        action="store_true",
        help="Write env var to systemd user override and restart service when using env key mode.",
    )
    parser.add_argument(
        "--systemd-service",
        default="openclaw",
        help="Systemd user service name for --persist-env-systemd.",
    )
    parser.add_argument("--model", default=None, help="Primary model ID.")
    parser.add_argument(
        "--models",
        default=None,
        help="Comma-separated provider model IDs. Defaults to qwen-max,qwen-flash,qwen3-coder-plus.",
    )
    parser.add_argument(
        "--list-models",
        action="store_true",
        help="List available models for the selected site and exit.",
    )
    parser.add_argument(
        "--set-default",
        action="store_true",
        help="Set agent default model to this provider/primary model.",
    )
    parser.add_argument(
        "--non-interactive",
        action="store_true",
        help="Fail instead of prompting for missing values.",
    )
    return parser.parse_args()


def detect_config_path() -> Path:
    candidates = [
        Path.home() / ".openclaw" / "openclaw.json",
        Path.home() / ".moltbot" / "moltbot.json",
        Path.home() / ".clawdbot" / "clawdbot.json",
    ]
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return candidates[0]


def parse_csv_models(raw: str) -> list[str]:
    return [item.strip() for item in raw.split(",") if item.strip()]


def dedupe_keep_order(items: list[str]) -> list[str]:
    seen = set()
    out: list[str] = []
    for item in items:
        if item not in seen:
            out.append(item)
            seen.add(item)
    return out


def title_from_model_id(model_id: str) -> str:
    return model_id.replace("-", " ").replace("_", " ").title()


def model_entry(model_id: str) -> Dict[str, Any]:
    entry = {
        "id": model_id,
        "name": title_from_model_id(model_id),
        **DEFAULT_MODEL_SPEC,
    }
    entry.update(KNOWN_MODEL_OVERRIDES.get(model_id, {}))
    return entry


def prompt_site() -> str:
    while True:
        print("Choose Alibaba Cloud Model Studio site:")
        print("  1) Beijing (China site, CN)")
        print("  2) Singapore (International site, INTL)")
        print("  3) Virginia (US site, US)")
        raw = input("Selection [1/2/3] (default 1): ").strip().lower()
        if raw in {"", "1", "cn", "beijing", "china", "zh"}:
            return "cn"
        if raw in {"2", "intl", "int", "international", "singapore", "sg"}:
            return "intl"
        if raw in {"3", "us", "usa", "america", "virginia"}:
            return "us"
        print("Invalid choice. Enter 1 (Beijing/CN), 2 (Singapore/INTL), or 3 (Virginia/US).")


def prompt_api_key() -> str:
    while True:
        value = getpass.getpass("DashScope API key (input hidden): ").strip()
        if value:
            return value
        print("API key cannot be empty.")


def prompt_set_default() -> bool:
    raw = input("Set as default model for agents? [Y/n]: ").strip().lower()
    return raw in {"", "y", "yes"}


def prompt_api_key_source() -> str:
    raw = input(
        "Use safer mode (store API key in environment variable, not openclaw.json)? [Y/n]: "
    ).strip().lower()
    if raw in {"", "y", "yes"}:
        return "env"
    return "inline"


def prompt_can_run_terminal_commands() -> bool:
    raw = input(
        "Can you run commands in terminal now? If yes, env-var mode is safer. [Y/n]: "
    ).strip().lower()
    return raw in {"", "y", "yes"}


def prompt_env_var_name(default_name: str) -> str:
    raw = input(f"Environment variable name (default {default_name}): ").strip()
    return raw or default_name


def prompt_persist_env_shell(default_profile: Path) -> bool:
    raw = input(f"Persist env var to shell profile {default_profile}? [Y/n]: ").strip().lower()
    return raw in {"", "y", "yes"}


def prompt_persist_env_systemd(default_service: str) -> bool:
    raw = input(
        f"Persist env var to systemd user service '{default_service}' and restart it? [y/N]: "
    ).strip().lower()
    return raw in {"y", "yes"}


def prompt_inline_fallback() -> bool:
    raw = input(
        "I still cannot detect the env var after 2 tries. Store API key inline in openclaw.json instead? [y/N]: "
    ).strip().lower()
    return raw in {"y", "yes"}


def prompt_change_primary_model(default_choice: str) -> bool:
    raw = input(f"Change primary model from default ({default_choice})? [y/N]: ").strip().lower()
    return raw in {"y", "yes"}


def prompt_primary_model(configured_models: list[str]) -> str:
    default_choice = DEFAULT_MODEL if DEFAULT_MODEL in configured_models else configured_models[0]
    print("Configured provider models:")
    print(", ".join(configured_models))
    value = input(f"Primary model (default {default_choice}): ").strip()
    return value or default_choice


def load_json(path: Path) -> Dict[str, Any]:
    if not path.exists():
        return {}

    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid JSON in {path}: {exc}") from exc


def ensure_dict(parent: Dict[str, Any], key: str) -> Dict[str, Any]:
    value = parent.get(key)
    if not isinstance(value, dict):
        value = {}
        parent[key] = value
    return value


def backup_file(path: Path) -> Path | None:
    if not path.exists():
        return None
    ts = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    backup = path.with_suffix(path.suffix + f".bak.{ts}")
    shutil.copy2(path, backup)
    return backup


def fetch_models(site: str, api_key: str) -> list[str]:
    url = SITE_TO_BASE_URL[site].rstrip("/") + "/models"
    req = request.Request(
        url=url,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        method="GET",
    )
    try:
        with request.urlopen(req, timeout=20) as resp:
            payload = json.loads(resp.read().decode("utf-8"))
    except error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"Failed to fetch models ({exc.code}): {body}") from exc
    except error.URLError as exc:
        raise RuntimeError(f"Failed to fetch models: {exc.reason}") from exc

    data = payload.get("data", [])
    models = sorted(
        item["id"]
        for item in data
        if isinstance(item, dict) and isinstance(item.get("id"), str) and item["id"]
    )
    return models


def validate_api_key(site: str, api_key: str) -> bool:
    try:
        fetch_models(site=site, api_key=api_key)
    except RuntimeError as exc:
        print(f"API key validation failed: {exc}")
        return False
    return True


def upsert_shell_env(profile_path: Path, var_name: str, value: str) -> None:
    profile_path.parent.mkdir(parents=True, exist_ok=True)
    if profile_path.exists():
        content = profile_path.read_text(encoding="utf-8")
        lines = content.splitlines()
    else:
        lines = []
    pattern = re.compile(rf"^\s*export\s+{re.escape(var_name)}=")
    replaced = False
    for idx, line in enumerate(lines):
        if pattern.match(line):
            lines[idx] = f'export {var_name}="{value}"'
            replaced = True
            break
    if not replaced:
        lines.append(f'export {var_name}="{value}"')
    profile_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def read_shell_env(profile_path: Path, var_name: str) -> str | None:
    if not profile_path.exists():
        return None
    pattern = re.compile(rf'^\s*export\s+{re.escape(var_name)}=(.*)$')
    value: str | None = None
    for line in profile_path.read_text(encoding="utf-8").splitlines():
        match = pattern.match(line.strip())
        if not match:
            continue
        raw = match.group(1).strip()
        if (raw.startswith('"') and raw.endswith('"')) or (raw.startswith("'") and raw.endswith("'")):
            raw = raw[1:-1]
        value = raw
    return value


def print_manual_env_commands(var_name: str, profile_path: Path, value: str | None = None) -> None:
    shown = value if value else "<YOUR_DASHSCOPE_API_KEY>"
    print("Please run these two commands, then come back:")
    print(f'1) export {var_name}="{shown}"')
    print(f'2) echo \'export {var_name}="{shown}"\' >> {profile_path}')


def detect_env_value(var_name: str, profile_path: Path) -> str | None:
    env_val = os.environ.get(var_name)
    if env_val:
        return env_val
    return read_shell_env(profile_path, var_name)


def wait_for_manual_env_setup(var_name: str, profile_path: Path, attempts: int = 2) -> str | None:
    for i in range(1, attempts + 1):
        raw = input(f"Type 'done' after running them (attempt {i}/{attempts}), or 'skip': ").strip().lower()
        if raw == "skip":
            return None
        if raw != "done":
            print("Please type 'done' or 'skip'.")
            continue
        detected = detect_env_value(var_name, profile_path)
        if detected:
            return detected
        print(f"Still cannot detect {var_name} in current env or {profile_path}.")
    return None


def persist_systemd_env(service_name: str, var_name: str, value: str) -> None:
    unit_override_dir = Path.home() / ".config" / "systemd" / "user" / f"{service_name}.service.d"
    unit_override_dir.mkdir(parents=True, exist_ok=True)
    override_file = unit_override_dir / "10-alibaba-cloud-model-setup.conf"
    override_file.write_text(
        f"[Service]\nEnvironment=\"{var_name}={value}\"\n",
        encoding="utf-8",
    )
    subprocess.run(["systemctl", "--user", "daemon-reload"], check=True)
    subprocess.run(["systemctl", "--user", "restart", f"{service_name}.service"], check=True)


def get_systemd_user_env(var_name: str) -> str | None:
    try:
        probe = subprocess.run(
            ["systemctl", "--user", "show-environment"],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            check=False,
        )
    except OSError:
        return None
    if probe.returncode != 0:
        return None
    prefix = f"{var_name}="
    for line in probe.stdout.splitlines():
        if line.startswith(prefix):
            return line[len(prefix) :]
    return None


def detect_systemd_user_service(service_name: str) -> bool:
    try:
        probe = subprocess.run(
            ["systemctl", "--user", "status", f"{service_name}.service"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
    except OSError:
        return False
    # 0: active, 3: inactive but unit exists; both indicate service-managed deployment.
    return probe.returncode in {0, 3}


def resolve_systemd_user_service(preferred_service: str) -> str | None:
    candidates = [preferred_service, "openclaw", "openclaw-gateway"]
    seen: set[str] = set()
    for name in candidates:
        if not name:
            continue
        if name in seen:
            continue
        seen.add(name)
        if detect_systemd_user_service(name):
            return name

    # Fallback: scan user units for openclaw*.service
    try:
        probe = subprocess.run(
            ["systemctl", "--user", "list-unit-files", "--type=service", "--no-legend"],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            check=False,
        )
    except OSError:
        return None
    if probe.returncode != 0:
        return None
    for line in probe.stdout.splitlines():
        unit = line.split(maxsplit=1)[0].strip()
        if unit.startswith("openclaw") and unit.endswith(".service"):
            name = unit[:-8]
            if detect_systemd_user_service(name):
                return name
    return None


def prompt_add_extra_models(site: str, api_key: str) -> list[str]:
    raw = input("Add more models from live list? [y/N]: ").strip().lower()
    if raw not in {"y", "yes"}:
        return []

    try:
        models = fetch_models(site=site, api_key=api_key)
    except RuntimeError as exc:
        print(str(exc))
        manual = input("Unable to fetch list. Enter extra model IDs (comma-separated), or empty to skip: ").strip()
        return parse_csv_models(manual) if manual else []

    if not models:
        print("No models returned by API.")
        return []

    print(f"Available models for {site.upper()} ({len(models)}):")
    for idx, mid in enumerate(models, start=1):
        print(f"{idx:>3}. {mid}")

    raw_pick = input(
        "Pick extra models by index or model ID (comma-separated), empty to skip: "
    ).strip()
    if not raw_pick:
        return []

    picked: list[str] = []
    for token in [t.strip() for t in raw_pick.split(",") if t.strip()]:
        if token.isdigit():
            pos = int(token) - 1
            if 0 <= pos < len(models):
                picked.append(models[pos])
            else:
                print(f"Ignore out-of-range index: {token}")
        else:
            picked.append(token)
    return dedupe_keep_order(picked)


def build_provider_config(site: str, api_key_value: str, model_ids: list[str]) -> Dict[str, Any]:
    return {
        "baseUrl": SITE_TO_BASE_URL[site],
        "apiKey": api_key_value,
        "api": "openai-completions",
        "models": [model_entry(mid) for mid in model_ids],
    }


def apply_config(
    data: Dict[str, Any],
    site: str,
    api_key_value: str,
    model_ids: list[str],
    primary_model: str,
    set_default: bool,
) -> Dict[str, Any]:
    root_models = ensure_dict(data, "models")
    # OpenClaw 2026.x expects provider-centric model config; omit legacy "mode" key.
    root_models.pop("mode", None)
    providers = ensure_dict(root_models, "providers")
    providers[PROVIDER_NAME] = build_provider_config(
        site=site, api_key_value=api_key_value, model_ids=model_ids
    )

    if set_default:
        agents = ensure_dict(data, "agents")
        defaults = ensure_dict(agents, "defaults")
        model_cfg = ensure_dict(defaults, "model")
        model_cfg["primary"] = f"{PROVIDER_NAME}/{primary_model}"
        model_cfg.setdefault("fallbacks", [])
        default_models = ensure_dict(defaults, "models")
        ensure_dict(default_models, f"{PROVIDER_NAME}/{primary_model}")

    return data


def main() -> int:
    args = parse_args()

    config_path = args.config or detect_config_path()
    site = args.site
    api_key = args.api_key
    api_key_source = args.api_key_source
    env_var_name = args.env_var
    primary_model = args.model
    models_arg = args.models
    set_default = args.set_default
    systemd_service = args.systemd_service
    detected_systemd_service: str | None = None
    systemd_detected = False
    runtime_api_key: str | None = None

    if args.list_models and args.non_interactive and not site:
        raise SystemExit("--list-models with --non-interactive requires --site.")

    if args.non_interactive:
        if not api_key_source:
            api_key_source = "env"
        if not site:
            raise SystemExit("--non-interactive requires --site.")
        if not api_key and not (api_key_source == "env" and os.environ.get(env_var_name)):
            raise SystemExit("--non-interactive requires --api-key (or set env var with --api-key-source env).")
        runtime_api_key = api_key or os.environ.get(env_var_name)
    else:
        if not args.api_key_source:
            if prompt_can_run_terminal_commands():
                api_key_source = "env"
            else:
                api_key_source = "inline"
                print("Will store API key inline in config.")
        if not api_key_source:
            api_key_source = "env"
        if api_key_source == "env" and args.env_var == "DASHSCOPE_API_KEY":
            env_var_name = prompt_env_var_name(args.env_var)
        if api_key_source == "env":
            print_manual_env_commands(env_var_name, args.shell_profile, api_key)
            detected = wait_for_manual_env_setup(env_var_name, args.shell_profile, attempts=2)
            if detected:
                runtime_api_key = detected
            elif prompt_inline_fallback():
                api_key_source = "inline"
                print("Will store API key inline in openclaw.json.")
            else:
                print("Config not changed.")
                return 1
        if api_key_source == "inline":
            if not api_key:
                api_key = prompt_api_key()
            runtime_api_key = api_key
        if not site:
            site = prompt_site()

    assert site is not None
    assert api_key_source is not None

    if api_key_source == "env":
        detected_systemd_service = resolve_systemd_user_service(systemd_service)
        systemd_detected = detected_systemd_service is not None

    # Resolve runtime key for validation requests.
    if not runtime_api_key and api_key_source == "env":
        runtime_api_key = os.environ.get(env_var_name) or get_systemd_user_env(env_var_name)
    if not runtime_api_key:
        print("No runtime API key available for validation. Provide --api-key or export the env var first.")
        return 1

    if args.list_models:
        try:
            live_models = fetch_models(site=site, api_key=runtime_api_key)
        except RuntimeError as exc:
            print(str(exc))
            return 1
        if not live_models:
            print("No models returned by API.")
            return 0
        print(f"Available models for {site.upper()} ({len(live_models)}):")
        for mid in live_models:
            print(mid)
        return 0

    # Fail closed for systemd deployments: ensure runtime env before any write.
    if api_key_source == "env" and systemd_detected and not get_systemd_user_env(env_var_name):
        try:
            subprocess.run(
                ["systemctl", "--user", "set-environment", f"{env_var_name}={runtime_api_key}"],
                check=True,
            )
        except (OSError, subprocess.CalledProcessError) as exc:
            print(f"Failed to set systemd user environment: {exc}")
            print("Config not changed.")
            return 1
        if not get_systemd_user_env(env_var_name):
            print(
                f"Systemd user environment missing {env_var_name}; "
                "refusing to update config to avoid gateway startup failure."
            )
            print("Config not changed.")
            return 1

    # Validate key before changing local config.
    if not validate_api_key(site=site, api_key=runtime_api_key):
        print("Config not changed because API key validation did not pass.")
        return 1

    if models_arg:
        model_ids = parse_csv_models(models_arg)
    else:
        model_ids = list(DEFAULT_PRESET_MODELS)
        if not args.non_interactive:
            model_ids = dedupe_keep_order(
                model_ids + prompt_add_extra_models(site=site, api_key=runtime_api_key)
            )
    if not model_ids:
        model_ids = list(DEFAULT_PRESET_MODELS)

    if primary_model:
        model_ids = dedupe_keep_order(model_ids + [primary_model])
    else:
        if args.non_interactive:
            primary_model = DEFAULT_MODEL
            model_ids = dedupe_keep_order(model_ids + [primary_model])
        else:
            default_choice = DEFAULT_MODEL if DEFAULT_MODEL in model_ids else model_ids[0]
            if prompt_change_primary_model(default_choice):
                primary_model = prompt_primary_model(model_ids)
            else:
                primary_model = default_choice
            model_ids = dedupe_keep_order(model_ids + [primary_model])

    if not args.non_interactive and not args.set_default:
        set_default = prompt_set_default()

    assert primary_model is not None

    data = load_json(config_path)
    api_key_value = runtime_api_key if api_key_source == "inline" else f"${{{env_var_name}}}"
    updated = apply_config(
        data=data,
        site=site,
        api_key_value=api_key_value,
        model_ids=model_ids,
        primary_model=primary_model,
        set_default=set_default,
    )

    config_path.parent.mkdir(parents=True, exist_ok=True)
    backup = backup_file(config_path)
    config_path.write_text(json.dumps(updated, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    restarted = False
    if systemd_detected:
        try:
            assert detected_systemd_service is not None
            subprocess.run(["systemctl", "--user", "restart", f"{detected_systemd_service}.service"], check=True)
            restarted = True
        except (OSError, subprocess.CalledProcessError) as exc:
            print(f"Warning: config updated, but failed to restart {detected_systemd_service}.service: {exc}")
    else:
        try:
            subprocess.run(["openclaw", "gateway", "restart"], check=True)
            restarted = True
        except (OSError, subprocess.CalledProcessError):
            pass

    print(f"Updated config: {config_path}")
    if backup:
        print(f"Backup created: {backup}")
    print(f"Provider: {PROVIDER_NAME}")
    print(f"Provider models: {', '.join(model_ids)}")
    print(f"Primary model: {primary_model}")
    print(f"Site: {site.upper()}")
    print(f"Base URL: {SITE_TO_BASE_URL[site]}")
    print(f"API key source: {'environment variable' if api_key_source == 'env' else 'openclaw.json'}")
    if api_key_source == "env":
        print(f"Configured apiKey value: ${{{env_var_name}}}")
        print(f"Ensure `{env_var_name}` is exported before starting OpenClaw.")
    if set_default:
        print(f"Default model set to: {PROVIDER_NAME}/{primary_model}")
    else:
        print("Default model unchanged.")
    if restarted:
        print("OpenClaw gateway restarted.")
    else:
        print("Please restart OpenClaw gateway/service to apply changes.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
