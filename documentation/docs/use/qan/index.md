# About Query Analytics

Query Analytics helps you find and fix slow queries. Use it to identify performance bottlenecks, understand query patterns, and track optimization progress.

![!image](../../images/PMM_Query_Analytics.jpg)

## Stored metrics and Real-time

Query Analytics offers two ways to analyze queries:

- **Stored metrics**: Review completed queries to find patterns, identify slow queries, and track optimization over time. Supports MySQL, PostgreSQL, and MongoDB.

- **Real-time**: See queries as they execute to troubleshoot problems as they happen. Supports MongoDB only (Technical Preview).

## Label-based access control

Query Analytics integrates with PMM's [label-based access control (LBAC)](../../admin/roles/access-control/intro.md) to enforce data security and user permissions.

When LBAC is enabled:

- users see only queries from databases and services permitted by their assigned roles
- filter dropdown options are dynamically restricted based on user permissions
- data visibility is controlled through Prometheus-style label selectors

## Get started

- [Stored metrics](../qan/QAN-stored-metrics.md) — analyze historical query performance
- [Real-time](../qan/QAN-realtime-analytics.md) — troubleshoot queries as they execute