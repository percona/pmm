---
title: Add an Agent
slug: addagent
categorySlug: inventory-api
---

## Add an Agent

This section describes how to add an Agent of any type to PMM Inventory.

In PMM versions prior to 3.0.0, we featured a separate API call for each Agent type. Starting with PMM 3.0.0, we have streamlined the process by offering a single API endpoint for all Agent types. 

Previously, the Agent type was defined by the endpoint, i.e. `Agents/MySQLdExporter`. In the new approach, the Agent type must be specified as the top-level property of the request payload. As part of this single API endpoint update, we have also deprecated individual API endpoints for each Agent type.

Here's how to add an Agent of type `mysql` using the old and new API calls.

**Old API call**

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/Agents/MySQLdExporter \
     --data '
{
  "pmm_agent_id": "pmm-server",
  "service_id": "13519ec9-eedc-4d21-868c-582e146e1d0e",
  "username":  "mysql-prod-user",
  "password":  "mysql-prod-pass",
  "listen_port": 33060,
  "custom_labels": {
    "department":  "sales",
    "environment": "sales-prod",
    "replication_set": "db-sales-prod-1-rs1",
    "cluster": "db-sales-prod-1"
  }
}
'
```

**New API call**

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/agents \
     --data '
{
  "mysqld_exporter": {
    "pmm_agent_id": "pmm-server",
    "service_id": "13519ec9-eedc-4d21-868c-582e146e1d0e",
    "username":  "mysql-prod-user",
    "password":  "mysql-prod-pass",
    "listen_port": 33060,
    "custom_labels": {
      "department":  "sales",
      "environment": "sales-prod",
      "replication_set": "db-sales-prod-1-rs1",
      "cluster": "db-sales-prod-1"
    }
  }
}
'
```

You can choose from the following Agent types:

- pmm_agent
- node_exporter
- mysqld_exporter
- mongodb_exporter
- postgres_exporter
- proxysql_exporter
- external_exporter
- rds_exporter
- azure_database_exporter
- qan_mysql_perfschema_agent
- qan_mysql_slowlog_agent
- qan_mongodb_profiler_agent
- qan_postgresql_pgstatements_agent
- qan_postgresql_pgstatmonitor_agent

To get the authentication token, check [Authentication](ref:authentication).
