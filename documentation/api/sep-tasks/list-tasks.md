---
title: List available tasks
slug: sep-list-tasks
category:
  uri: sep-tasks-api
position: 5
---

List registered tasks to find the task names you need for [execute](ref:sep-execute-task) and [schedule](ref:sep-list-periodic-tasks) calls.

## List all tasks

```
GET /
```

Returns a paginated list of all tasks registered in SEP.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `owner` | string | none | Filter by the app that owns the task (for example, `mysql-backups`) |
| `target` | string | none | Filter by target |
| `backup_type` | string | none | Filter by backup type |
| `search` | string | none | Case-insensitive search across task names and other searchable columns |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `-created_at` | Sort key. Prefix with `-` for descending. Options: `backend`, `created_at`, `name`, `owner`, `updated_at` |

**Example:**

```shell
curl -sk "https://<pmm-server>/sep/?owner=mysql-backups" \
     -H "Authorization: Bearer <token>"
```

## Get a specific task

```
GET /{task_name}
```

Returns the full record for a single task by name.

**Example:**

```shell
curl -sk https://<pmm-server>/sep/mysql-backup-xtrabackup \
     -H "Authorization: Bearer <token>"
```

## List executor hosts

```
GET /hosts/
```

Returns the available executor hosts where tasks can run. Use this to find valid `target` values for the [execute](ref:sep-execute-task) and [connectivity check](ref:sep-connectivity-check) endpoints.
