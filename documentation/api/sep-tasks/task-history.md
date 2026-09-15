---
title: Monitor task runs
slug: sep-task-history
category:
  uri: sep-tasks-api
position: 2
---

After [executing a task](ref:sep-execute-task), use the history endpoints to track status, stream logs, browse output files, and stop a run in progress.

## List runs for a task

```
GET /{task}/history/
```

Returns paginated execution history for a task, newest first.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `status` | string | none | Filter by status: `failed`, `pending`, `running`, `success`, `stopped`, `lost`, `stale`, `unlaunchable` |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `-created_at` | Sort key. Prefix with `-` for descending. Options: `created_at`, `executed_by`, `finished_at`, `started_at`, `status` |
| `search` | string | none | Case-insensitive search across searchable columns |

**Example:**

```shell
curl -sk "https://<pmm-server>/sep/mysql-backup-xtrabackup/history/?status=failed&limit=10" \
     -H "Authorization: Bearer <token>"
```

## Get a single run

```
GET /history/{task_history_id}
```

Returns the full record for one execution, identified by the `id` returned when the task was executed.

**Example:**

```shell
curl -sk https://<pmm-server>/sep/history/42 \
     -H "Authorization: Bearer <token>"
```

## Stream logs

```
GET /history/{task_history_id}/logs/
```

Streams log output for a run. While the task is running, returns a live stream. For finished tasks, returns stored logs.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `step` | string | none | Filter to a specific step name within the run |
| `tail` | integer | none | For finished tasks only: return only the last N lines |
| `backend` | string | `nomad` | Executor backend: `nomad`, `proxy`, or `celery` |

**Example:**

```shell
curl -sk "https://<pmm-server>/sep/history/42/logs/" \
     -H "Authorization: Bearer <token>"
```

## List execution events

```
GET /history/{task_history_id}/events
```

Returns lifecycle events from the executor in chronological order. Each event has a `timestamp`, `type`, `description`, and optional `step` name. Useful for tracing what happened during a run step by step.

## List output files

```
GET /history/{task_history_id}/files/
```

Lists files written by the task, with their size and whether they are directories.

## Download an output file

```
GET /history/{task_history_id}/file/?path=<path>
```

Streams a specific file from the task's output. Use the `path` value returned by [List output files](#list-output-files).

## Get task statistics

```
GET /stats/{task}
```

Returns aggregate statistics for all runs of a task: total run count, counts by status, duration breakdown, and the last finished timestamp.

**Example:**

```shell
curl -sk https://<pmm-server>/sep/stats/mysql-backup-xtrabackup \
     -H "Authorization: Bearer <token>"
```

## Stop a run

```
POST /history/{task_history_id}/stop/
```

Stops a running task. Returns the updated history record with `status: stopped`.

**Example:**

```shell
curl -sk -X POST https://<pmm-server>/sep/history/42/stop/ \
     -H "Authorization: Bearer <token>"
```
