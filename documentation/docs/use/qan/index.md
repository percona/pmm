# About query analytics (QAN)

The Query Analytics dashboard shows how queries are executed and where they spend their time. It helps you analyze database queries over time, optimize database performance, and find and remedy the source of problems.

![!image](../../images/PMM_Query_Analytics.jpg)

## Supported databases

Query Analytics supports MySQL, MongoDB and PostgreSQL with the following minimum requirements:

### MySQL requirements

- MySQL 5.1 or later (if using the slow query log).
- MySQL 5.6.9 or later (if using Performance Schema).
- Percona Server 5.6+ (all Performance Schema and slow log features)


### Dashboard components
Query Analytics displays metrics in both visual and numeric form. Performance-related characteristics appear as plotted graphics with summaries.

The dashboard contains three panels:

- the [Filters Panel](panels/filters.md);
- the [Overview Panel](panels/overview.md);
- the [Details Panel](panels/details.md).

!!! note alert alert-primary "Note"
    Query Analytics data retrieval is not instantaneous and can be delayed due to network conditions. In such situations *no data* is reported and a gap appears in the sparkline.

## Limitation: Missing query examples in MySQL Performance Schema

When using MySQL Performance Schema, you may see *“Sorry, no examples found”* in query examples. This can happen due to high number of threads, a large volume of unique queries, or Performance Schema settings that cause the history buffer to be overwritten.

All query metrics are collected normally - only examples are affected.

### Why This Happens
MySQL Performance Schema uses two tables:

- Summary table (`events_statements_summary_by_digest`) - stores query statistics (always available)
- History table (`events_statements_history`) - stores individual query examples (limited buffer that gets overwritten)

In busy systems, examples get overwritten before PMM can collect them.

### Workaround

If you are missing query examples, enable the `slowlog` log for reliable query examples. Then [configure PMM to use the `slow query log`](../../../docs/install-pmm/install-pmm-client/connect-database/mysql/mysql.md#configure-data-source) instead of `Performance Schema`. The slow query log retains actual query text over time and isn't subject to buffer limitations.