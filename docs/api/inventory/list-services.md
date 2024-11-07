---
title: List Services
slug: listservices
categorySlug: inventory-api
---

## List Services

The following API call lists the available services on a Node:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/services?node_id=7d07a712-7fb9-4265-8a7d-a0db8aa35762&service_type=SERVICE_TYPE_MYSQL_SERVICE
```

To get the authentication token, check [Authentication](ref:authentication).

If you need to know the `node_id`, refer to [List Nodes endpoint](ref:listnodes).

Choose the `service_type` that you want to list. The options are:

- SERVICE_TYPE_MYSQL_SERVICE
- SERVICE_TYPE_MONGODB_SERVICE
- SERVICE_TYPE_POSTGRESQL_SERVICE
- SERVICE_TYPE_PROXYSQL_SERVICE
- SERVICE_TYPE_HAPROXY_SERVICE
- SERVICE_TYPE_EXTERNAL_SERVICE

If you prefer to get all services running on the node, you can omit the `service_type` parameter.

Otherwise, calling the same endpoint with no parameters will return all services known to this node.
