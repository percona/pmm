---
title: Make a backup
slug: startbackup
excerpt: This endpoint allows to make an unscheduled, or ad-hoc, backup of a given service.
categorySlug: backup-api
---

PMM can backup the monitored services.

This section describes how to make an ad-hoc backup of a service.

### Creating a Backup

Here is an example of an API call to create a backup:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups:start \
     --data '
{
     "service_id": "2c756c17-e4cd-4180-a3d4-d7a3fe1e4816",
     "location_id": "0bd7b27d-e54e-4299-a0e2-3fe9990e635d",
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
