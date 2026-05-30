---
title: Scheduled Backups
slug: scheduled-backups
category:
  uri: backup-api
---

PMM can run backups on a recurring schedule using cron expressions. This section describes how to manage scheduled backups.

### Schedule a Backup

Use the following API call to schedule a recurring backup:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups:schedule \
     --data '
{
  "service_id": "2c756c17-e4cd-4180-a3d4-d7a3fe1e4816",
  "location_id": "0bd7b27d-e54e-4299-a0e2-3fe9990e635d",
  "folder": "my-service-backups",
  "cron_expression": "0 2 * * *",
  "name": "Nightly Backup",
  "description": "Automated nightly backup",
  "enabled": true,
  "retries": 2,
  "retry_interval": "60s",
  "mode": "BACKUP_MODE_SNAPSHOT",
  "data_model": "DATA_MODEL_LOGICAL",
  "retention": 7
}
'
```

You require an authentication token, which is described [here](ref:authentication).

Also, you require the [service_id](ref:listservices) and [location_id](ref:listlocations).

Key parameters:

- `cron_expression` - How often the backup runs, in [cron format](https://en.wikipedia.org/wiki/Cron).
- `mode` - Backup mode:
  - `BACKUP_MODE_SNAPSHOT` - A full point-in-time backup of the database.
  - `BACKUP_MODE_INCREMENTAL` - Only the changes since the last backup are stored, reducing storage usage.
  - `BACKUP_MODE_PITR` - Continuously streams write-ahead or oplog data, enabling restore to any point in time.
- `data_model` - Data model: `DATA_MODEL_LOGICAL` or `DATA_MODEL_PHYSICAL`.
- `retention` - How many artifacts to keep. Set to `0` for unlimited retention.
- `folder` - Optional folder path within the backup location to store artifacts.

### List Scheduled Backups

Use the following API call to retrieve all scheduled backups:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/scheduled
```

The response includes details for each scheduled backup such as its ID, cron expression, last and next run times, and current enabled state.

### Change a Scheduled Backup

Use the following API call to update an existing scheduled backup. Only the fields you include in the request body are updated:

```shell
curl --insecure -X PUT \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups:changeScheduled \
     --data '
{
  "scheduled_backup_id": "7e8a5f20-9c3b-11ee-b9d1-0242ac120002",
  "enabled": false,
  "cron_expression": "0 3 * * *",
  "retention": 14
}
'
```

### Remove a Scheduled Backup

Use the following API call to remove a scheduled backup:

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --url https://127.0.0.1/v1/backups/7e8a5f20-9c3b-11ee-b9d1-0242ac120002
```

Removing a scheduled backup only stops future executions. It does not delete any existing backup artifacts created by previous runs.
