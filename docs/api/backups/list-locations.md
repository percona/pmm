---
title: List Locations
slug: listlocations
excerpt: ListLocations returns a list of all backup locations.
category: 66aa56507e69ed004a736efe
---

The following API call will list all the available backup locations:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
  --url https://127.0.0.1/v1/management/backup/Locations/List \
```

Please note that locations are good for storing any type of backup disregarding the technology or the database vendor. However, for a better organization of your file system storage, you'll probably want to create different locations based on a certain criteria. For example, it can be a department, a region, etc.

You will need the [authetication token](ref:authentication).
