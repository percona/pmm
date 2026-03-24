---
title: Manage Backup Locations
slug: manage-backup-locations
category:
  uri: backup-api
---

Backup locations define where backup artifacts are physically stored. PMM supports two types of backup locations:

- **Filesystem** - A local or network-mounted filesystem path accessible by the PMM Server.
- **S3** - An S3-compatible object storage bucket.

### Add a Backup Location

Use the following API call to add a new backup location backed by an S3-compatible storage:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups/locations \
     --data '
{
  "name": "my-s3-location",
  "description": "S3 bucket for production backups",
  "s3_config": {
    "endpoint": "s3.amazonaws.com",
    "access_key": "AKIAIOSFODNN7EXAMPLE",
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "bucket_name": "my-pmm-backups"
  }
}
'
```

To add a filesystem-backed location, use `filesystem_config` instead:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups/locations \
     --data '
{
  "name": "my-local-location",
  "description": "Local filesystem backup storage",
  "filesystem_config": {
    "path": "/srv/backup"
  }
}
'
```

You require an authentication token, which is described [here](ref:authentication).

### Change a Backup Location

Use the following API call to update an existing backup location. You need the `location_id` returned when you created the location, or listed via [List Backup Locations](ref:listlocations):

```shell
curl --insecure -X PUT \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups/locations/0bd7b27d-e54e-4299-a0e2-3fe9990e635d \
     --data '
{
  "location_id": "0bd7b27d-e54e-4299-a0e2-3fe9990e635d",
  "name": "updated-location-name",
  "description": "Updated description",
  "s3_config": {
    "endpoint": "s3.amazonaws.com",
    "access_key": "AKIAIOSFODNN7EXAMPLE",
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "bucket_name": "my-pmm-backups-updated"
  }
}
'
```

### Remove a Backup Location

Use the following API call to remove an existing backup location:

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --url https://127.0.0.1/v1/backups/locations/0bd7b27d-e54e-4299-a0e2-3fe9990e635d
```

To force-remove a location even if it has associated artifacts, add `?force=true` to the URL:

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/backups/locations/0bd7b27d-e54e-4299-a0e2-3fe9990e635d?force=true'
```

### Test a Backup Location

Before using a backup location, you can verify that the configuration and credentials are correct:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/backups/locations:testConfig \
     --data '
{
  "s3_config": {
    "endpoint": "s3.amazonaws.com",
    "access_key": "AKIAIOSFODNN7EXAMPLE",
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "bucket_name": "my-pmm-backups"
  }
}
'
```

A successful response returns an empty JSON object `{}`. Any connectivity or credential issues are reported as an error.
