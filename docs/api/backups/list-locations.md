---
title: List Locations
slug: listlocations
excerpt: ListLocations returns a list of all backup locations.
categorySlug: backup-api
---

The following API call will list all the available backup locations:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/backups/locations
```

Please note that locations are good for storing any type of backup disregarding the technology or the database vendor. However, for a better organization of your file system storage, you'll probably want to create different locations based on a certain criteria. For example, it can be a department, a region, etc.

You require an authentication token, which is described [here](ref:authentication).
