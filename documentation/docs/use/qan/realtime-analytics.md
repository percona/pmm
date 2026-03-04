# Real-Time Query Analytics for MongoDB

!!! warning "Technical Preview"
    This feature is not production-ready. Use for testing and feedback only.

Real-Time Query Analytics (RTA) shows you what's happening on your databases right now.

While [Query Analytics (QAN)](../qan/index.md) stores queries after they complete for performance review and optimization, Real-Time Analytics shows queries as they execute. This means that when your database is struggling, you can immediately see what's causing the problem.

With a live stream updated every 1-2 seconds, RTA lets you spot long-running queries, identify lock contention, and investigate problematic operations as they happen.

Currently, RTA only supports MongoDB databases. MySQL and PostgreSQL support is planned for future releases.

## How it works

RTA creates a dedicated agent that queries MongoDB's `currentOp()` at regular intervals. By default, this is every 2 seconds. The data flows from the agent to PMM Server, where it's stored briefly in memory and streamed on the **Query Analytics (QAN) > Real-time** page. 


RTA data is not persisted. It exists only in memory for approximately 30 seconds. When you open the Real-Time view, you see what's happening now—there's no history to scroll back through.

## Before you start

To use RTA, you need:

- PMM Server and PMM Client running
- At least one MongoDB service registered with PMM
- A configured MongoDB exporter (RTA reuses its credentials)

## Start a session

To start monitoring a MongoDB service:
{.power-number}

1. Go to **Query Analytics** in the sidebar.
2. Select the **Real-time** tab.
3. Select a MongoDB service from the dropdown.
4. Click **Start Session**.

The live operations table appears and begins updating automatically.

You can run multiple RTA sessions simultaneously for different MongoDB services.

## The operations table

The main table shows currently executing operations. Click any column header to sort.

| Column | Description |
|--------|-------------|
| **Query** | The reconstructed query text. See [Query reconstruction](#query-reconstruction). |
| **Operation ID** | The unique identifier the database assigns to track this operation. |
| **Elapsed time** | How long this operation has been running so far. |

Click any row to open the details panel.

## Details panel

The details panel shows full information about the selected operation.

### Details tab

| Field | Description |
|-------|-------------|
| **Operation ID** | The unique identifier the database assigns to track this operation. Useful for finding this operation in logs, referencing it in support tickets, or killing it if needed. |
| **Elapsed exec. time** | How long this operation has been running so far. Long-running operations may indicate missing indexes, lock contention, or queries that need optimization. |
| **DB instance address** | The network address where the database sees itself. This may differ from how the service is registered in PMM. |
| **Database name** | The database where this operation is running. Use this to identify which database is under load. |
| **Collection** | The MongoDB collection this operation queries. If you see the same collection repeatedly with slow operations, it may need better indexing. |
| **Operation** | The type of database action being performed, such as query, insert, update, or command. Use this to understand the workload pattern. |
| **User name** | The database user who started this operation. Use this to identify which application or service is generating problematic queries. |
| **Client address** | The IP address and port of the client that sent this query. Use this to identify which server or container is generating load. |
| **Client app name** | The name of the application or driver that started this operation. Use this to identify which application is causing issues when multiple apps connect to the same database. |
| **Plan summary** | How the database is executing this query. Look for `COLLSCAN` (full collection scan) which often means a missing index, or `IXSCAN` (index scan) which is typically more efficient. |
| **Service** | The PMM service name for this database instance. Use this to find related metrics on other PMM dashboards. |
| **Host** | The server hostname and port where this operation is running. In replica sets, this shows which member is handling the query. |
| **Operation start time** | When the database started executing this operation. Compare with other timestamps to correlate slow queries with events like deployments or traffic spikes. |
| **Data capture time** | When PMM captured this snapshot. Compare with start time to see how long the query had been running at capture. |

### Raw data tab

Shows the exact JSON response from MongoDB's `currentOp()` command. Use this to see all fields MongoDB returns, including those not displayed in the Details tab.

## Query reconstruction

The **Query** column shows a reconstructed version of the original query. MongoDB's `currentOp()` command doesn't return the exact query text—it returns a structured command object. RTA attempts to reconstruct readable query text from this data.

This reconstruction may differ slightly from the original query you sent. For the exact data MongoDB returned, check the **Raw data** tab.

## Controls

### Refresh interval

Controls how often the UI fetches new data from the server. Options range from 1 to 5 seconds, with 2 seconds as the default.

This setting only affects how often the UI updates—not how often the agent collects data from MongoDB.

### Pause and resume

Click **Pause** to freeze the current view. The agent continues collecting data in the background, but the table stops updating. This lets you investigate a specific operation without it disappearing.

Click **Resume** to continue live updates.

### One-time refresh

When paused, click **Refresh** to fetch the latest data once without resuming automatic updates.

## Filters

Filter the operations list by:

- **Cluster/Service**: Show operations from specific MongoDB instances
- **Min/Max elapsed time**: Show only operations running longer than a threshold

## Share your view

Click the **Share** icon to copy a link. The link preserves your current filters and settings, but not the data—recipients see live operations when they open it.

## Stop a session

To stop monitoring:
{.power-number}

1. Go to **Query Analytics > Real-time**.
2. In the sessions list, select the session(s) to stop.
3. Click **Stop Selected** or **Stop All Sessions**.

When you stop a session, the RTA agent stops but isn't removed. This preserves any custom credentials configured for that agent.

## Privacy considerations

!!! caution "Sensitive data may be visible"
    RTA displays raw query data from MongoDB, which may include sensitive information such as:
    
    - Values in insert or update statements
    - Filter criteria containing user data
    - Credentials passed in queries
    
    This data is visible to any PMM user who can access the Real-Time view. Consider your security requirements before enabling RTA in production environments.

RTA shows exactly what MongoDB returns—it doesn't add or expose additional information beyond what `currentOp()` provides. If your MongoDB instance has encryption or log redaction enabled, those protections apply to RTA data as well.

## Limitations

- **MongoDB only**: MySQL and PostgreSQL support is planned for future releases
- **No data persistence**: RTA data is not stored; you cannot view historical real-time data
- **Fast queries may not appear**: Queries that complete between collection intervals won't be captured
- **Reconstructed queries**: Query text is approximated, not exact (check Raw data for the exact response)
- **Credentials required**: RTA cannot start if no MongoDB exporter is configured for the service

## Troubleshooting

### No data appears

- Verify the MongoDB exporter is running and healthy in **PMM Inventory > Services**
- Check that the RTA session status shows as **Running**, not **Failing**
- Generate some database load—if queries complete faster than the collection interval, the table may appear empty

### Session won't start

RTA copies credentials from the existing MongoDB exporter. If no exporter is configured for the service, the session cannot start.

### "Failing" status in Inventory

After stopping an RTA session, the service may briefly show a warning status because the RTA agent is in a "done" state. This is expected and doesn't indicate a problem with your MongoDB monitoring.

## See also

- [Query Analytics overview](index.md)
- [Connect MongoDB to PMM](../../install-pmm/install-pmm-client/connect-database/mongodb.md)
- [MongoDB dashboards](../../reference/dashboards/dashboard-mongodb-instance-summary.md)