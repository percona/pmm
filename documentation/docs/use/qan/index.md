# About Query Analytics

Query Analytics (QAN) helps you find and fix slow queries. Use it to identify performance bottlenecks, understand query patterns, and track optimization progress.

![!image](../../images/PMM_Query_Analytics.jpg)

## Stored metrics and Real-time QAN

Query Analytics offers two ways to analyze queries:

- **Stored metrics**: Choose stored metrics when you want to  analyze completed queries to identify patterns, find slow queries, and track optimization progress over time. 

- **Real-time**: Choose real-time when you need to identify problematic operations during an active incident.  

### Real-time vs. Stored metrics capabilities

| Feature | Real-time Analytics (RTA) | Stored metrics (QAN) |
|---------|---------------------------|------------------------|
| **Data type** | Currently executing queries | Completed queries |
| **Purpose** | Live troubleshooting | Performance optimization |
| **Time range** | Live data (updates every 1-5 seconds) | Historical data (configurable retention) |
| **Use case** | Spot problematic operations during incidents | Analyze trends and optimize past performance |
| **Database support** | MongoDB (Technical Preview) | MySQL, PostgreSQL, MongoDB |
| **Data retention** | Temporary (refreshes with new data) | Persistent (stored for analysis) |
| **Refresh rate** | Live updates (1-5 seconds, configurable) | Historical snapshots |
| **Query details** | Raw operation data from `db.currentOp()` (no aggregation, grouping, or processing) | Aggregated metrics and query fingerprints |
| **Best for** | "What's slowing down my database right now?" | "Which queries should I optimize?" |

### Requirements for Profiler

    - Profiling enabled for Query Analytics
    - Appropriate user roles: `clusterMonitor`, `read` (local), and custom monitoring roles. For MongoDB 8.0+: Additional `directShardOperations` role required for sharded clusters

    ### Requirements for Mongolog

    - MongoDB configured to log slow operations to a file
    - MongoDB server has write permissions to the log directory and file
    - PMM agent has read permissions to the MongoDB log file
    - Appropriate user roles: `clusterMonitor`, or custom monitoring roles (`getCmdLineOpts` privilege on `{ cluster: true }`)


## ClickHouse configuration profiles

Query Analytics in PMM supports two ClickHouse configuration profiles for different memory environments:

- **default-config.xml** *(default)* — for normal environments and better performance
- **low-memory-config.xml** — for limited memory resources

!!! note
    Low memory config is based on ClickHouse' recommendations: https://clickhouse.com/docs/operations/tips#using-less-than-16gb-of-ram
 
Both files are located in `/etc/clickhouse-server/` inside the PMM Server container.

### How to switch profiles

To switch between profiles, use the `switch-config.sh` script (available in `/etc/clickhouse-server/` and as `/opt/switch-config.sh`).

**Usage:**
Run inside the container from one of the paths mentioned above:

    ./switch-config.sh [low|default]

Or from outside the container:

    docker exec -it -u pmm pmm-server ./switch-config.sh [low|default]

Replace `[low|default]` with the desired profile.

Where:

- `low` switches to the low-memory configuration
- `default` switches to the default memory configuration

The script will:

1. Stop the ClickHouse service using `supervisorctl`
2. Update `/etc/clickhouse-server/config.xml` and `/etc/clickhouse-server/users.xml` to point to the selected profile (both config and users files are switched for `default` and `low`)
3. Print a confirmation message

**Example:**

    /switch-config.sh low

This activates the low-memory configuration.

### Default vs. low-memory configuration: key differences

The following table summarizes the main differences between the two ClickHouse configuration profiles and explains each property:

| Property | Default | Low memory | Description |
|---|---|---|---|
| `concurrent_threads_soft_limit_num` | 0 (unlimited, uses all cores) | 1 | Maximum query processing threads. 1 limits parallelism for low memory. |
| `max_block_size` | 65409 | 8192 | Max block size (rows) for query processing. Lower value reduces memory per query. |
| `max_download_threads` | 0 (unlimited) | 1 | Max threads for downloading data. 1 = less concurrency, less memory. |
| `input_format_parallel_parsing` | 1 | 0 | Disables parallel parsing of input formats. Saves memory. |
| `output_format_parallel_formatting` | 1 | 0 | Disables parallel formatting of output. Saves memory. |
| `trace_log` | 1 | 0 | Disables logging for this component. Saves memory. |
| `metric_log` | 1 | 0 | Disables logging for this component. Saves memory. |
| `asynchronus_metric_log` | 1 | 0 | Disables logging for this component. Saves memory. |
| `max_server_memory_usage_to_ram_ratio` | 0.75 | 0.5 | Fraction of system RAM ClickHouse can use. Lower value is safer for low-memory hosts. |
| `uncompressed_cache_size` | 8 GB | 2 GB | Cache for uncompressed data blocks. Lower value saves memory. |
| `mark_cache_size` | 5 GB | 512 MB | Cache for index marks. Lower value saves memory, but may slow queries. |

**Summary:**

- The **default** config is tuned for performance and parallelism, using more RAM and allowing more connections and threads.
- The **low-memory** config restricts concurrency, cache sizes, and memory usage to fit into smaller environments, at the cost of performance.


## Dashboard components
Query Analytics displays metrics in both visual and numeric form. Performance-related characteristics appear as plotted graphics with summaries.

## Dashboard layout
The dashboard contains three panels:

- the [Filters panel](panels/filters.md)
- the [Overview panel](panels/overview.md)
- the [Details panel](panels/details.md)

### Data retrieval delays

Query Analytics data retrieval is not instantaneous because metrics are collected once per minute. When collection delays occur, no data is reported and gaps will appear in the sparkline.

## Label-based access control

Query Analytics integrates with PMM's [label-based access control (LBAC)](../../admin/roles/access-control/intro.md) to enforce data security and user permissions.

When LBAC is enabled:

- users see only queries from databases and services permitted by their assigned roles
- filter dropdown options are dynamically restricted based on user permissions
- data visibility is controlled through Prometheus-style label selectors

## Get started

- [Stored metrics QAN](../qan/QAN-stored-metrics.md) 
- [Real-time analytics for MongoDB](../qan/QAN-realtime-analytics.md) 