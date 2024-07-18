---
title: Restore from a backup
slug: restorebackup
excerpt: This endpoint allows to restore a database from a previously made backup.
category: 626badcabbc59c02acc1a540
order: 1
---

PMM can backup the monitored services.

This section describes making ad-hoc backups from a service.

### Restoring From a Backup

Here is an example of an API call to restore from a backup:

```shell
curl --insecure -X POST -H 'Authorization: Bearer XXXXX' \
     --request POST \
     --url https://127.0.0.1/v1/management/backup/Backups/Restore \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json' \
     --data '
{
     "service_id": "40499c38-522d-4ed1-ab3f-8a099684f46d",
     "artifact_id": "ff582c9d-49ea-437c-9f3a-362c57e7ad38",
     "pitr_timestamp": "2023-09-09T10:02:25.998"
}
'
```

You require an authentication token, which is described [here](ref:authentication).

Also, you require the [service_id](ref:listservices) and [location_id](ref:listlocations).

You can defined a `name` and a `description` for each backup. You can also configure `retry_interval` and `retries` if required.

### Error messages

The API call could return an error message in the details, containing a specific ErrorCode indicating the failure reason:
- ERROR_CODE_XTRABACKUP_NOT_INSTALLED - xtrabackup is not installed on the service
- ERROR_CODE_INVALID_XTRABACKUP - different versions of xtrabackup and xbcloud
- ERROR_CODE_INCOMPATIBLE_XTRABACKUP - xtrabackup is not compatible with MySQL for making a backup
- ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL - target MySQL version is not compatible with the artifact to perform a restore of the backup
