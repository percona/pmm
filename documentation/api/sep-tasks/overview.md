---
title: Overview
slug: sep-tasks-overview
category:
  uri: sep-tasks-api
position: 0
---

The SEP Tasks API gives you programmatic control over database operations available in the PMM UI under **Apps**.

Use it to:

- [execute](ref:sep-execute-task) database operations (backups, diagnostics, schema changes) on your hosts
- [monitor runs](ref:sep-task-history): check status, stream logs, and browse output files
- [stop](ref:sep-task-history) a running task
- [schedule recurring operations](ref:sep-list-periodic-tasks) with cron or interval schedules
- [verify connectivity](ref:sep-connectivity-check) between an executor host and a database before running a task

## Concepts

**Task**: a named operation template registered in SEP. PMM ships a set of built-in tasks with stable identifiers like `mysql-backup-xtrabackup`.

**Task history**: a single execution of a task. Each run produces a history record with a status, timestamps, logs, and output files.

**Periodic task**: a schedule that triggers a task automatically on a cron or interval cadence.

**Executor host**: the PMM Client node where the task runs. For XtraBackup, this must be the database host itself. For Mydumper and Binlog, any host with network access to the database works.

## Base URL

The SEP Tasks API is proxied through PMM Server at:

```
https://<pmm-server>/sep/
```

All endpoints in this reference are relative to that base.

## Authentication

Use the same PMM service account token you use for the PMM REST API. Pass it as a Bearer token:

```shell
curl -sk https://<pmm-server>/sep/execute/mysql-backup-xtrabackup \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{...}'
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
