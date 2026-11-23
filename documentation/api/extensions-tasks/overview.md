---
title: Overview
slug: extensions-tasks-overview
category:
  uri: extensions-tasks-api
position: 0
---

The PMM Extensions Tasks API gives you programmatic control over database operations available in the PMM UI under **Apps**.

Use it to:

- [execute](ref:extensions-execute-task) database operations on your hosts
- [monitor runs](ref:extensions-task-history): list status and stream logs
- [stop](ref:extensions-task-history) a running task
- [schedule recurring operations](ref:extensions-list-periodic-tasks) with cron or interval schedules

## Concepts

**Task**: a named operation you create in an app. For example, each backup you configure in the MySQL Backups app is a task, identified by the task name you enter.

**Task run**: a single execution of a task. Each run produces a task history record with a status, timestamps, and logs.

**Periodic task**: a schedule that triggers a task on a cron or interval cadence.

**Executor host**: the PMM Client node where the task runs. For XtraBackup, this must be the database host itself. For Mydumper and Binlog, any host with network access to the database works.

## Base URL

The PMM Extensions API is served through PMM Server at:

```
https://<pmm-server>/extensions
```

Endpoint paths in this reference are full paths relative to the server root. To form a complete request URL, prepend `https://<pmm-server>` to the path shown.

## Authentication

Use the same PMM service account token you use for the PMM REST API. Pass it as a Bearer token:

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/apps/mysql_backups/<your_task_name>/execute \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{}'
```

To generate a service account token, see [Authentication](ref:authentication) in the PMM API welcome section.

All `POST`, `PUT`, and `DELETE` requests require a token with the **Admin** role. Tokens with the Viewer or Editor role get `403 Forbidden`, and requests without a token get `401 Unauthorized`. `GET` requests work with any role.

## Task statuses

| Status | Meaning |
|--------|---------|
| `pending` | Queued, not yet picked up by the executor |
| `running` | Currently executing on the target host |
| `success` | Completed successfully |
| `failed` | Completed with an error |
| `stopped` | Manually stopped via the API or UI |
| `lost` | The executor lost track of the run |
| `stale` | Skipped: the executor never started the run within the staleness threshold |
| `unlaunchable` | The executor node could not find a required command |
