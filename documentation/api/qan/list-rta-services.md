---
title: List services
slug: list-rta-services
content:
  excerpt: Retrieve services that support Real-time Analytics.
category:
  uri: rta-api
---

## List services

`GET /v1/realtimeanalytics/services`

Returns a list of services that support Real-time Analytics. Use this endpoint to discover which services can be monitored with RTA before starting a session.

This endpoint only returns services whose PMM Agent is new enough to run the RTA collector for that database: **3.7.0 or later for MongoDB**, **3.10.0 or later for MySQL and PostgreSQL**. Services monitored by older PMM Agents won't appear in the results, even if they're registered in PMM. PMM Server's own PostgreSQL database (`pmm-server-postgresql`) is never returned.

### Query parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_type` | string | No | Filter by service type. Default: `SERVICE_TYPE_UNSPECIFIED` (returns all supported types) |


#### Supported service types

| Value | Description |
|-------|-------------|
| `SERVICE_TYPE_UNSPECIFIED` | Returns all supported service types (default) |
| `SERVICE_TYPE_MONGODB_SERVICE` | MongoDB services |
| `SERVICE_TYPE_MYSQL_SERVICE` | MySQL services, including Percona Server for MySQL and MariaDB |
| `SERVICE_TYPE_POSTGRESQL_SERVICE` | PostgreSQL services |

Other service types (Valkey, ProxySQL, HAProxy, External) return an error. Support for additional database types is planned for future releases.

> 📘 Info
> 
> MariaDB and Percona Server for MySQL are registered as MySQL services and are returned under `SERVICE_TYPE_MYSQL_SERVICE`. There is no separate service type for either.

### Response

The response contains one array per supported service type. An array is present but empty when no services of that type are RTA-compatible.

```json
{
  "mongodb": [
    {
      "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "service_name": "mongodb-production-rs0",
      "node_id": "pmm-server",
      "address": "mongodb-rs0.example.com",
      "port": 27017,
      "environment": "production",
      "cluster": "production-cluster",
      "replication_set": "rs0",
      "custom_labels": {
        "team": "backend"
      },
      "version": "7.0.5"
    }
  ],
  "mysql": [
    {
      "service_id": "89008765-c771-44a9-a9a5-e1c5ff51fc36",
      "service_name": "mariadb-production-01",
      "node_id": "pmm-server",
      "address": "mariadb-01.example.com",
      "port": 3306,
      "socket": "",
      "environment": "production",
      "cluster": "production-cluster",
      "replication_set": "",
      "custom_labels": {
        "team": "backend"
      },
      "version": "",
      "extra_dsn_params": {}
    }
  ],
  "postgresql": [
    {
      "service_id": "39c8e124-284a-4871-b9c0-32d6708dee4a",
      "service_name": "postgresql-production-01",
      "database_name": "postgres",
      "node_id": "pmm-server",
      "address": "pg-01.example.com",
      "port": 5432,
      "socket": "",
      "environment": "production",
      "cluster": "production-cluster",
      "replication_set": "",
      "custom_labels": {
        "team": "backend"
      },
      "version": "17.6",
      "auto_discovery_limit": 0
    }
  ]
}
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `mongodb` | array | List of MongoDB services supporting RTA |
| `mongodb[].service_id` | string | Unique service identifier (use this to start sessions) |
| `mongodb[].service_name` | string | User-defined service name |
| `mongodb[].node_id` | string | Node identifier where the service runs |
| `mongodb[].address` | string | Access address (DNS name or IP) |
| `mongodb[].port` | integer | Access port |
| `mongodb[].socket` | string | Access unix socket (alternative to address/port) |
| `mongodb[].environment` | string | Environment name |
| `mongodb[].cluster` | string | Cluster name |
| `mongodb[].replication_set` | string | Replication set name |
| `mongodb[].custom_labels` | object | Custom user-assigned labels |
| `mongodb[].version` | string | MongoDB version |
| `mysql` | array | List of MySQL services supporting RTA, including Percona Server for MySQL and MariaDB |
| `mysql[].service_id` | string | Unique service identifier (use this to start sessions) |
| `mysql[].service_name` | string | User-defined service name |
| `mysql[].node_id` | string | Node identifier where the service runs |
| `mysql[].address` | string | Access address (DNS name or IP) |
| `mysql[].port` | integer | Access port |
| `mysql[].socket` | string | Access unix socket (alternative to address/port) |
| `mysql[].environment` | string | Environment name |
| `mysql[].cluster` | string | Cluster name |
| `mysql[].replication_set` | string | Replication set name |
| `mysql[].custom_labels` | object | Custom user-assigned labels |
| `mysql[].version` | string | MySQL version, when PMM has detected it |
| `mysql[].extra_dsn_params` | object | Additional connection parameters |
| `postgresql` | array | List of PostgreSQL services supporting RTA |
| `postgresql[].service_id` | string | Unique service identifier (use this to start sessions) |
| `postgresql[].service_name` | string | User-defined service name |
| `postgresql[].database_name` | string | Database name |
| `postgresql[].node_id` | string | Node identifier where the service runs |
| `postgresql[].address` | string | Access address (DNS name or IP) |
| `postgresql[].port` | integer | Access port |
| `postgresql[].socket` | string | Access unix socket (alternative to address/port) |
| `postgresql[].environment` | string | Environment name |
| `postgresql[].cluster` | string | Cluster name |
| `postgresql[].replication_set` | string | Replication set name |
| `postgresql[].custom_labels` | object | Custom user-assigned labels |
| `postgresql[].version` | string | PostgreSQL version, when PMM has detected it |
| `postgresql[].auto_discovery_limit` | integer | Limit of databases for auto-discovery |

### Examples

#### List all RTA-compatible services
```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/services" \
  -H "Authorization: Bearer glsa_xxxxx"
```

#### Filter by MongoDB services only
```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/services?service_type=SERVICE_TYPE_MONGODB_SERVICE" \
  -H "Authorization: Bearer glsa_xxxxx"
```

#### Filter by MySQL services only

Returns MySQL, Percona Server for MySQL and MariaDB services.

```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/services?service_type=SERVICE_TYPE_MYSQL_SERVICE" \
  -H "Authorization: Bearer glsa_xxxxx"
```

#### Filter by PostgreSQL services only

```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/services?service_type=SERVICE_TYPE_POSTGRESQL_SERVICE" \
  -H "Authorization: Bearer glsa_xxxxx"
```

### Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | Services retrieved successfully |
| `400` | Invalid Argument | The requested `service_type` does not support RTA |
| `401` | Unauthorized | Missing or invalid authentication token |
| `403` | Forbidden | Insufficient permissions |
| `500` | Internal Server Error | Server error processing request |

### Error response format
```json
{
  "code": 16,
  "message": "Unauthorized",
  "details": []
}
```

Requesting a service type that does not support RTA:

```json
{
  "error": "Service type valkey does not support Real-Time Analytics",
  "code": 3,
  "message": "Service type valkey does not support Real-Time Analytics",
  "details": []
}
```

To get the authentication token, see [Authentication](ref:authentication).