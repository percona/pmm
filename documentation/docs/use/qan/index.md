# About Query Analytics

Query Analytics helps you find and fix slow queries. Use it to identify performance bottlenecks, understand query patterns, and track optimization progress.

![!image](../../images/PMM_Query_Analytics.jpg)

## Stored metrics vs. Real-time

Query Analytics offers two ways to analyze queries:

- **Stored metrics**: Review completed queries to find patterns, identify slow queries, and track optimization over time. Supports MySQL, PostgreSQL, and MongoDB.

- **Real-time**: See queries as they execute to troubleshoot problems as they happen. Supports MongoDB only (Technical Preview).

| Feature | Real-time Analytics (RTA) | Stored metrics (QAN) |
|---------|---------------------------|------------------------|
| **Data type** | Currently executing queries | Completed queries |
| **Purpose** | Live troubleshooting | Performance optimization |
| **Time range** | Current moment only (30-second display window) | Historical data (configurable retention) |
| **Use case** | Spot problematic operations during incidents | Analyze trends and optimize past performance |
| **Database support** | MongoDB (Technical Preview) | MySQL, PostgreSQL, MongoDB |
| **Data retention** | Ephemeral (disappears after ~30 seconds) | Persistent (stored for analysis) |
| **Refresh rate** | Live updates (1-5 seconds, configurable) | Historical snapshots |
| **Query details** | Raw operation data from `db.currentOp()` | Aggregated metrics and query fingerprints |
| **Best for** | "What's slowing down my database right now?" | "Which queries should I optimize?" |

## Label-based access control

Query Analytics integrates with PMM's [label-based access control (LBAC)](../../admin/roles/access-control/intro.md) to enforce data security and user permissions.

When LBAC is enabled:

- users see only queries from databases and services permitted by their assigned roles
- filter dropdown options are dynamically restricted based on user permissions
- data visibility is controlled through Prometheus-style label selectors

## Get started

- [Stored metrics QAN](../qan/QAN-stored-metrics.md) 
- [Real-time analytics for MongoDB](../qan/QAN-realtime-analytics.md) 