---
title: Add a Service
slug: addservice
categorySlug: inventory-api
---

## Add a Service

This section describes how to add a Service of any type to PMM Inventory.

In PMM versions prior to 3.0.0, we featured a separate API call for each Service type. Starting with PMM 3.0.0, we have streamlined the process by offering single API endpoint for all Service types. 

Previously, the Service type was defined by the endpoint, i.e. `Services/AddMySQL`. In the new approach, the Service type must be specified as the top-level property of the request payload. As part of the single API endpoint update, we have deprecated individual API endpoints for each Service type.

Here's how to add a Node of type `mysql` using the old and the new API calls:

**Old API call**:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/Services/AddMySQL \
     --data '
{
  "service_name": "mysql-sales-db-prod-1",
  "node_id": "pmm-server",
  "address":  "209.0.25.100",
  "port": 3306,
  "environment": "sales-prod",
  "cluster": "db-sales-prod-1",
  "replication_set": "db-sales-prod-1-rs1",
  "custom_labels": {
    "department":  "sales"
  }
}
'
```

**New API call**:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/services \
     --data '
{
  "mysql": {
    "service_name": "mysql-sales-db-prod-1",
    "node_id": "pmm-server",
    "address":  "209.0.25.100",
    "port": 3306,
    "environment": "sales-prod",
    "cluster": "db-sales-prod-1",
    "replication_set": "db-sales-prod-1-rs1",
    "custom_labels": {
      "department":  "sales"
    }
  }
}
'
```

You can choose from the following Service types:

- mysql
- mongodb
- postgresql
- proxysql
- haproxy
- external

To get the authentication token, check [Authentication](ref:authentication).
