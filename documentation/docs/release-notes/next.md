# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15530](https://perconadev.atlassian.net/browse/PMM-15530): Every tool on the [MCP endpoint](../api/mcp.md) now returns structured JSON next to its text. Most results link to the same data in PMM, and a result read over a time range names that window. Errors carry a code and say whether a retry can help. Long lists of services, variables and queries are paged with `limit` and `offset`. The new `pmm_catalog` tool lists the Query Analytics columns that `pmm_top_queries` accepts.
- [PMM-15529](https://perconadev.atlassian.net/browse/PMM-15529): On the [MCP endpoint](../api/mcp.md), `pmm_get_explain` and `pmm_get_schema` now work from a `query_ref` or from a `queryid` alone, and stay within your services. `pmm_query_detail` takes a `query_ref` too. A `query_ref` keeps the time window, so you can explain a query that you saw earlier. `pmm_get_schema` finds the tables of a MySQL query in a live plan. `pmm_query_detail` now reads only your services. `pmm_get_explain` no longer takes the `query` argument: for MongoDB, it explains the example that Query Analytics stored for the query. A `pmm_get_explain` call with `service_id` and `queryid` now searches only the last hour, unless it passes `period_from` and `period_to` or a `query_ref`. Before, such a call found MySQL and PostgreSQL queries of any age, and MongoDB queries of the last 24 hours. To explain an older query, pass `period_from` and `period_to` covering when it ran, or its `query_ref`. Live-inspection calls on one service (`pmm_get_schema`, and `pmm_get_explain` for MySQL and MongoDB) run one at a time per PMM Server replica, and a second call waits up to 10 seconds for the first.
- [PMM-15790](https://perconadev.atlassian.net/browse/PMM-15790): The [MCP endpoint](../api/mcp.md) now answers questions about your metrics. The new `pmm_query_metrics` tool runs a MetricsQL query, instant or over a range, and returns the largest series as points or summaries that fit in 16 KiB. The new `pmm_list_metrics` tool lists metric names, label names, label values and series. The new `pmm_service_health` tool reads the triage signals of one service and its node, taken from the PMM dashboards, and judges nothing. `pmm_catalog` with `kind=metrics` lists the metrics of MySQL, PostgreSQL, MongoDB and the node, each with its meaning, unit and a ready query. Queries are checked before they run: an expression reads at most 7 days and a selector matches at most 2,000 series, or the call fails with the reason. Metric calls are limited per caller and per PMM Server replica, and a call that is refused for its rate says in `retry_after` how long to wait. Label-based access control applies to every metrics tool. `pmm_top_queries` and `pmm_query_detail` now read a window of at most 30 days. For the limits, see [Metrics tools](../api/mcp.md#metrics-tools).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
