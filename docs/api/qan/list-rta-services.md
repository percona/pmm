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

### Query parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_type` | string | No | Filter by service type. Default: `SERVICE_TYPE_UNSPECIFIED` (returns all supported types) |

#### Supported service types

| Value | Description |
|-------|-------------|
| `SERVICE_TYPE_UNSPECIFIED` | All supported service types (default) |
| `SERVICE_TYPE_MONGODB_SERVICE` | MongoDB services only |
| `SERVICE_TYPE_MYSQL_SERVICE` | MySQL services (future support) |
| `SERVICE_TYPE_POSTGRESQL_SERVICE` | PostgreSQL services (future support) |
| `SERVICE_TYPE_VALKEY_SERVICE` | Valkey services (future support) |
| `SERVICE_TYPE_PROXYSQL_SERVICE` | ProxySQL services (future support) |
| `SERVICE_TYPE_HAPROXY_SERVICE` | HAProxy services (future support) |
| `SERVICE_TYPE_EXTERNAL_SERVICE` | External services (future support) |

### Response
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

### Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | Services retrieved successfully |
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

To get the authentication token, see [Authentication](ref:authentication).