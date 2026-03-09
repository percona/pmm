---
title: Change real-time analytics configuration
slug: changerealtimeconfig
content:
  excerpt: Enable or disable real-time analytics for MongoDB services or clusters.
category:
  uri: rta-api
---

## Change real-time analytics configuration

`POST /v1/realtime/change`

Enables or disables real-time analytics (RTA) for a specific MongoDB service or an entire cluster. When enabled, PMM begins collecting and displaying currently executing queries in real-time.

### Request body

You can enable/disable RTA for either a specific service or an entire cluster (but not both in the same request).

#### Enable RTA for a service
```json
{
  "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
  "enabled": true
}
```

#### Enable RTA for a cluster
```json
{
  "cluster": "production-cluster",
  "enabled": true
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_id` | string | Conditional* | Unique identifier for a MongoDB service |
| `cluster` | string | Conditional* | Name of a MongoDB cluster |
| `enabled` | boolean | Yes | `true` to enable RTA, `false` to disable |

**Note:** Exactly one of `service_id` or `cluster` must be specified, but not both.

### Response
```json
{}
```

**Success response:** Empty object with HTTP 200 status code

### Examples

#### Enable RTA for a specific service
```bash
curl -X POST "https://your-pmm-server/v1/realtime/change" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
    "enabled": true
  }'
```

#### Disable RTA for a specific service
```bash
curl -X POST "https://your-pmm-server/v1/realtime/change" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
    "enabled": false
  }'
```

#### Enable RTA for an entire cluster
```bash
curl -X POST "https://your-pmm-server/v1/realtime/change" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "production-cluster",
    "enabled": true
  }'
```

#### Disable RTA for an entire cluster
```bash
curl -X POST "https://your-pmm-server/v1/realtime/change" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "production-cluster",
    "enabled": false
  }'
```

### Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | RTA configuration changed successfully |
| `400` | Bad Request | Invalid request (missing required fields, both service_id and cluster specified) |
| `401` | Unauthorized | Missing or invalid authentication token |
| `403` | Forbidden | Insufficient permissions to modify RTA configuration |
| `404` | Not Found | Specified service_id or cluster not found |
| `500` | Internal Server Error | Server error processing request |

#### Error response format
```json
{
  "code": 3,
  "message": "service_id or cluster must be specified, but not both",
  "details": []
}
```

#### Common errors

**Both service_id and cluster specified:**
```json
{
  "code": 3,
  "message": "Exactly one of service_id or cluster must be specified",
  "details": []
}
```

**Service not found:**
```json
{
  "code": 5,
  "message": "Service with ID '7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f' not found",
  "details": []
}
```

**Cluster not found:**
```json
{
  "code": 5,
  "message": "Cluster 'production-cluster' not found",
  "details": []
}
```

### Behavior notes

#### Service-level configuration

When enabling RTA for a specific service:
- RTA starts immediately for that service only
- Other services in the same cluster are not affected
- Configuration persists across PMM Server restarts

#### Cluster-level configuration

When enabling RTA for a cluster:
- RTA is enabled for **all services** currently in that cluster
- Services added to the cluster later will **not** automatically have RTA enabled
- You must enable RTA for new services individually or run this command again

#### Performance impact

Enabling RTA has a performance impact on both MongoDB and PMM:
- MongoDB profiling overhead (minimal for most workloads)
- Additional network traffic between MongoDB and PMM
- Increased storage usage for real-time data in PMM

We recommend:
- Enabling RTA selectively for services that need active monitoring
- Disabling RTA during non-critical periods if resources are constrained
- Monitoring PMM Server resource usage when RTA is enabled on multiple services

### Troubleshooting

#### Configuration doesn't apply

**Possible causes:**
- Service or cluster doesn't exist in PMM inventory
- Typographical error in service_id or cluster name
- PMM agent not connected to the service

**Solutions:**
1. Verify service_id using [List services](ref:listservices) endpoint
2. Check cluster name matches exactly (case-sensitive)
3. Confirm PMM agent is running and connected: `pmm-admin status`

#### RTA not showing data after enabling

**Possible causes:**
- MongoDB profiler not enabled
- No active queries on the database
- PMM agent connection issues

**Solutions:**
1. Check MongoDB profiling level: `db.getProfilingLevel()`
2. Verify there's actual query traffic on the database
3. Check PMM agent logs for connection errors
4. Confirm required MongoDB user permissions are granted

To get the authentication token, check [Authentication](ref:authentication).

For service_id values, refer to [List services](ref:listservices).