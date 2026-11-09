# Use custom queries on MySQL

Run custom SQL queries to collect metrics that PMM does not monitor by default, such as internal statistics, application-level data, or business metrics.

The MySQL exporter automatically reads query definitions from YAML files placed in a specific directory on the PMM Client host. The subdirectory you place the file in determines how often the query runs.

To set up a custom query:
{.power-number}

1. Place your query file in one of the following directories. The MySQL exporter reads files from these directories automatically, and the subdirectory you choose sets the collection frequency. Since all queries in a directory run sequentially, keep them fast to avoid missing the collection window:                                                 

    - `/usr/local/percona/pmm/collectors/custom-queries/mysql/high-resolution/` — every 5 seconds
    - `/usr/local/percona/pmm/collectors/custom-queries/mysql/medium-resolution/` — every 10 seconds
    - `/usr/local/percona/pmm/collectors/custom-queries/mysql/low-resolution/` — every 60 seconds


2. In the directory you chose, create a .yaml file to define the custom query the MySQL exporter will run. The file specifies the SQL query, a metric namespace to group the results under, and how each returned column maps to a metric. PMM builds the metric name by combining the namespace with the column name, for example `metric_namespace_col2`:

    ```yaml
    metric_namespace:
      query: "SELECT col1, col2 FROM your_table"
      metrics:
        - col1:
            usage: "LABEL"
            description: "Description of col1"
        - col2:
            usage: "GAUGE"
            description: "Description of col2"
    ```

    ??? example "Example: Collecting InnoDB index statistics"
        ```yaml
        mysql_innodb_index_stats:
          query: "SELECT database_name, table_name, index_name, stat_name, stat_value
                  FROM mysql.innodb_index_stats"
          metrics:
            - database_name:
                usage: "LABEL"
                description: "Database name"
            - table_name:
                usage: "LABEL"
                description: "Table name"
            - index_name:
                usage: "LABEL"
                description: "Index name"
            - stat_name:
                usage: "LABEL"
                description: "Statistic name"
            - stat_value:
                usage: "GAUGE"
                description: "Index statistic value in bytes"
        ```

        This produces metrics like `mysql_innodb_index_stats_stat_value{database_name="mydb", table_name="orders", ...}`.

3. For each column returned by your query, assign a metric type in the `usage` field:

    | Type | Description | Use case |
    |------|-------------|----------|
    | `GAUGE` | A value that can go up or down | Connection count, buffer pool size |
    | `COUNTER` | A cumulative value that only increases | Total queries executed, bytes written |
    | `LABEL` | A string dimension, not plotted as a metric | Database name, table name, status |
    | `DURATION` | A time duration | Query execution time, lock wait time |
    | `DISCARD` | Column is ignored and not exported | Columns returned by the query but not needed as metrics |

## Related topics

- [Running custom MySQL queries in PMM](https://www.percona.com/blog/running-custom-queries-in-percona-monitoring-and-management/)
