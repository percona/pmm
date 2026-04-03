---
title: Overview
slug: real-time-analytics
category:
  uri: rta-api
position: 0
privacy:
  view: public
---

Real-time Analytics (RTA) provides live visibility into currently executing queries on your MongoDB clusters. Unlike stored metrics Query Analytics (QAN), which analyzes historical query performance data, RTA shows you what's happening right now on your database.

Use the RTA API to:

- start and stop real-time monitoring sessions for MongoDB services
- search currently executing queries in active sessions
- list all active monitoring sessions
- integrate live query monitoring into custom dashboards
- automate session management

**Base URL:** `https://your-pmm-server/v1/realtimeanalytics`

**Authentication:** All endpoints require [Bearer token authentication](ref:authentication#bearer-authentication).

## Real-time vs. stored metrics

| QAN feature | Real-time Analytics (RTA) | Stored metrics  |
|---------|---------------------------|------------------------|
| **Data type** | Currently executing queries | Historical query performance |
| **Time range** | Live data (updates every 1-5 seconds) | Historical data (configurable retention) |
| **Use case** | Identify active issues now | Analyze trends and patterns |
| **Database support** | MongoDB (Technical Preview) | MySQL, PostgreSQL, MongoDB |
| **Data retention** | Ephemeral (not stored) | Persistent (stored for analysis) |

## Available endpoints

- [List RTA-compatible services](ref:list-rta-services): retrieve services that support Real-Time Analytics
- [Search real-time analytics queries](ref:search-rta-queries): retrieve currently executing queries from active sessions
- [Manage real-time analytics sessions](ref:manage-rta-sessions): start, stop, and list real-time monitoring sessions for MongoDB services

## Common use cases

### Monitor active database operations

When your database is experiencing performance issues, use RTA to see exactly what queries are running and identify bottlenecks in real-time:

1. List available services with `GET /v1/realtimeanalytics/services`
2. Start a session with `POST /v1/realtimeanalytics/sessions:start`
3. Search active queries with `POST /v1/realtimeanalytics/queries:search`
4. Filter results by service to focus on specific instances

### Automated session management

Integrate RTA into your monitoring workflows to automatically enable real-time monitoring during peak hours or when alerts trigger:

1. List existing sessions with `GET /v1/realtimeanalytics/sessions`
2. Start monitoring when needed with `POST /v1/realtimeanalytics/sessions:start`
3. Stop monitoring when done with `POST /v1/realtimeanalytics/sessions:stop`

### Cluster-wide monitoring

Get comprehensive visibility across your entire MongoDB cluster by monitoring all replica set members simultaneously:

1. Start sessions for each service in the cluster
2. Use the cluster filter in list sessions to view cluster status
3. Search queries across all cluster services

## Authentication

All RTA endpoints require authentication using service account tokens. Include your token in the request header:

```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/sessions:start" \
  -H "Authorization: Bearer YOUR_SERVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"service_id": "your-service-id"}'
```

For details about creating and managing service account tokens, see [Authentication with service accounts](https://docs.percona.com/percona-monitoring-and-management/3/api/authentication.html).

## Best practices

### Session management and resource considerations

Real-time monitoring adds overhead to both MongoDB and PMM Server. Manage sessions carefully to minimize performance impact:

- start sessions only when actively troubleshooting or for services that need active monitoring
- stop sessions when monitoring is no longer needed or during maintenance windows
- use list sessions to track which services are being monitored
- monitor session status regularly to detect errors or failures
- monitor PMM Server resource usage when multiple sessions are active

### Query search optimization

To minimize server load and improve response times when searching for active queries, follow these guidelines.

- Use `service_ids` filter to limit results to specific services
- Use `limit` parameter to control result set size
- Match your polling interval to the session's collection interval (check `collect_interval` in the session response). Polling slower than the collection interval means you'll miss queries that start and finish between your requests.

## Session status values

Each RTA session has a status that indicates whether it's actively collecting data, has encountered an error, or has been stopped:

| Status | Description |
|--------|-------------|
| `SESSION_STATUS_RUNNING` | Session is actively collecting data |
| `SESSION_STATUS_ERROR` | Session encountered an error |
| `SESSION_STATUS_DOWN` | Session has been stopped or disabled |

## Related resources

- [Real-time Analytics user documentation](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics.html)
- [Complete PMM API documentation](https://percona-pmm.readme.io/reference/introduction)