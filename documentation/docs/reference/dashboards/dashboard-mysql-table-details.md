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

**Top Tables by Rows Read** and **Top Tables by Rows Changed** show the top 5 tables by row activity.

On Community MySQL, these panels use `performance_schema.table_io_waits_summary_by_table`. On [Percona Server](https://www.percona.com/doc/percona-server/5.6/diagnostics/user_stats.html) and [MariaDB](https://mariadb.com/docs/server/ha-and-performance/optimization-and-tuning/query-optimizations/statistics-for-optimizing-queries/user-statistics), they use user statistics. To enable user statistics, set `userstat=ON`.

## Rows read

The number of rows read from the table, shown for the top 5 tables.

## Rows Changed

The number of rows changed in the table, shown for the top 5 tables.

## Auto Increment Usage

The current value of an `auto_increment` column from `information_schema`, shown for the top 10 tables.
