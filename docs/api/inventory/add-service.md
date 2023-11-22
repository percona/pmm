---
title: Add a Service
slug: addservice
category: 626de009b977e3003179f7dd
---

## Add a Service

This section describes how to add a Service of any type to PMM Inventory.

In PMM versions prior to 3.0.0, we featured a separate API call for each Service type. Starting with PMM 3.0.0, we offer a single API endpoint for all Service types. While previously the Service type was defined by the endpoint, i.e. `Services/AddMySQL`, now the Service type must be specified as the top-level property of the request payload. Along with this single API endpoint, we are deprecating the separate API calls for each Service type.

Let's see how to add a Node of type `mysql` using the old and new API calls.

Old API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
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

New API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Services/Add \
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

To get the authentication token, please visit [this page](ref:authentication).
