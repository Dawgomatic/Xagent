#!/usr/bin/env python3
"""
HFP Klientenportal - RZL Portal Automation

Upload accounting documents to HFP tax accountant portal.
Uses sync Playwright for browser automation.
"""

import sys
sys.stdout.reconfigure(line_buffering=True)
sys.stderr.reconfigure(line_buffering=True)

import argparse
import json
import os
import re
import shutil
import time
from pathlib import Path
from datetime import datetime

# Fast path: allow --help without requiring Playwright
if "-h" in sys.argv or "--help" in sys.argv:
    sync_playwright = None  # type: ignore[assignment]
    PlaywrightTimeout = Exception  # type: ignore[assignment]
else:
    try:
        from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeout
    except ImportError:
        print("ERROR: playwright not installed. Run: pip install playwright && playwright install chromium")
        sys.exit(1)

# ---------------------------------------------------------------------------
# Workspace & path resolution (§5.1, §5.22)
# ---------------------------------------------------------------------------

_SKILL_NAME = "klientenportal"


def _find_workspace_root() -> Path:
    """Find the workspace root (directory containing 'skills/').

    Resolution order:
    1. OPENCLAW_WORKSPACE env var
    2. CWD if it contains 'skills/'
    3. Walk up from script location
    4. Fall back to CWD
    """
    env = os.environ.get("OPENCLAW_WORKSPACE")
    if env:
        return Path(env).resolve()

    cwd = Path.cwd()
    if (cwd / "skills").is_dir():
        return cwd

    d = Path(__file__).resolve().parent
    for _ in range(6):
        if (d / "skills").is_dir() and d != d.parent:
            return d
        d = d.parent

    return cwd


WORKSPACE_ROOT = _find_workspace_root()
CONFIG_DIR = WORKSPACE_ROOT / _SKILL_NAME
CONFIG_FILE = CONFIG_DIR / "config.json"
PROFILE_DIR = CONFIG_DIR / ".pw-profile"

# ---------------------------------------------------------------------------
# State directory hardening (§5.15)
# ---------------------------------------------------------------------------


def _set_strict_umask() -> None:
    try:
        os.umask(0o077)
    except Exception:
        pass


def _harden_path(p: Path) -> None:
    try:
        if p.is_dir():
            os.chmod(p, 0o700)
        elif p.is_file():
            os.chmod(p, 0o600)
    except Exception:
        pass


_set_strict_umask()

# ---------------------------------------------------------------------------
# Output path sandboxing (§5.3)
# ---------------------------------------------------------------------------

_TMP_ROOT = Path(os.environ.get("OPENCLAW_TMP") or "/tmp").expanduser().resolve()
DEFAULT_OUTPUT_DIR = _TMP_ROOT / "openclaw" / _SKILL_NAME


def _safe_output_path(raw: str) -> Path:
    """Validate output path is within workspace or /tmp."""
    p = Path(raw).expanduser().resolve()
    ws = WORKSPACE_ROOT.resolve()
    tmp = Path("/tmp").resolve()
    if p == ws or p.is_relative_to(ws):
        return p
    if p == tmp or p.is_relative_to(tmp):
        return p
    raise SystemExit(
        f"ERROR: output path must be under workspace ({ws}) or /tmp, got: {p}"
    )


# ---------------------------------------------------------------------------
# Filename sanitization (§5.5)
# ---------------------------------------------------------------------------


def _safe_filename(name: str) -> str:
    """Sanitize a string for use as a filename component."""
    name = name.replace("/", "_").replace("\\", "_")
    name = re.sub(r"\.\.+", ".", name)
    name = re.sub(r"[^\w\s\-.]", "_", name)
    return name.strip().strip(".")


# ---------------------------------------------------------------------------
# Config (§5.2 — no .env, config.json only)
# ---------------------------------------------------------------------------

BELEGKREIS_MAP = {
    "ER": "Eingangsrechnungen",
    "AR": "Ausgangsrechnungen",
    "KA": "Kassa",
    "SP": "Sparkasse",
}


def _load_config() -> dict:
    """Load config from config.json or env vars.

    Env vars (override config.json values):
        KLIENTENPORTAL_PORTAL_ID, KLIENTENPORTAL_USER_ID, KLIENTENPORTAL_PASSWORD
    """
    cfg: dict = {}
    if CONFIG_FILE.exists():
        cfg = json.loads(CONFIG_FILE.read_text(encoding="utf-8"))
        _harden_path(CONFIG_FILE)

    # Env vars override / supplement config.json
    portal_id = os.environ.get("KLIENTENPORTAL_PORTAL_ID") or cfg.get("portal_id", "")
    user_id = os.environ.get("KLIENTENPORTAL_USER_ID") or cfg.get("user_id", "")
    password = os.environ.get("KLIENTENPORTAL_PASSWORD") or cfg.get("password", "")

    if not portal_id or not user_id or not password:
        missing = [n for n, v in [("portal_id", portal_id), ("user_id", user_id), ("password", password)] if not v]
        print(f"ERROR: Missing config: {', '.join(missing)}")
        print(f"Create {CONFIG_FILE} or set env vars. See SKILL.md for details.")
        sys.exit(1)

    cfg["portal_id"] = portal_id
    cfg["portal_url"] = cfg.get("portal_url") or f"https://klientenportal.at/prod/{portal_id}"
    cfg["user_id"] = user_id
    cfg["password"] = password
    return cfg


def _ensure_dir(p: Path) -> None:
    p.mkdir(parents=True, exist_ok=True)
    _harden_path(p)


# ---------------------------------------------------------------------------
# Browser helpers
# ---------------------------------------------------------------------------


def dismiss_modals(page):
    """Dismiss any modal dialogs (Wartungsmeldung, etc.)."""
    for _attempt in range(5):
        try:
            ok_btn = page.locator('dialog button:has-text("OK")')
            if ok_btn.count() > 0:
                ok_btn.first.click(force=True, timeout=2000)
                time.sleep(0.5)
                continue
        except Exception:
            pass
        try:
            page.keyboard.press("Escape")
            time.sleep(0.3)
        except Exception:
            pass
        try:
            modal_root = page.locator("dxbl-modal-root")
            if modal_root.count() == 0:
                break
        except Exception:
            break
    try:
        page.locator('text="Klient:"').click(timeout=1000)
        time.sleep(0.3)
    except Exception:
        pass


def _launch_context(pw, *, visible: bool = False):
    """Launch a persistent browser context."""
    _ensure_dir(PROFILE_DIR)
    return pw.chromium.launch_persistent_context(
        user_data_dir=str(PROFILE_DIR),
        headless=not visible,
        viewport={"width": 1280, "height": 900},
    )


def login(page, config: dict) -> bool:
    """Log in to the portal."""
    portal_url = config["portal_url"]

    # Already logged in?
    if page.locator("text=Abmelden").count() > 0:
        print("[login] Session still valid", flush=True)
        dismiss_modals(page)
        return True

    page.goto(f"{portal_url}/account/login", wait_until="networkidle")
    time.sleep(1)

    if page.locator("text=Abmelden").count() > 0:
        print("[login] Already logged in", flush=True)
        dismiss_modals(page)
        return True

    print("[login] Logging in...", flush=True)
    page.locator('input[type="text"]').first.fill(config["user_id"])
    page.locator('input[type="password"]').first.fill(config["password"])
    page.locator('button:has-text("Login")').click()

    page.wait_for_load_state("networkidle")
    time.sleep(1)
    dismiss_modals(page)

    if page.locator("text=Abmelden").count() > 0:
        print("[login] Success", flush=True)
        return True
    print("[login] Failed", flush=True)
    return False


def _ensure_logged_in(page, config: dict) -> bool:
    """Navigate to upload page, log in if needed."""
    portal_url = config["portal_url"]
    page.goto(f"{portal_url}/Klient/Beleg/BelegTransfer/upload", wait_until="networkidle")
    time.sleep(1)

    if "/account/login" in page.url:
        print("[upload] Session expired, logging in...", flush=True)
        if not login(page, config):
            return False
        page.goto(
            f"{portal_url}/Klient/Beleg/BelegTransfer/upload",
            wait_until="networkidle",
        )
        time.sleep(1)

    dismiss_modals(page)
    return True


def _select_belegkreis(page, belegkreis: str) -> None:
    """Select a Belegkreis on the upload page."""
    if belegkreis == "ER":
        return  # Default
    print(f"[upload] Selecting Belegkreis: {belegkreis}", flush=True)
    try:
        bk_combo = (
            page.locator('text="Belegkreis:"').locator("..").locator('[role="combobox"]')
        )
        bk_combo.click(force=True, timeout=5000)
        time.sleep(0.5)
        bk_combo.fill(belegkreis)
        time.sleep(0.5)
        page.keyboard.press("Enter")
        time.sleep(0.5)
    except Exception as e:
        print(f"[upload] Warning: Could not set Belegkreis: {e}", flush=True)


def upload_file(page, file_path: Path) -> bool:
    """Upload a single file via Direktübermittlung."""
    print(f"[upload] Uploading: {file_path.name}", flush=True)
    try:
        file_inputs = page.locator('input[type="file"]')
        if file_inputs.count() == 0:
            print("[upload] ERROR: No file input found", flush=True)
            return False
        file_inputs.first.set_input_files(str(file_path))
        time.sleep(3)
        print(f"[upload] ✓ {file_path.name}", flush=True)
        return True
    except Exception as e:
        print(f"[upload] ERROR: {e}", flush=True)
        return False


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------


def cmd_login(args):
    """Test login."""
    config = _load_config()
    with sync_playwright() as p:
        context = _launch_context(p, visible=args.visible)
        page = context.new_page()
        try:
            return 0 if login(page, config) else 1
        finally:
            context.close()


def cmd_upload(args):
    """Upload files to portal."""
    config = _load_config()

    # Resolve files (restricted to workspace + /tmp)
    allowed_roots: list[Path] = [Path("/tmp").resolve()]
    cwd = Path.cwd()
    if (cwd / "skills").is_dir():
        allowed_roots.append(cwd.resolve())
    ws = os.environ.get("OPENCLAW_WORKSPACE")
    if ws:
        allowed_roots.append(Path(ws).resolve())

    def _is_allowed(p: Path) -> bool:
        s = str(p)
        return any(s.startswith(str(a) + "/") or s == str(a) for a in allowed_roots)

    files: list[Path] = []
    for pattern in args.file:
        path = Path(pattern).expanduser()
        if path.is_file():
            resolved = path.resolve()
            if not _is_allowed(resolved):
                print(f"ERROR: Refusing to upload '{pattern}' — must be inside workspace or /tmp")
                return 1
            files.append(resolved)
        else:
            parent = path.parent if path.parent.exists() else Path.cwd()
            for m in sorted(parent.glob(path.name)):
                if m.is_file():
                    resolved = m.resolve()
                    if not _is_allowed(resolved):
                        print(f"ERROR: Refusing to upload '{m}' — must be inside workspace or /tmp")
                        return 1
                    files.append(resolved)

    if not files:
        print("ERROR: No files found to upload")
        return 1

    print(f"[upload] Uploading {len(files)} file(s) to Belegkreis {args.belegkreis}")

    with sync_playwright() as p:
        context = _launch_context(p, visible=args.visible)
        page = context.new_page()
        try:
            if not _ensure_logged_in(page, config):
                return 1

            _select_belegkreis(page, args.belegkreis)

            ok = 0
            for file_path in files:
                if upload_file(page, file_path):
                    ok += 1
                time.sleep(1)

            print(f"\n[upload] Uploaded {ok}/{len(files)} files")
            return 0 if ok == len(files) else 1
        finally:
            context.close()


def cmd_released(args):
    """List released (freigegebene) files."""
    config = _load_config()
    with sync_playwright() as p:
        context = _launch_context(p, visible=args.visible)
        page = context.new_page()
        try:
            portal_url = config["portal_url"]
            page.goto(f"{portal_url}/Klient/Beleg/BelegHistory", wait_until="networkidle")
            time.sleep(1)
            if "/account/login" in page.url:
                if not login(page, config):
                    return 1
                page.goto(f"{portal_url}/Klient/Beleg/BelegHistory", wait_until="networkidle")
                time.sleep(1)
            dismiss_modals(page)

            # Extract table rows
            rows = page.locator("table tbody tr")
            count = rows.count()
            if count == 0:
                print("No released files found.")
                return 0
            for i in range(count):
                cells = rows.nth(i).locator("td")
                parts = [cells.nth(j).inner_text().strip() for j in range(min(cells.count(), 5))]
                print(" | ".join(parts))
            return 0
        finally:
            context.close()


def cmd_download(args):
    """Download Kanzleidokumente."""
    config = _load_config()
    out_dir = _safe_output_path(args.output) if args.output else DEFAULT_OUTPUT_DIR
    out_dir.mkdir(parents=True, exist_ok=True)

    with sync_playwright() as p:
        context = _launch_context(p, visible=args.visible)
        page = context.new_page()
        try:
            portal_url = config["portal_url"]
            page.goto(f"{portal_url}/Klient/Beleg/BelegTransfer/noupload", wait_until="networkidle")
            time.sleep(1)
            if "/account/login" in page.url:
                if not login(page, config):
                    return 1
                page.goto(
                    f"{portal_url}/Klient/Beleg/BelegTransfer/noupload",
                    wait_until="networkidle",
                )
                time.sleep(1)
            dismiss_modals(page)

            # Find download links
            links = page.locator('a[href*="download"], a[href*="Download"]')
            count = links.count()
            if count == 0:
                print("No documents available for download.")
                return 0

            print(f"[download] Found {count} document(s)")
            for i in range(count):
                link = links.nth(i)
                with page.expect_download() as dl_info:
                    link.click()
                download = dl_info.value
                suggested = _safe_filename(download.suggested_filename or f"document_{i}.pdf")
                dest = out_dir / suggested
                download.save_as(dest)
                print(f"[download] ✓ {dest}")

            return 0
        finally:
            context.close()


def cmd_logout(args):
    """Clear session (delete Playwright profile)."""
    print(f"[logout] Clearing profile: {PROFILE_DIR}")
    if PROFILE_DIR.exists():
        shutil.rmtree(PROFILE_DIR)
        print("[logout] ✓ Session cleared")
    else:
        print("[logout] No profile to clear")
    return 0


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(
        description="HFP Klientenportal Automation",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s login
  %(prog)s upload -f invoice.pdf --belegkreis KA
  %(prog)s upload -f *.xml --belegkreis SP
  %(prog)s released
  %(prog)s download
  %(prog)s logout
        """,
    )

    parser.add_argument("--visible", action="store_true", help="Show browser")

    subparsers = parser.add_subparsers(dest="command", required=True)

    # login
    subparsers.add_parser("login", help="Test login").set_defaults(func=cmd_login)

    # upload
    upload_p = subparsers.add_parser("upload", help="Upload files")
    upload_p.add_argument(
        "-f", "--file", nargs="+", required=True, help="File(s) to upload"
    )
    upload_p.add_argument(
        "--belegkreis",
        default="ER",
        choices=list(BELEGKREIS_MAP.keys()),
        help="Document category (default: ER)",
    )
    upload_p.set_defaults(func=cmd_upload)

    # released
    subparsers.add_parser("released", help="List released files").set_defaults(
        func=cmd_released
    )

    # download
    dl_p = subparsers.add_parser("download", help="Download Kanzleidokumente")
    dl_p.add_argument("-o", "--output", help="Output directory (default: /tmp/openclaw/klientenportal)")
    dl_p.set_defaults(func=cmd_download)

    # logout
    subparsers.add_parser("logout", help="Clear session").set_defaults(func=cmd_logout)

    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main() or 0)
