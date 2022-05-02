---
slug: 'listlocations'
---

## List Locations

The following Curl API call will list all the available Locations:

```bash
curl --insecure -X POST -H 'Authorization: Bearer XXXXX' \
     --request POST \
     --url https://127.0.0.1/v1/management/backup/Locations/List \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json'
```

You will need the [authetication string](ref:authentication).