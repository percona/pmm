---
title: Overview
slug: extensions-tasks-overview
category:
  uri: extensions-tasks-api
position: 0
---

The PMM Extensions Tasks API gives you programmatic control over database operations available in the PMM UI under **Apps**.

Use it to:

- [execute](ref:extensions-execute-task) database operations (backups, diagnostics, schema changes) on your hosts
- [monitor runs](ref:extensions-task-history): check status, stream logs, and browse output files
- [stop](ref:extensions-task-history) a running task
- [schedule recurring operations](ref:extensions-list-periodic-tasks) with cron or interval schedules
- [verify connectivity](ref:extensions-connectivity-check) between an executor host and a database before running a task

## Concepts

**Task**: a named operation template registered in PMM Extensions. PMM ships a set of built-in tasks with stable identifiers like `mysql-backup-xtrabackup`.

**Task history**: a single execution of a task. Each run produces a history record with a status, timestamps, logs, and output files.

**Periodic task**: a schedule that triggers a task automatically on a cron or interval cadence.

**Executor host**: the PMM Client node where the task runs. For XtraBackup, this must be the database host itself. For Mydumper and Binlog, any host with network access to the database works.

## Base URL

The PMM Extensions Tasks API is proxied through PMM Server at:

```
https://<pmm-server>/extensions/
```

All endpoints in this reference are relative to that base.

## Authentication

Use the same PMM service account token you use for the PMM REST API. Pass it as a Bearer token:

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/apps/mysql-backups/mysql-backup-xtrabackup/execute \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{}'
```

To generate a service account token, see [Authentication](ref:authentication) in the PMM API welcome section.

## Task statuses

| Status | Meaning |
|--------|---------|
| `pending` | Queued, not yet picked up by the executor |
| `running` | Currently executing on the target host |
| `success` | Completed successfully |
| `failed` | Completed with an error |
| `stopped` | Manually stopped via the API or UI |
| `lost` | The executor lost track of the run (for example, the Nomad allocation disappeared) |
| `stale` | Skipped. The executor never placed the run within the staleness threshold |
| `unlaunchable` | The executor node could not find a required command. The payload never ran |
