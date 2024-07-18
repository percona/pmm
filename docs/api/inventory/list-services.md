---
title: List Services
slug: listservices
category: 626de009b977e3003179f7dd
---

## List Services

The following API call lists the available services on a Node:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Services/List \
  --data '
{
  "node_id": "7d07a712-7fb9-4265-8a7d-a0db8aa35762",
  "service_type": "MYSQL_SERVICE"
}'
```

First, get the [authentication token](ref:authentication).

Then, you need to know the [node_id](ref:listnodes).

Choose the `service_type` that you want to list. The options are:

- MYSQL_SERVICE
- MONGODB_SERVICE
- POSTGRESQL_SERVICE
- PROXYSQL_SERVICE
- HAPROXY_SERVICE
- EXTERNAL_SERVICE

If you prefer to get all services running on the node, you can omit the `service_type` parameter.

Otherwise, calling the same endpoint with no parameters will return all services known to this PMM instance.
