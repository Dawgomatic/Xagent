# Meegle API Skill

> OpenClaw skills for Meegle (Feishu Project / Lark Project) Open API

This repository provides a collection of OpenClaw skills that help agents understand and correctly call Meegle Open API for space, work items, comments, views, and related operations.

## Overview

Meegle API Skill splits Meegle OpenAPI into multiple sub-skills by functional area. Before calling any Meegle API, read **meegle-api-credentials** first to obtain domain, access token, context (project_key, user_key), and request headers. Other skills (Users, Space, Work Items, Setting, etc.) assume you have already obtained everything from credentials.

## Required Credentials

Required credentials are declared in [meegle-api-credentials](./meegle-api-credentials/SKILL.md). For obtaining domain and token you need:

| Credential | Description | Where to obtain |
|------------|-------------|-----------------|
| `plugin_id` | Plugin ID | Meegle Developer Platform → Plugin → Basic Information |
| `plugin_secret` | Plugin secret | Meegle Developer Platform → Plugin → Basic Information |
| `domain` | API host | `project.larksuite.com` (international) or `project.feishu.cn` (China) |

Optional: `authorization_code` and `refresh_token` for obtaining `user_access_token`. Context (`project_key`, `user_key`) and request headers are also documented in [meegle-api-credentials](./meegle-api-credentials/SKILL.md); in Meegle Developer Platform, double-click the **project icon** for `project_key` and the **avatar** for `user_key`.

## Skill List

| Order | Sub-skill | When to read |
|-------|-----------|--------------|
| 1 | [meegle-api-credentials](./meegle-api-credentials/SKILL.md) | Domain, token, context (project_key, user_key), request headers. **Read this before any Meegle API call.** |
| 2 | [meegle-api-users](./meegle-api-users/SKILL.md) | User-related OpenAPIs (e.g. user groups, members) |
| 3 | [meegle-api-space](./meegle-api-space/SKILL.md) | Space (project) operations |
| 4 | [meegle-api-work-items](./meegle-api-work-items/SKILL.md) | Create, get, update work items (tasks, stories, bugs, etc.) |
| 5 | [meegle-api-setting](./meegle-api-setting/SKILL.md) | Settings, work item types, fields, process configuration |
| 6 | [meegle-api-comments](./meegle-api-comments/SKILL.md) | Comments on work items or other entities |
| 7 | [meegle-api-views-measurement](./meegle-api-views-measurement/SKILL.md) | Views, kanban, Gantt, charts, measurement |

## Work Item Sub-skills

`meegle-api-work-items` includes:

| Sub-skill | Directory | Description |
|-----------|-----------|-------------|
| Create / Read / Update work items | `work-item-read-and-write/` | Create work items, get details, update work items |
| List & search work items | `work-item-lists/` | Filter, search, full-text search, associated items, universal search |
| Workflows & nodes | `workflows-and-nodes/` | Workflow and node related APIs |
| Tasks | `tasks/` | Task related APIs |
| Attachment | `attachment/` | Work item attachment related APIs |
| Space association | `space-association/` | Space association related APIs |
| Group | `group/` | Work item group related APIs |

## Usage

1. Reference this repository as a Cursor skill, or copy the relevant `SKILL.md` files into your Cursor skills directory.
2. When working on Meegle-related tasks in Cursor, have the AI use the `Read` tool to load the corresponding `SKILL.md` for the needed API area.
3. Before any Meegle API call, have the AI read **meegle-api-credentials** to obtain domain, token, request headers, and context.
4. If you are using **OpenClaw**, you can search for this skill pack on **Clawhub** by name `meegle-api-skill` and install it directly via the Clawhub CLI

## API Regions

- **International**: `https://project.larksuite.com`
- **China (Feishu Project)**: `https://project.feishu.cn`

See [meegle-api-credentials](./meegle-api-credentials/SKILL.md) for domain and authentication details.

## Repository Structure

```
meegle-api-skill/
├── SKILL.md                      # Skill index (entry point)
├── README.md                     # This file
├── meegle-api-credentials/       # Domain, token, context, request headers
├── meegle-api-users/             # User-related OpenAPIs
├── meegle-api-space/             # Space
├── meegle-api-work-items/        # Work items (includes sub-skills)
├── meegle-api-setting/           # Settings
├── meegle-api-comments/          # Comments
└── meegle-api-views-measurement/ # Views and measurement
```

## License

See the LICENSE file in the repository root, if present.
