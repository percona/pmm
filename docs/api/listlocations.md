---
slug: 'listlocations'
---

## List Locations

The following API call will list all the available Locations:

```bash
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
  --url https://127.0.0.1/v1/management/backup/Locations/List \
```

You will need the [authetication token](ref:authentication).
