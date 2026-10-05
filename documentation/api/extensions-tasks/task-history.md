---
title: Monitor task runs
slug: extensions-task-history
category:
  uri: extensions-tasks-api
position: 2
---

After [executing a task](ref:extensions-execute-task), use these endpoints to list runs, stream logs, and stop a run in progress.

## List runs

```
GET /extensions/api/extensions/task-history/
```

Returns a paginated list of task history records, newest first.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `task_names` | string (repeatable) | none | Filter to one or more task names. Repeat the parameter for multiple names. Omit to list all history. |
| `status` | string | none | Filter by status: `pending`, `running`, `success`, `failed`, `stopped`, `lost`, `stale`, `unlaunchable` |
| `exclude_internal` | boolean | `false` | Exclude internal maintenance tasks |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page |

**Example — list failed runs for a task:**

```shell
curl -sk "https://<pmm-server>/extensions/api/extensions/task-history/?task_names=mysql-backup-xtrabackup&status=failed" \
     -H "Authorization: Bearer <token>"
```

## Stop a run

```
POST /extensions/api/extensions/task-history/{task_history_id}/stop/
```

Stops a running task. Returns the updated history record with `status: stopped`. Returns HTTP 400 if the task is not running.

**Example:**

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/extensions/task-history/42/stop/ \
     -H "Authorization: Bearer <token>"
```

## Get task statistics

```
GET /extensions/api/extensions/task-stats/{task_name}
```

Returns aggregate statistics for all runs of a task: total count, counts by status, and the last finished timestamp.

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/api/extensions/task-stats/mysql-backup-xtrabackup \
     -H "Authorization: Bearer <token>"
```

## Stream logs

```
GET /extensions/stream-logs/{task_history_id}
```

Streams log output as [server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events). While the task is running, events arrive live. For finished tasks, stored logs are replayed and then a `finish` event is emitted.

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/stream-logs/42 \
     -H "Authorization: Bearer <token>"
```

## Stream execution events

```
GET /extensions/stream-logs/{task_history_id}/execution-events
```

Streams lifecycle events (start, step transitions, completion) as server-sent events. Useful for tracking which step of a multi-step task is currently running.
