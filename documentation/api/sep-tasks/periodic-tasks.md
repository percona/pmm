---
title: Schedule recurring tasks
slug: sep-list-periodic-tasks
category:
  uri: sep-tasks-api
position: 3
---

Periodic tasks run a task automatically on a cron or interval schedule. Each periodic task is linked to a named task and carries an execution request with the same `meta`, `payload`, and other fields you pass when executing manually.

## List all schedules

```
GET /periodic/
```

Returns a paginated list of all periodic tasks managed by SEP.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `owner` | string | none | Filter to schedules owned by this app |
| `enabled` | boolean | none | Filter by enabled state |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |

## List schedules for a specific task

```
GET /{task_name}/periodic/
```

Returns all periodic tasks associated with the named task.

## Get a schedule

```
GET /periodic/{periodic_task_id}
```

Returns a single periodic task by its numeric ID.

## Create a schedule

```
POST /{task_name}/periodic/
```

Creates a new periodic task for the named task. Supply either an `interval` or a `crontab` schedule, but not both.

**Request body:**

```json
{
  "name": "",
  "task": "mysql-backup-xtrabackup",
  "enabled": true,
  "description": "Nightly XtraBackup",
  "execute_request": {
    "meta": {
      "_target": "db-host-01",
      "_backup_dir": "/backups/mysql"
    }
  },
  "crontab": {
    "minute": "0",
    "hour": "2",
    "day_of_week": "*",
    "day_of_month": "*",
    "month_of_year": "*",
    "timezone": "UTC"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Schedule name. Leave empty to auto-generate. |
| `task` | string | The task name this schedule will run. |
| `enabled` | boolean | Whether the schedule is active. Default: `true`. |
| `description` | string | A label for this schedule. |
| `execute_request` | object | The execution parameters passed to the task on each run. Same fields as the [execute endpoint](ref:sep-execute-task) request body. |
| `interval` | object | Interval schedule: `{"every": 1, "period": "days"}`. Period options: `days`, `hours`, `minutes`, `seconds`. |
| `crontab` | object | Crontab schedule with `minute`, `hour`, `day_of_week`, `day_of_month`, `month_of_year`, and `timezone` fields. |
| `start_time` | datetime | Earliest time the schedule can fire. |

**Interval example (every 24 hours):**

```json
{
  "interval": {
    "every": 24,
    "period": "hours"
  }
}
```

**Crontab example (every day at 02:00 UTC):**

```json
{
  "crontab": {
    "minute": "0",
    "hour": "2",
    "day_of_week": "*",
    "day_of_month": "*",
    "month_of_year": "*",
    "timezone": "UTC"
  }
}
```

## Update a schedule

```
PUT /periodic/{periodic_task_id}
```

Replaces the schedule configuration. Supply the full updated object with the same fields as the create request.

## Delete a schedule

```
DELETE /periodic/{periodic_task_id}
```

Deletes the periodic task. Returns HTTP 204 on success.
