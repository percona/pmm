---
title: Overview
slug: real-time-analytics
category:
  uri: rta-api
position: 0
privacy:
  view: public
---

Real-Time Analytics (RTA) provides live visibility into currently executing queries on your MongoDB clusters. Unlike Query Analytics (QAN), which analyzes historical query performance data, RTA shows you what's happening right now on your database.

Use the RTA API to:

- monitor currently executing queries in real-time
- identify long-running or problematic queries as they happen
- enable or disable real-time monitoring for specific services or clusters
- integrate live query monitoring into custom dashboards
- automate real-time monitoring configuration

**Base URL:** `https://your-pmm-server/v1/realtime`

**Authentication:** All endpoints require Bearer token authentication. See [Authentication](ref:authentication).

## Real-Time vs. Stored metrics

| Feature | Real-Time Analytics (RTA) | Query Analytics (QAN) |
|---------|---------------------------|------------------------|
| **Data Type** | Currently executing queries | Historical query performance |
| **Time Range** | Current moment only | Any past time range |
| **Use Case** | Identify active issues now | Analyze trends and patterns |
| **Database Support** | MongoDB | MySQL, PostgreSQL, MongoDB |
| **Refresh Rate** | Live updates | Historical snapshots |

## Available Endpoints

- [Get Real-Time Query Data](ref:getrealtimequerydata) - Retrieve currently executing queries
- [Change Real-Time Analytics Configuration](ref:changerealtimeconfig) - Enable/disable RTA for services or clusters

## Common Use Cases

### Monitor Active Database Load

Use RTA to see what queries are currently running and identify potential performance bottlenecks in real-time:

1. Call `GET /v1/realtime/query-data` to retrieve active queries
2. Filter by `cluster` or `service` to focus on specific instances
3. Identify long-running queries that may be impacting performance

### Enable RTA for a New Service

When adding a new MongoDB service to monitoring, enable RTA to get immediate visibility:

1. Add the MongoDB service to PMM inventory
2. Call `POST /v1/realtime/change` with `service_id` and `enabled: true`
3. Verify RTA is active by calling `GET /v1/realtime/query-data`

### Cluster-Wide Monitoring

Enable RTA across an entire MongoDB cluster for comprehensive visibility:

1. Call `POST /v1/realtime/change` with `cluster` name and `enabled: true`
2. All services in the cluster will have RTA enabled
3. Use `GET /v1/realtime/query-data?cluster=<name>` to see cluster-wide activity

## Authentication

All RTA endpoints require authentication using service account tokens. Include your token in the request header:
```bash