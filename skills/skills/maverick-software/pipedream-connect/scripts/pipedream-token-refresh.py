#!/usr/bin/env python3
"""
Pipedream Token Refresh Script

Refreshes OAuth access tokens for all Pipedream MCP servers in mcporter.json.
Tokens expire after 1 hour, so this should run at least every 50 minutes.

Setup:
  1. Copy to ~/clawd/scripts/pipedream-token-refresh.py
  2. Add cron job: */45 * * * * python3 ~/clawd/scripts/pipedream-token-refresh.py

Usage:
  python3 pipedream-token-refresh.py [--config PATH] [--quiet]

Options:
  --config PATH   Path to mcporter.json (default: ~/clawd/config/mcporter.json)
  --quiet         Only output errors
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

# Default paths
DEFAULT_CONFIG = Path.home() / "clawd" / "config" / "mcporter.json"
CREDENTIALS_FILE = Path.home() / "clawd" / "config" / "pipedream-credentials.json"
LOG_FILE = Path.home() / "clawd" / "logs" / "pipedream-token-refresh.log"

# Parse arguments
quiet = "--quiet" in sys.argv or "-q" in sys.argv
config_path = DEFAULT_CONFIG

for i, arg in enumerate(sys.argv[1:], 1):
    if arg in ("--config", "-c") and i < len(sys.argv) - 1:
        config_path = Path(sys.argv[i + 1])


def log(message: str, is_error: bool = False):
    """Log a message with timestamp."""
    if quiet and not is_error:
        return
    
    LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().isoformat()
    log_entry = f"[{timestamp}] {message}\n"
    
    try:
        with open(LOG_FILE, "a") as f:
            f.write(log_entry)
    except:
        pass  # Don't fail if we can't write to log
    
    if is_error:
        print(log_entry.strip(), file=sys.stderr)
    else:
        print(log_entry.strip())


def get_new_token(client_id: str, client_secret: str) -> dict:
    """Request a new access token from Pipedream."""
    url = "https://api.pipedream.com/v1/oauth/token"
    data = json.dumps({
        "grant_type": "client_credentials",
        "client_id": client_id,
        "client_secret": client_secret
    }).encode("utf-8")
    
    req = Request(url, data=data, headers={
        "Content-Type": "application/json"
    })
    
    try:
        with urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except HTTPError as e:
        error_body = e.read().decode("utf-8") if e.fp else str(e)
        raise Exception(f"HTTP {e.code}: {error_body}")
    except URLError as e:
        raise Exception(f"URL Error: {e.reason}")


def get_credentials():
    """Get Pipedream credentials from various sources."""
    # First try dedicated credentials file
    if CREDENTIALS_FILE.exists():
        try:
            with open(CREDENTIALS_FILE) as f:
                creds = json.load(f)
                if creds.get("clientId") and creds.get("clientSecret"):
                    return creds["clientId"], creds["clientSecret"]
        except:
            pass
    
    # Fall back to extracting from mcporter config
    if config_path.exists():
        try:
            with open(config_path) as f:
                config = json.load(f)
            
            for name, server in config.get("mcpServers", {}).items():
                if name.startswith("pipedream"):
                    env = server.get("env", {})
                    client_id = env.get("PIPEDREAM_CLIENT_ID")
                    client_secret = env.get("PIPEDREAM_CLIENT_SECRET")
                    if client_id and client_secret:
                        return client_id, client_secret
        except:
            pass
    
    return None, None


def main():
    # Check config exists
    if not config_path.exists():
        log(f"ERROR: Config file not found: {config_path}", is_error=True)
        sys.exit(1)
    
    # Get credentials
    client_id, client_secret = get_credentials()
    
    if not client_id or not client_secret:
        log("ERROR: No Pipedream credentials found", is_error=True)
        log("  Checked: " + str(CREDENTIALS_FILE), is_error=True)
        log("  Checked: " + str(config_path), is_error=True)
        sys.exit(1)
    
    log("Requesting new access token...")
    
    try:
        response = get_new_token(client_id, client_secret)
    except Exception as e:
        log(f"ERROR: Failed to get token: {e}", is_error=True)
        sys.exit(1)
    
    new_token = response.get("access_token")
    if not new_token:
        log(f"ERROR: No access_token in response: {response}", is_error=True)
        sys.exit(1)
    
    expires_in = response.get("expires_in", 3600)
    log(f"New token obtained, expires in {expires_in}s")
    
    # Load config
    with open(config_path) as f:
        config = json.load(f)
    
    # Update all pipedream servers
    updated = 0
    for name, server in config.get("mcpServers", {}).items():
        if name.startswith("pipedream"):
            if "headers" not in server:
                server["headers"] = {}
            server["headers"]["Authorization"] = f"Bearer {new_token}"
            updated += 1
    
    if updated == 0:
        log("WARNING: No Pipedream servers found in config")
        sys.exit(0)
    
    # Write back atomically
    temp_file = config_path.with_suffix(".tmp")
    with open(temp_file, "w") as f:
        json.dump(config, f, indent=2)
    temp_file.rename(config_path)
    
    log(f"Successfully updated {updated} Pipedream server(s) with new token")


if __name__ == "__main__":
    main()
