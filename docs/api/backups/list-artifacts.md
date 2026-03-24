---
title: Manage Backup Artifacts
slug: manage-backup-artifacts
category:
  uri: backup-api
---

Backup artifacts are the backup files created by PMM backup operations. This section describes how to list, delete, and work with backup artifacts.

### List Artifacts

Use the following API call to list all backup artifacts:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/artifacts
```

You require an authentication token, which is described [here](ref:authentication).

The response includes details for each artifact, such as its ID, name, database vendor, associated service, backup location, data model, backup mode, status, and creation time.

### Delete an Artifact

Use the following API call to delete a backup artifact:

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --url https://127.0.0.1/v1/backups/artifacts/ff582c9d-49ea-437c-9f3a-362c57e7ad38
```

To also remove the backup files from storage, add `?remove_files=true`:

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/backups/artifacts/ff582c9d-49ea-437c-9f3a-362c57e7ad38?remove_files=true'
```

### List Compatible Services for a Backup Artifact

Before restoring a backup, you can check which services are compatible with a given artifact:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/ff582c9d-49ea-437c-9f3a-362c57e7ad38/compatible-services
```

The response lists the services that are compatible with the artifact and can be used as a restore target.

### List PITR Timeranges

For MongoDB backups taken with Point-in-Time Recovery (PITR) enabled, you can retrieve the available recovery timeranges:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/artifacts/ff582c9d-49ea-437c-9f3a-362c57e7ad38/pitr-timeranges
```

The response returns a list of timeranges, each with a `start_timestamp` and `end_timestamp`. You can use any timestamp within one of these ranges as the `pitr_timestamp` in a [restore request](ref:restorebackup).

### Get Backup Logs

Use the following API call to retrieve the logs from the underlying backup tool for a given backup artifact:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url 'https://127.0.0.1/v1/backups/ff582c9d-49ea-437c-9f3a-362c57e7ad38/logs?offset=0&limit=100'
```

Logs are returned as ordered chunks. Use `offset` and `limit` to paginate through large log outputs. The `end` field in the response indicates whether the log stream has ended.
