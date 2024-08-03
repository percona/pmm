---
title: Make a backup
slug: startbackup
excerpt: This endpoint allows to make an unscheduled, or ad-hoc, backup of a given service.
category: 66aa56507e69ed004a736efe
---

PMM can backup the monitored services.

This section describes how to make an ad-hoc backup of a service.

### Creating a Backup

Here is an example of an API call to create a backup:

```shell
curl --insecure -X POST -H 'Authorization: Bearer XXXXX' \
     --request POST \
     --url https://127.0.0.1/v1/management/backup/Backups/Start \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json' \
     --data '
{
     "service_id": "/service_id/XXXXX",
     "location_id": "/location_id/XXXXX",
     "name": "Test Backup",
     "description": "Test Backup",
     "retry_interval": "60s",
     "retries": 1
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
- ERROR_CODE_INCOMPATIBLE_XTRABACKUP - xtrabackup is not compatible with MySQL to make a backup
