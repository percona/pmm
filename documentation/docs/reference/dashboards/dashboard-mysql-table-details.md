# MySQL Table Details

![!image](../../images/PMM_MySQL_Table_Details.jpg)

## Largest Tables

Largest Tables by Row Count
: The estimated number of rows in the table from `information_schema.tables`.

Largest Tables by Size
: The size of the table components from `information_schema.tables`.

## Pie

Total Database Size
: The total size of the database: as data + index size, so freeable one.

Most Fragmented Tables by Freeable Size
: The list of 5 most fragmented tables ordered by their freeable size

## Table Activity

**Top Tables by Rows Read** shows the five tables with the highest read activity. Use it to identify your most-read tables, which benefit most from caching, indexing, or read replicas.

**Top Tables by Rows Changed** shows the five tables with the most write activity (inserts, updates, and deletes combined). Use it to pinpoint write hotspots that may be causing lock contention, replication lag, or I/O pressure.

### Data source compatibility

On [Percona Server](https://www.percona.com/doc/percona-server/5.6/diagnostics/user_stats.html) and [MariaDB](https://mariadb.com/docs/server/ha-and-performance/optimization-and-tuning/query-optimizations/statistics-for-optimizing-queries/user-statistics), these panels use `INFORMATION_SCHEMA.TABLE_STATISTICS` when `userstat` is enabled. On Oracle MySQL, including MySQL 9.7, they use `performance_schema.table_io_waits_summary_by_table`.

Both data sources require per-table statistics collection. PMM disables this for services that exceed the configured table statistics limit, and these panels show no data for those services.

## Rows read

The number of rows read from the table, shown for the top 5 tables.

## Rows Changed

The number of rows changed in the table, shown for the top 5 tables.

## Auto Increment Usage

The current value of an `auto_increment` column from `information_schema`, shown for the top 10 tables.
