---
slug: 'adhoc-backup'
---

## Ad hoc Backup

PMM can backup the monitored servers. 

This section describes making ad hoc backups from a service.


### Creating a Backup

Here is an example of a Curl API call to create a backup:


```bash
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

You require an authentication string which is described [here](ref:authentication).

Also, you require the [service_id](ref:listservices) and [location_id](ref:listlocations).

You can choose a `name` and `description` for the backup. You can also configure `retry_interval` and `retries` if required. 

