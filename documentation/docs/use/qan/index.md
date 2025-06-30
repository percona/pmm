# About query analytics (QAN)

The *Query Analytics* dashboard shows how queries are executed and where they spend their time.  It helps you analyze database queries over time, optimize database performance, and find and remedy the source of problems.

![!image](../../images/PMM_Query_Analytics.jpg)

Query Analytics supports MySQL, MongoDB and PostgreSQL. The minimum requirements for MySQL are:

- MySQL 5.1 or later (if using the slow query log).
- MySQL 5.6.9 or later (if using Performance Schema).

Query Analytics displays metrics in both visual and numeric form. Performance-related characteristics appear as plotted graphics with summaries.

The dashboard contains three panels:

- the [Filters Panel](panels/filters.md);
- the [Overview Panel](panels/overview.md);
- the [Details Panel](panels/details.md).

!!! note alert alert-primary "Note"
    Query Analytics data retrieval is not instantaneous and can be delayed due to network conditions. In such situations *no data* is reported and a gap appears in the sparkline.

## Limitation: Missing Query Examples in MySQL Performance Schema

### Overview

Depending on your MySQL configuration — including the number of threads, volume of unique queries, and Performance Schema settings — it's possible to **miss query examples** in certain cases. **All other query metrics will still be collected as expected.**

---

### What's Happening Under the Hood

- `events_statements_summary_by_digest`  
  - This table stores **aggregated metrics** for each unique query (digest).  
  - Each normalized query appears **only once**, regardless of how many times it was executed.

- `events_statements_history` (or `events_statements_history_long` in MariaDB)  
  - This table stores **individual query executions**, meaning **multiple rows** can exist for the same query.  
  - It works as a **fixed-size rolling buffer** and is subject to being overwritten as new queries come in.

---

### What Can Go Wrong

A query may appear in the **digest summary** (`events_statements_summary_by_digest`) but **not in the history table**. This happens when:

- The query was executed frequently enough to be included in the digest summary.
- However, all individual executions have already been **removed from the history buffer** due to its size limit and ongoing activity.

As a result QAN may show the query’s metrics, but **fail to display an example**, because it's no longer available in `events_statements_history` during capturing.

---

### Workaround

If you are missing query examples, consider enabling and using the `slowlog` as an alternative query source. The `slowlog` retains actual query texts over time and can help capture examples even when Performance Schema history buffers are exhausted.
