# Pylon API quick reference

- **Base URL:** `https://api.usepylon.com`
- **Auth:** Bearer token (`Authorization: Bearer <token>`). Store the token in `PYLON_API_TOKEN`.
- **Pagination:** Most list endpoints return `{ "pagination": { "has_next_page": bool, "cursor": "..." } }`. Pass `cursor=<value>` to the same endpoint to continue.
- **Rate limits:** Documented per endpoint (commonly 10–20 req/min). Handle HTTP 429 by retrying later.

## Core ticket endpoints

| Endpoint | Purpose | Notes |
| --- | --- | --- |
| `GET /issues` | List issues | Supports filters: `state`, `team_id`, `assignee_id`, `requester_id`, `tag`, `limit`, `cursor`. Returns `data[]` of Issue objects plus pagination. |
| `GET /issues/{id}` | Fetch a specific issue | Returns a full Issue payload (title, state, requester, link, tags, custom fields, etc.). |
| `PATCH /issues/{id}` | Update an issue | Supply a JSON body with fields to change (e.g., `{ "state": "waiting_on_customer" }`). |
| `POST /issues/{id}/messages` | Add a reply or internal note | Body example: `{ "message_html": "<p>Update text</p>", "is_private": true }`. |
| `GET /issues/{id}/messages` | List all messages on an issue | Messages include HTML body, author, `is_private`, attachments. |
| `GET /issues/{id}/threads` | List threads (customer-visible or internal) | Each message references `thread_id`. |

## Useful supporting endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /users` | Resolve team member IDs for assignee filters. |
| `GET /contacts` | List customer contacts (requesters). |
| `GET /teams` | List teams and their IDs. |
| `GET /tags` | Enumerate available tags. |

## Example requests

List open tickets assigned to you:

```bash
python3 scripts/pylon_request.py /issues --param state=waiting_on_you --param assignee_id=<USER_ID>
```

Move a ticket into "waiting_on_customer":

```bash
python3 scripts/pylon_request.py /issues/ISSUE_ID \
  --method PATCH \
  --data '{"state":"waiting_on_customer"}'
```

Add an internal note:

```bash
python3 scripts/pylon_request.py /issues/ISSUE_ID/messages \
  --method POST \
  --data '{"message_html":"<p>Reminder to loop product</p>","is_private":true}'
```

Full docs: https://docs.usepylon.com/pylon-docs/developer/api/api-reference
