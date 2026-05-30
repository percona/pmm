---
title: Restore History
slug: listrestores
category:
  uri: backup-api
---

PMM keeps a history of all backup restore operations. This section describes how to list the restore history and retrieve restore logs.

### List Restore History

Use the following API call to list all backup restore history items:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/restores
```

You require an authentication token, which is described [here](ref:authentication).

The response includes details for each restore operation, such as its ID, the artifact used, the target service, backup location, data model, restore status, start and finish times, and the PITR timestamp if applicable.

Possible restore status values are:

- `RESTORE_STATUS_IN_PROGRESS` - The restore is currently running.
- `RESTORE_STATUS_SUCCESS` - The restore completed successfully.
- `RESTORE_STATUS_ERROR` - The restore failed.

### Get Restore Logs

Use the following API call to retrieve the logs from the underlying restore tool for a given restore operation:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url 'https://127.0.0.1/v1/backups/restores/a1b2c3d4-1234-5678-abcd-ef0123456789/logs?offset=0&limit=100'
```

Logs are returned as ordered chunks. Use `offset` and `limit` to paginate through large log outputs. The `end` field in the response indicates whether the log stream has ended.
