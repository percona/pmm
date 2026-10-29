# MongoDB Top Queries

This dashboard ranks MongoDB queries by time spent, executions, documents examined, query targeting, and collection scans. It also shows the busiest collections and the users who run the most queries. Use these rankings to decide which queries and collections to optimize first.

Open it from **MongoDB > Top queries** in the main menu. Use the filters at the top to scope the view to an environment, cluster, node, service, replica set, or node type. The **QAN Agent** filter limits the query panels to one Query Analytics (QAN) source, the profiler or mongolog.

## Prerequisites

The dashboard combines two data sources:

- QAN data for the **Top Queries**, **Missing Indexes / Collection Scans**, and **Users** rows. Configure a [query source for MongoDB](../../install-pmm/install-pmm-client/connect-database/mongodb.md#step-2-configure-query-source-for-mongodb-query-analytics) on each service.
- MongoDB exporter metrics for the **Overview** and **Hottest Collections** rows. **Hottest Collections** needs the `topmetrics` collector, which is disabled by default. Enable it with [`--enable-all-collectors`](../../use/commands/pmm-admin/add.md#collector-options) when you add the service. Without it, both **Hottest Collections** panels show **No data**.

## Overview

These panels show the overall workload of the selected services, from MongoDB exporter metrics.

### Operations per Second

Shows the operations per second for each selected service, excluding commands.

### Read & Write Latency

Shows the average latency for each operation type, such as reads, writes, and commands, across the selected services. The data comes from `serverStatus` `opLatencies`.

A rising read latency with flat throughput can mean that queries are examining more documents.

### Connections

Shows the current and active client connections for each service.

## Top Queries

Each panel in this row ranks the 10 queries that score highest on one measure over the selected time range. The tables list the cluster, service, query ID, database, collection, and query fingerprint. Click a query to open it in [Query Analytics](../../use/qan/qan_mongo.md) in a new tab.

### Top 10 Queries by Time Spent — Trend

Shows the load of the 10 queries with the most execution time in the selected range. Load is query time per second, the same measure as **Load** in Query Analytics.

Use this to tell a steady load from a spike.

### Top 10 Slowest Queries

Lists the queries with the highest average execution time, calculated as total time divided by total executions.

Check **Executed** before acting on a row: a query that ran only once can top this list.

### Top 10 Time Spent

Lists the queries with the most total execution time.

### Top 10 Queries Executed the Most

Lists the queries with the most executions.

### Top 10 Queries Returning the Most Documents

Lists the queries that returned the most documents.

### Top 10 Queries Examining the Most Documents

Lists the queries that examined the most documents, with the keys examined and the **Examined / Returned** ratio. If a query returned no documents, **Examined / Returned** shows the number of documents examined.

### Top 10 Queries by Poor Query Targeting

Lists the queries with the highest ratio of documents examined to documents returned, which is MongoDB's query targeting ratio. A ratio of 1 is ideal. A large ratio means that the query reads far more documents than it returns.

A **Keys Examined / Returned** value near 0 alongside a high ratio means that no index was used. Operations that return no documents, such as writes and reads that match nothing, are excluded because the ratio is undefined for them. Find those in **Top 10 Queries Examining the Most Documents** and **Top 10 Collection Scans**. Queries that examine no documents, such as covered queries, are also excluded.

## Missing Indexes / Collection Scans

This row lists the queries that scanned a whole collection.

### Top 10 Collection Scans (missing index candidates)

Lists the queries whose plan was a full collection scan (`COLLSCAN`), ranked by total execution time. These are the strongest candidates for a new index.

The plan is recorded per execution, so a query appears here only for the periods in which it scanned the collection.

## Hottest Collections

These panels show which collections receive the most operations, the same view as `mongotop`. They require the `topmetrics` collector and work on `mongod` services only, not on `mongos`.

Both panels show the rate at the end of the selected time range. After a workload stops, the bars drop to 0 even if the range still contains activity.

### Top 10 Hottest Collections by Read

Shows read operations per second for each collection. Only queries count; `getmore` operations are excluded. This is the same measure as **Top 5 Hottest Collections by Read (Rate)** on the [MongoDB Collections Overview](dashboard-mongodb-collection_overview.md) dashboard.

### Top 10 Hottest Collections by Write

Shows write operations per second for each collection, as the sum of inserts, updates, and removes.

## Users

This row shows which database users generate the query load.

### Top 10 Users Executed the Most Queries

Lists the users who executed the most queries, aggregated per user across all databases. **Services** and **Databases** show how many monitored services and databases the user ran queries on, and **Total Time** shows their total execution time.
