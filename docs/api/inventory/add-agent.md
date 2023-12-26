---
title: Add an Agent
slug: addagent
category: 626de009b977e3003179f7dd
---

## Add an Agent

This section describes how to add an Agent of any type to PMM Inventory.

In PMM versions prior to 3.0.0, we featured a separate API call for each Agent type. Starting with PMM 3.0.0, we offer a single API endpoint for all Agent types. While previously the Agent type was defined by the endpoint, i.e. `Agents/MySQLdExporter`, now the Agent type must be specified as the top-level property of the request payload. Along with this single API endpoint, we are deprecating individual API endpoints for each Agent type.

Let's see how to add an Agent of type `mysql` using the old and new API calls.

Old API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Agents/MySQLdExporter \
	--data '
{
  "pmm_agent_id": "pmm-server",
  "service_id": "/service_id/13519ec9-eedc-4d21-868c-582e146e1d0e",
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

New API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Agents/Add \
	--data '
{
  "mysqld_exporter": {
    "pmm_agent_id": "pmm-server",
    "service_id": "/service_id/13519ec9-eedc-4d21-868c-582e146e1d0e",
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


To get the authentication token, please visit [this page](ref:authentication).
