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

- monitor currently executing queries in real-time
- identify long-running or problematic queries as they happen
- enable or disable real-time monitoring for specific services or clusters
- integrate live query monitoring into custom dashboards
- automate real-time monitoring configuration

**Base URL:** `https://your-pmm-server/v1/realtime`

**Authentication:** All endpoints require [Bearer token authentication](ref:authentication#bearer-authentication).

## Real-time vs. stored metrics

| Feature | Real-time Analytics (RTA) | Query Analytics  |
|---------|---------------------------|------------------------|
| **Data type** | Currently executing queries | Historical query performance |
| **Time range** | Current moment only | Any past time range |
| **Use case** | Identify active issues now | Analyze trends and patterns |
| **Database support** | MongoDB | MySQL, PostgreSQL, MongoDB |
| **Refresh rate** | Live updates | Historical snapshots |

## Available endpoints

- [Get real-time query data](ref:getrealtimequerydata): Retrieve currently executing queries
- [Change real-time analytics configuration](ref:changerealtimeconfig): Enable/disable RTA for services or clusters

## Common use cases

### Monitor active database load

Use RTA to see what queries are currently running and identify potential performance bottlenecks in real-time:

1. Call `GET /v1/realtime/query-data` to retrieve active queries
2. Filter by `cluster` or `service` to focus on specific instances
3. Identify long-running queries that may be impacting performance

### Enable RTA for a new service

When adding a new MongoDB service to monitoring, enable RTA to get immediate visibility:

1. Add the MongoDB service to PMM inventory
2. Call `POST /v1/realtime/change` with `service_id` and `enabled: true`
3. Verify RTA is active by calling `GET /v1/realtime/query-data`

### Cluster-wide monitoring

Enable RTA across an entire MongoDB cluster for comprehensive visibility:

1. Call `POST /v1/realtime/change` with `cluster` name and `enabled: true`
2. All services in the cluster will have RTA enabled
3. Use `GET /v1/realtime/query-data?cluster=<name>` to see cluster-wide activity

## Authentication

All RTA endpoints require authentication using service account tokens. Include your token in the request header:

```sh
curl -X GET "https://your-pmm-server/v1/realtime/query-data" \
  -H "Authorization: Bearer YOUR_SERVICE_TOKEN" \
  -H "Content-Type: application/json"
```

For details about creating and managing service account tokens, see [Authentication with service accounts](https://docs.percona.com/percona-monitoring-and-management/3/api/authentication.html)

## Best practices

### Request optimization

To minimize server load and improve response times:

- **Use appropriate time ranges**: limit your queries to the smallest time window that meets your needs
- **Implement pagination**: use offset and limit parameters for large result sets
- **Cache filter results**: the available filters change infrequently, so cache GetFilters responses
- **Avoid duplicate requests**: ensure your application logic triggers API calls only once per user action

### Efficient polling

When integrating RTA into dashboards or monitoring tools:

- poll at reasonable intervals (5-10 seconds minimum)
- use service or cluster filters to reduce response size
- implement exponential backoff if the API is unavailable
- cache configuration state to avoid unnecessary `change` calls

### Resource considerations

Real-time monitoring adds overhead to both MongoDB and PMM Server. To minimize performance impact, enable RTA selectively:

- enable RTA only for services that need active monitoring
- disable RTA during maintenance windows if not needed
- monitor PMM Server resource usage when RTA is enabled on many services

## Troubleshooting

### Duplicate requests

When opening or refreshing QAN, you may see the same API requests (`getFilters` and `getMetrics`) triggered multiple times simultaneously, which causes unnecessary server load and slower response times.

**Cause:** Page refresh or navigation triggers may fire API calls multiple times

**Solution:** Make sure your code doesn't send the same request twice:
- cancel pending requests
- wait briefly after user actions
- tracking requests that are already running

### Empty results

The API returns an empty result set even though you expect to see query data.

**Possible causes:**

- No query data available for the specified time range
- Filters too restrictive
- Selected service has no QAN data collection enabled

**Solution:**

- Verify QAN is enabled for your monitored services
- Check time range covers period with actual query activity
- Review filter criteria for typos or invalid values

## Related resources

- [Query Analytics user documentation](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics.html)
- [Complete PMM API documentation](https://percona-pmm.readme.io/reference/introduction)