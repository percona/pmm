---
slug: 'adhoc-backup'
---

## Adhoc Backup

PMM is able to backup the monitored servers. 

In this section we will show you how can you take Adhoc Backups from a service.


### Creating a Backup

Here is an example Curl API call to create a backup:

```
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

You need the authentication string which is described [here](ref:authentication).

Then you need the [service_id](ref:listservices) and [location_id](ref:location_id).

You can choose your own `name` and `description` for the backup. You can also configure `retry_interval` and `retries` if that is necessary. 

