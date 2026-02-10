# About Query Analytics (QAN)

The Query Analytics dashboard shows how queries are executed and where they spend their time. It helps you analyze database queries over time, optimize database performance, and find and remedy the source of problems.

![!image](../../images/PMM_Query_Analytics.jpg)

Query Analytics supports MySQL, MongoDB and PostgreSQL with the following minimum requirements:

=== "MySQL requirements"
    - MySQL 5.1 or later (if using the slow query log)
    - MySQL 5.6.9 or later (if using Performance Schema)
    - Percona Server 5.6+ (all Performance Schema and slow log features)
    - MariaDB 5.2+ (for user statistics), 10.0+ (for Performance Schema)

    Some limitations and tuning options apply only when using MySQL’s Performance Schema.  
    See [Query Analytics with MySQL](../qan/mysql.md#limitations-with-performance-schema).

=== "PostgreSQL requirements"
    - PostgreSQL 11 or later
    - `pg_stat_monitor` extension (recommended) or `pg_stat_statements` extension
    - Appropriate `shared_preload_libraries` configuration
    - Superuser privileges for PMM monitoring account

=== "MongoDB requirements"
    - MongoDB 6.0 or later (4.4+ may work with limited features)

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

## QAN for PMM Server's internal PostgreSQL

By default, Query Analytics (QAN) hides queries from PMM Server’s internal PostgreSQL database. This avoids clutter and keeps the focus on your monitored databases.

Enable QAN for PMM Server when you need to troubleshoot performance issues, check resource usage, or ensure that applications are not using the default `postgres` database instead of dedicated databases. This is particularly useful in [High Availability (HA) deployments](../../install-pmm/HA.md) where monitoring system health is essential.

### Enable QAN for PMM Server
To include PMM Server’s own queries in QAN, enable the feature in the settings:
{.power-number}

1. Go to **PMM Configuration > Settings > Advanced Settings**.
2. Switch on the **QAN for PMM Server** option.
3. Open **PMM Query Analytics (QAN)** from the main menu and filter by the `pmm-server-postgresql` service to view queries.

When enabled, QAN displays queries related to PMM’s internal operations—such as inventory, settings, advisor checks, alerts, backups, and authentication. 

These are usually lightweight, but unusual spikes in volume, latency, or unexpected queries may indicate performance issues or misuse of the database.

!!! warning
    Do not use the default PostgreSQL database for application workloads. PMM monitors it for visibility, but applications should always run on dedicated databases.
