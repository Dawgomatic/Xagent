---
name: pylon-support
description: Work with Pylon tickets via their REST API. Use when you need to list or inspect issues, add internal notes/customer replies, or run any ad‑hoc Pylon API call.
---

# Pylon Support Operations

This skill bundles lightweight tooling for interacting with Pylon's REST API so you can audit tickets, chase follow‑ups, or post updates without leaving the terminal.

## Setup

1. Create a Pylon API token with the permissions you need (issues, messages, contacts, etc.).
2. Export it before running any script:
   ```bash
   export PYLON_API_TOKEN="<token>"
   ```
3. Optional: override the base URL (for staging) with `PYLON_API_BASE`.

See [`references/pylon_api.md`](references/pylon_api.md) for endpoint summaries and example payloads.

## Scripts

### `scripts/pylon_list_issues.py`
Quickly dumps `/issues` with common filters so you can spot blockers.

```bash
python3 scripts/pylon_list_issues.py --state waiting_on_you --limit 25
python3 scripts/pylon_list_issues.py --assignee-id usr_123 --team-id team_9
```

The script prints the API response and, when applicable, the `cursor` for the next page. Feed that cursor back through `--page-cursor` to continue.

### `scripts/pylon_request.py`
General-purpose wrapper for any Pylon endpoint. Provide the path, method, and optional params/body.

```bash
# Update a ticket state
python3 scripts/pylon_request.py /issues/iss_123 \
  --method PATCH \
  --data '{"state":"waiting_on_customer"}'

# Add an internal note
python3 scripts/pylon_request.py /issues/iss_123/messages \
  --method POST \
  --data '{"message_html":"<p>Looping product...</p>","is_private":true}'

# Fetch issue messages (GET is default)
python3 scripts/pylon_request.py /issues/iss_123/messages
```

Flags:
- `--param key=value` (repeatable) to add query params.
- `--data '{...}'` or `--data-file payload.json` for the request body.

### `scripts/pylon_client.py`
Shared helper that handles auth, base URL, and JSON parsing. Import it if you add more task-specific scripts.

## Workflow tips

1. **Triage every morning**: `pylon_list_issues.py --state waiting_on_you` gives you the queue of items where your team owes a reply.
2. **Deep dive a ticket**: use `pylon_request.py /issues/<id>` to pull the metadata, then `/issues/<id>/messages` for the conversation history.
3. **Update statuses quickly**: patch the issue state or snooze window via `--method PATCH` calls.
4. **Add context while handoffs happen**: post internal notes with `is_private=true` before tagging teammates in Slack.

Refer to [`references/pylon_api.md`](references/pylon_api.md) for the full list of helpful endpoints (users, contacts, tags, etc.) and link to the official docs when you need fields not covered here.
