# Connect PostgreSQL databases to PMM

Connect your PostgreSQL databases—whether self-hosted or running in the cloud—to Percona Monitoring and Management (PMM) for comprehensive performance monitoring and analysis.

PMM Client supports collecting metrics from PostgreSQL-based database systems:

- [PostgreSQL][PostgreSQL]
- [Percona Distribution for PostgreSQL][Percona Distribution for PostgreSQL]

For monitoring Amazon RDS PostgreSQL instances, see [Connect Amazon RDS instance](../connect-database/aws.md).

??? info "Setup process at a glance"
    These are the high-level steps for configuring PostgreSQL monitoring in PMM:
    {.power-number}

    1. [Prerequisites](#before-you-start): Ensure PMM Server is running and PMM Client is installed
    2. [Create PMM user](#create-a-database-account-for-pmm): `CREATE USER pmm WITH SUPERUSER ENCRYPTED PASSWORD 'StrongPassword'`
    3. [Configure extension](#choose-a-monitoring-extension): Set up `pg_stat_statements` or `pg_stat_monitor`
    4. [Add service](#add-service-to-pmm): Use PMM UI or command line to add the PostgreSQL instance
    5. [Verify connection](#check-the-service): Check PMM Inventory and dashboards for data. 

## Before you start

Check that:

- [PMM Server is installed](../../install-pmm-server/index.md) and running with a known IP address accessible from the client node.
- [PMM Client is installed](../../install-pmm-client/index.md) and the [nodes are registered with PMM Server](../../register-client-node/index.md).
- You have superuser (root) access on the client host.
- You have superuser access to any database servers that you want to monitor.

PMM follows [PostgreSQL's end-of-life policy][POSTGRESQL_VERSIONING]. For specific details on supported platforms and versions, see [Percona's Software Platform Lifecycle page][PERCONA_LIFECYCLE].

## Create a database account for PMM

PMM requires a dedicated database account with appropriate permissions to collect metrics effectively. We recommend creating a PMM database account that can connect to the postgres database with the SUPERUSER role.

=== "Standard PostgreSQL"

    To create a user with the SUPERUSER role:
    {.power-number}
    
    1. Create a user with SUPERUSER role:
    
        ```sql
        CREATE USER pmm WITH SUPERUSER ENCRYPTED PASSWORD 'strong_password';
        ```
    
    2. Edit the `pg_hba.conf` file to enable local login:
    
        ```conf
        local   all             pmm                                md5
        # TYPE  DATABASE        USER        ADDRESS                METHOD
        ```

=== "Amazon RDS/Aurora PostgreSQL"

    For RDS instances where SUPERUSER cannot be assigned directly:
    {.power-number}
    
    1. Create the user:
    
        ```sql
        CREATE USER pmm WITH ENCRYPTED PASSWORD 'strong_password';
        ```
    
    2. Grant the `rds_superuser` role:
    
        ```sql
        GRANT rds_superuser TO pmm;
        ```
    
    3. Optionally, set a connection limit (only if the user is not a SUPERUSER):
    
        ```sql
        ALTER USER pmm CONNECTION LIMIT 10;
        ```

### Configure authentication

After creating the database user, complete the configuration:
{.power-number}

1. Reload the PostgreSQL configuration:

    ```sh
    su - postgres
    psql -c "select pg_reload_conf()"
    ```

2. Verify the PMM user can connect locally:

    ```sh
    psql postgres pmm -c "\conninfo"
    ```

3. Enter the password for the pmm user when prompted:

## Choose a monitoring extension

Decide which  PostgreSQL monitoring extensions to use, and configure your database server for it. 

Choose:

- [`pg_stat_statements`](#configure-pg_stat_statements) when you need a lightweight, built-in solution with minimal overhead
- [`pg_stat_monitor`](#configure-pg_stat_monitor) when you need comprehensive monitoring capabilities with more detailed insights into query performance:

| Aspect | **pg_stat_statements** | **pg_stat_monitor** |
|----------------|---------------------|------------------|
| **Origin & packaging** | • Official PostgreSQL extension<br>• Part of `postgresql-contrib` package on Linux | • Developed by Percona<br>• Requires separate installation |
| **Key features** | ✔ Basic query statistics<br>✔ Simple aggregation<br>✔ Minimal overhead<br>✔ Query timing and execution counts | ✔ Enhanced metrics collection<br>✔ Bucket-based aggregation<br>✔ Query examples<br>✔ Histogram data<br>✔ Includes all pg_stat_statements features |
| **Best for** | ✔ Development environments<br>✔ Simple monitoring needs<br>✔ Resource-constrained servers | ✔ Production environments<br>✔ Detailed query analysis<br>✔ Performance tuning |
| **Installation complexity** | ⚠ Low | ⚠ Medium |
| **Benefits** | • Part of official PostgreSQL<br>• Minimal overhead<br>• Simple to set up and use | • Builds on pg_stat_statements features<br>• Bucket-based time-series analysis<br>• Query examples for troubleshooting<br>• More accurate performance data |
| **Drawbacks** | • No aggregated statistics or histograms<br>• No Query Examples<br>• Limited metrics collection | • Slightly higher resource overhead<br>• Requires separate installation<br>• More complex configuration |

For a more detailed comparison of extensions, see the [pg_stat_monitor documentation](https://docs.percona.com/pg-stat-monitor/user_guide.html).

### Configure monitoring extension

=== "pg_stat_statements"

    pg_stat_statements is the built-in PostgreSQL extension for tracking query performance, available as part of the postgresql-contrib package.
    {.power-number}
    
    1. Install the required package:
        -  Debian/Ubuntu: `apt install -y postgresql-contrib`
        - Red Hat/CentOS: `yum install -y postgresql-contrib`
    
    2. Add these lines to your `postgresql.conf` file:
    
        ```conf
        shared_preload_libraries = 'pg_stat_statements'
        track_activity_query_size = 2048 # Increase tracked query string size
        pg_stat_statements.track = all   # Track all statements including nested
        track_io_timing = on             # Capture read/write stats
        ```
    
    3. Restart the PostgreSQL server:
    
        ```sh
        systemctl restart postgresql
        ```
    
    4. Create the extension:
    
        ```sh
        psql postgres postgres -c "CREATE EXTENSION pg_stat_statements SCHEMA public"
        ```
        
        !!! note "Best practice"
            Create the extension in the `postgres` database to access statistics from all databases without configuring each one individually.

=== "pg_stat_monitor"

    pg_stat_monitor is Percona’s advanced PostgreSQL monitoring extension that enhances observability with detailed query metrics and improved aggregation. It is compatible with PostgreSQL and Percona Distribution for PostgreSQL versions 11 through 15.
    {.power-number}
    
    1. Install the extension:
    
        - For [Percona Distribution for PostgreSQL][PERCONA_POSTGRESQL_INSTALL]: Install via your Linux package manager after setting up the Percona repository, making sure to use the command with your actual PostgreSQL version: `apt install -y pg-stat-monitor-15`                    
        - For [Standard PostgreSQL][PG_STAT_MONITOR_INSTALL]: Download and compile from source code.
    
    2. Configure PostgreSQL settings in `postgresql.conf`:
    
        ```ini
        # Add to shared libraries
        shared_preload_libraries = 'pg_stat_monitor'
        
        # Required for PMM
        pg_stat_monitor.pgsm_query_max_len = 2048
        
        # Recommended settings
        pg_stat_monitor.pgsm_normalized_query = 1
        pg_stat_monitor.pgsm_enable_query_plan = 1
        ```
        
        !!! caution "Using both extensions?"
            If using with pg_stat_statements, list it first:
            ```ini
            shared_preload_libraries = 'pg_stat_statements, pg_stat_monitor'
            ```
    
    3. Restart your PostgreSQL instance:
    
        ```sh
        systemctl restart postgresql
        ```
    
    4. Create the extension:
    
        ```sql
        psql -d postgres -c "CREATE EXTENSION pg_stat_monitor;"
        ```
    
    5. Verify the installation:
    
        ```sql
        SELECT pg_stat_monitor_version();
        ```
        
    #### About bucket-based aggregation
    
    `pg_stat_monitor` uses buckets to collect and aggregate statistics:
    
    - Each bucket collects data for a configurable time period
    - When a bucket expires, data moves to the next bucket in the chain
    - When all buckets are full, the oldest bucket is reused
    - If a bucket fills before expiring, excess data is discarded

## Add service to PMM

After configuring your database server with the appropriate extension, you need to add it as a service to PMM. You can do this either through the PMM user interface or via the command line.

=== "Using the PMM user interface"

    To add the service from the user interface:
    {.power-number}
    
    1. Go to  **PMM Configuration > Add Service > PostgreSQL**.
    
    2. Enter or select values for the fields.
    
    3. Click **Add service**.
    ![!](../../../images/PMM_Add_Instance_PostgreSQL.jpg)

    4. If using TLS, check **Use TLS for database connections** and fill in your TLS certificates and key.
        
    For TLS connection to work SSL needs to be configured in your PostgreSQL instance. 
    
    Make sure SSL is enabled in the server configuration file `postgresql.conf`, and that hosts are allowed to connect in the client authentication configuration file `pg_hba.conf`. 
    See PostgreSQL documentation on [Secure TCP/IP Connections with SSL].

=== "Using the command line"

    === "Basic setup"
    
        Add an instance with default node name:
        
        ```sh
        pmm-admin add postgresql \
        --username=pmm \
        --password=password \
        --server-url=https://admin:admin@X.X.X.X:443 \
        --server-insecure-tls
        ```
        
        The service name will be automatically generated based on the node name.
    
    === "Custom service name"
    
        Add an instance with a specified service name:
        
        ```sh
        pmm-admin add postgresql \
        --username=pmm \
        --password=password \
        --server-url=https://admin:admin@X.X.X.X:443 \
        --server-insecure-tls \
        --service-name=SERVICE-NAME
        ```
    
    === "UNIX socket connection"
    
        Add an instance using a UNIX socket:
        
        ```sh
        pmm-admin add postgresql --socket=/var/run/postgresql
        ```
        
        Where:

        - `/var/run/postgresql`: Directory containing the socket
    
    === "TLS connection"
    
        Add an instance with TLS security:
        
        ```sh
        pmm-admin add postgresql --tls \
        --tls-cert-file=PATHTOCERT \
        --tls-ca-file=PATHTOCACERT \
        --tls-key-file=PATHTOKEY \
        --host=HOST \
        --port=PORT \
        --username=USER \
        --service-name=SERVICE-NAME
        ```
        
        where:

        - `PATHTOCERT`: Path to client certificate file
        - `PATHTOCACERT`: Path to certificate authority file
        - `PATHTOKEY`: Path to client key file
        - `HOST`: Instance hostname or IP
        - `PORT`: PostgreSQL service port number
        - `USER`: Database user allowed to connect via TLS (should match the CN in the client certificate)
        - `SERVICE-NAME`: Name to give to the service within PMM

### Auto-discovery configuration

Auto-discovery dynamically identifies all databases in your PostgreSQL instance. This feature helps balance comprehensive monitoring with resource efficiency.

!!! caution "Performance Impact"
    Limiting auto-discovery may result in fewer metrics being captured from the non-primary databases. Ensure that you set the limit appropriately:

    - **High limits** may impact performance by creating too many connections
    - **Low limits** may result in missing metrics from non-primary databases

=== "Command line configuration"

    The `pmm-admin` flag controls Auto-discovery behavior:
    
    ```sh
    --auto-discovery-limit=XXX
    ```
    
    How the limit works:
    
    - If number of databases > Auto-discovery limit: Auto-discovery is **OFF**
    - If number of databases <= Auto-discovery limit: Auto-discovery is **ON**
    - If Auto-discovery limit is not defined: Default value is 0 (server-defined with limit 10)
    - If Auto-discovery limit < 0: Auto-discovery is **OFF**
    
    **Example**:
    
    ```sh
    pmm-admin add postgresql \
    --username="pmm" \
    --password="password" \
    --auto-discovery-limit=10
    ```
    
    If your PostgreSQL instance has 11 databases, automatic discovery will be disabled.

=== "UI configuration"

    By default, **Auto-discovery** is enabled with a server-defined limit of 10 databases.
    
    ![Auto-discovery Enabled](../../../images/PMM_Add_Instance_PostgreSQL_autodiscovery_enabled.png)
    
    When you select **Disabled**, the **Auto-discovery limit** will be set to `-1`.
    
    ![Auto-discovery Disabled](../../../images/PMM_Add_Instance_PostgreSQL_autodiscovery_disabled.png)
    
    For a custom value, select **Custom** and enter your preferred limit.
    
    ![Auto-discovery Custom](../../../images/PMM_Add_Instance_PostgreSQL_autodiscovery_custom.png)

## Check the service

After adding a PostgreSQL service, verify that it's properly connected and sending data to PMM.

=== "Using command line"

    Run this command to view all services:
    
    ```sh
    pmm-admin inventory list services
    ```
    
    !!! hint "Docker environments"
        If using Docker, use: 
        ```sh
        docker exec pmm-client pmm-admin inventory list services
        ```
    
    Look for your PostgreSQL service in the output and verify that its status is "RUNNING".

=== "Using the PMM user interface"

    Use the UI to confirm that your service was added and is actively monitored:
    {.power-number}
    
    1. Select **Configuration > Inventory**
    
    2. In the **Services** tab, verify that **Service name** matches what you configured, **Address** points to your PostgreSQL instance and **Status** shows as "Active".
    
    3. In the **Options** column, expand the **Details** section to check that agents are properly registered and that the expected data source is being used.

=== "Check the dashboard data"

    Ensure PostgreSQL metrics are flowing and visualized correctly:
    {.power-number}
    
    1. Open the **PostgreSQL Instance Summary** dashboard
    
    2. Select your service name from the dropdown
    
    3. Verify that metrics are being displayed
    
    4. Check that the graphs are updating with current data

### Running custom queries

The PostgreSQL exporter can execute custom queries to collect additional metrics beyond what PMM provides by default.

=== "Configuration basics"

    Custom queries must be defined in this directory on the host where the exporter is running:
    ```
    /usr/local/percona/pmm/collectors/custom-queries/postgresql
    ```
    
    Three resolution directories are available:
    
    | Directory | Execution Frequency |
    |-----------|---------------------|
    | `high-resolution/`   | Every 5 seconds  |
    | `medium-resolution/` | Every 10 seconds |
    | `low-resolution/`    | Every 60 seconds |
    
    Choose the appropriate directory based on how frequently you need the data.

=== "Query definition format"

    Create YAML files in the appropriate resolution directory with this structure:
    
    ```yaml
    query_name:
      query: "SELECT statement goes here"
      master: true|false  # Whether to run only on the master
      metrics:
        - metric_name:
            usage: "GAUGE|COUNTER|LABEL|MAPPEDMETRIC|DURATION"
            description: "Human-readable description of the metric"
    ```
    
    **Example**:
    
    ```yaml
    pg_postmaster_uptime:
      query: "select extract(epoch from current_timestamp - pg_postmaster_start_time()) as seconds"
      master: true
      metrics:
        - seconds:
            usage: "GAUGE"
            description: "PostgreSQL service uptime in seconds"
    ```

=== "Metric types"

    | Metric Type | Description | Use Case |
    |-------------|-------------|----------|
    | `GAUGE` | A value that can go up or down | Memory usage, connection count |
    | `COUNTER` | A cumulative value that only increases | Total queries, bytes transferred |
    | `LABEL` | A string value used for labeling | Database name, table name |
    | `MAPPEDMETRIC` | Maps a query result to a numeric value | State conversion (e.g., "on"=1, "off"=0) |
    | `DURATION` | A time duration | Query execution time, lock wait time |

## Related topics

- [`pmm-admin` man page for `pmm-admin add postgresql`](../../../use/commands/pmm-admin.md#postgresql)
- [Configuring Percona Repositories with percona-release][PERCONA_RELEASE]
- [Percona Blog -- Running Custom MySQL Queries in Percona Monitoring and Management][BLOG_CUSTOM_QUERIES_MYSQL]

[PostgreSQL]: https://www.postgresql.org/
[Percona Distribution for PostgreSQL]: https://www.percona.com/software/postgresql-distribution
[POSTGRESQL_VERSIONING]: https://www.postgresql.org/support/versioning/
[PERCONA_LIFECYCLE]: https://www.percona.com/services/policies/percona-software-platform-lifecycle/
[PERCONA_RELEASE]: https://www.percona.com/doc/percona-repo-config/percona-release.html
[PERCONA_POSTGRESQL_INSTALL]: https://www.percona.com/doc/postgresql/LATEST/installing.html
[PG_STAT_MONITOR_INSTALL]: https://github.com/percona/pg_stat_monitor#installation
[PMM_ADMIN]: ../../../use/commands/pmm-admin.md
[Secure TCP/IP Connections with SSL]: https://www.postgresql.org/docs/current/ssl-tcp.html
[BLOG_CUSTOM_QUERIES_MYSQL]: https://www.percona.com/blog/2020/06/10/running-custom-queries-in-percona-monitoring-and-management/