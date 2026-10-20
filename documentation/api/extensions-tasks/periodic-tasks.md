---
title: Schedule recurring tasks
slug: extensions-list-periodic-tasks
category:
  uri: extensions-tasks-api
position: 3
---

Periodic tasks run a task automatically on a cron or interval schedule.

## List schedules

```
GET /extensions/api/extensions/periodic-tasks/
```

Returns a paginated list of all periodic tasks.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page |

## Create a schedule

```
POST /extensions/api/extensions/periodic-tasks/{task_name}/
```

Creates a recurring schedule for the named task. Requires the **Admin** role. Use either `interval` or `crontab` to set the recurrence, not both. Returns HTTP 201 on success.

**Request body:**

```json
{
  "name": "",
  "enabled": true,
  "description": "Nightly XtraBackup",
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
| `enabled` | boolean | Whether the schedule is active. Default: `true`. |
| `description` | string | A label for this schedule. |
| `interval` | object | Interval schedule: `{"every": 1, "period": "days"}`. Period options: `days`, `hours`, `minutes`. |
| `crontab` | object | Crontab schedule with `minute`, `hour`, `day_of_week`, `day_of_month`, `month_of_year`, and `timezone`. |
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
PUT /extensions/api/extensions/periodic-tasks/{periodic_task_id}/
```

Updates an existing schedule. Requires the **Admin** role. Send the full schedule object with your changes, using the same fields as the create request. Use the numeric ID returned when you created the schedule as `{periodic_task_id}`.

## Delete a schedule

```
DELETE /extensions/api/extensions/periodic-tasks/{periodic_task_id}
```

Deletes the periodic task. Requires the **Admin** role. Returns HTTP 204.

## Preview a schedule

```
POST /extensions/api/extensions/periodic-tasks/schedule/preview/
```

Returns the next N fire times for a given crontab or interval, without creating a schedule. Useful for validating a schedule before saving it.
