---
title: Overview
slug: access-control
categorySlug: access-control-api
hidden: 0
---

Access Control in PMM can be used to restrict access to individual metrics.  

Access Control is currently in Technical Preview. To use this feature, enable it manually in PMM's settings.

Once enabled, restricting access to metrics can be performed by:

1. Creating a Percona role
2. Assigning the role to a user

### Create a Percona role

```shell
curl -X POST -k "http://127.0.0.1/v1/accesscontrol/roles" \
     --header "Authorization: Basic XXXXX" \
     --header "Content-Type: application/json" \
     --data '{
        "title": "My custom role",
        "filter": "{environment=\"staging\"}"
      }'
```

The `filter` parameter is a [PromQL query](https://prometheus.io/docs/prometheus/latest/querying/basics/) restricting access to the specified metrics.  
Full access can be provided by specifying an empty `filter` field.

### Assign a Percona role

Users can be assigned roles by using the `/v1/accesscontrol/roles:assign` API.  
The endpoint assigns new roles to a user. Other roles, that may have been assigned to the user previously, stay intact.

```shell
curl -X POST -k "http://127.0.0.1/v1/accesscontrol/roles:assign" \
     --header "Authorization: Basic XXXXX" \
     --header "Content-Type: application/json" \
     --data '{
        "user_id": 1,
        "role_ids": [2, 3, 4]
      }'
```
