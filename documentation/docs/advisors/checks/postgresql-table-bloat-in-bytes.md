# Check Table bloat in bytes

## Description

This advisor check verifies the table bloat from the currently connected database.

It returns a WARNING if any table in the current database has a bloat over 1GB size and it represents at least 20% of the total table size. This is to avoid messages for 1GB bloat on huge tables.


!!! warning alert alert-success "Warning"
    At the moment of creation of this advisor, the PMM advisor engine connects only to the *pmm* database, so this advisor only checks that database. We would need the engine to be able to establish connections to each database in the PostgreSQL instance to get information.

Bloat is a natural side effect of how the Multiversion Concurrency Control (MVCC) works in a PostgreSQL instance. 

So every time a tuple gets updated or deleted, a "dead" version of it gets created within the data pages. These dead tuples are tracked in the visibility map, and the regular VACUUM operations should be enough to release them and mark them as free space so new inserts can reuse the space. 

There are some situations where the VACUUM cannot catch up with the volume of changes, and the number of dead tuples starts to rise. In such conditions, after some time, the table gets bloated because of all this unused and wasted space. 

When a table gets bloated, this affects the general performance of the queries over the table. This is because accessing the data needs to go through all the existing dead tuples, adding time to the read operations. 


## Rule 

```yaml 
POSTGRESQL_SELECT

* FROM (
WITH a AS (
SELECT current_database(), schemaname, tblname, tblpages as pages,est_tblpages as est_pages ,est_tblpages_ff as est_pages_ff,
bs*tblpages AS cur_size_byte, (tblpages-est_tblpages)*bs AS free_bytes,
CASE WHEN tblpages - est_tblpages > 0
    THEN 100 * (tblpages - est_tblpages)/tblpages::float
    ELSE 0
END AS extra_pages_pct, fillfactor,
CASE WHEN tblpages - est_tblpages_ff > 0
    THEN (tblpages-est_tblpages_ff)*bs
    ELSE 0
END AS bloatsize_byte,
CASE WHEN tblpages - est_tblpages_ff > 0
    THEN 100 * (tblpages - est_tblpages_ff)/tblpages::float
    ELSE 0
END AS bloat_pct, is_na AS stats_missing
FROM (
SELECT ceil( reltuples / ( (bs-page_hdr)/tpl_size ) ) + ceil( toasttuples / 4 ) AS est_tblpages,
    ceil( reltuples / ( (bs-page_hdr)*fillfactor/(tpl_size*100) ) ) + ceil( toasttuples / 4 ) AS est_tblpages_ff,
    tblpages, fillfactor, bs, tblid, schemaname, tblname, heappages, toastpages, is_na
FROM (
    SELECT
    ( 4 + tpl_hdr_size + tpl_data_size + (2*ma)
        - CASE WHEN tpl_hdr_size%ma = 0 THEN ma ELSE tpl_hdr_size%ma END
        - CASE WHEN ceil(tpl_data_size)::int%ma = 0 THEN ma ELSE ceil(tpl_data_size)::int%ma END
    ) AS tpl_size, bs - page_hdr AS size_per_block, (heappages + toastpages) AS tblpages, heappages,
    toastpages, reltuples, toasttuples, bs, page_hdr, tblid, schemaname, tblname, fillfactor, is_na
    FROM (
    SELECT
        tbl.oid AS tblid, ns.nspname AS schemaname, tbl.relname AS tblname, tbl.reltuples,
        tbl.relpages AS heappages, coalesce(toast.relpages, 0) AS toastpages,
        coalesce(toast.reltuples, 0) AS toasttuples,
        coalesce(substring(
        array_to_string(tbl.reloptions, ' ')
        FROM 'fillfactor=([0-9]+)')::smallint, 100) AS fillfactor,
        current_setting('block_size')::numeric AS bs,
        CASE WHEN version()~'mingw32' OR version()~'64-bit|x86_64|ppc64|ia64|amd64' THEN 8 ELSE 4 END AS ma,
        24 AS page_hdr,
        23 + CASE WHEN MAX(coalesce(s.null_frac,0)) > 0 THEN ( 7 + count(s.attname) ) / 8 ELSE 0::int END
        + CASE WHEN bool_or(att.attname = 'oid' and att.attnum < 0) THEN 4 ELSE 0 END AS tpl_hdr_size,
        sum( (1-coalesce(s.null_frac, 0)) * coalesce(s.avg_width, 0) ) AS tpl_data_size,
        bool_or(att.atttypid = 'pg_catalog.name'::regtype)
        OR sum(CASE WHEN att.attnum > 0 THEN 1 ELSE 0 END) <> count(s.attname) AS is_na
    FROM pg_attribute AS att
        JOIN pg_class AS tbl ON att.attrelid = tbl.oid
        JOIN pg_namespace AS ns ON ns.oid = tbl.relnamespace
            AND ns.nspname NOT IN ('pg_catalog', 'information_schema')
        LEFT JOIN pg_stats AS s ON s.schemaname=ns.nspname
        AND s.tablename = tbl.relname AND s.inherited=false AND s.attname=att.attname
        LEFT JOIN pg_class AS toast ON tbl.reltoastrelid = toast.oid
    WHERE NOT att.attisdropped
        AND tbl.relkind in ('r','m')
    GROUP BY 1,2,3,4,5,6,7,8,9,10
    ORDER BY 2,3
    ) AS s
) AS s2
) AS s3
ORDER BY schemaname, tblname)
SELECT  current_database,
        schemaname,
        tblname,
        cur_size_byte real_size,
        pg_size_pretty(cur_size_byte::numeric) real_size_pretty, 
        free_bytes, 
        bloatsize_byte bloat_size_byte,
        pg_size_pretty(bloatsize_byte::numeric) bloat_size_byte_pretty
FROM a
WHERE schemaname != 'pg_catalog'
AND bloatsize_byte > 1073741824     -– 1GB
ORDER BY cur_size_byte DESC
) x

```
## Resolution
When a table accumulates significant bloat, running VACCUM will be insufficient. The solution is to rebuild the table so all the unused space from the dead tuples is eradicated. 

There are two main ways to do this:

- Running VACUUM FULL on the bloated table.
- Executing [pg_repack](https://reorg.github.io/pg_repack/) in the bloated table.

In both cases, the table gets physically rebuilt. But you need to consider that the VACUUM FULL operation will hold an exclusive lock on the table, preventing any other operation until it finishes. This could impact a running production service and generally is undesired.

The **pg_repack** approach is much less intrusive than the VACUUM FULL, it only acquires brief locks on the table, and all the rebuild operations happen online on a "shadow" table. Once it is complete, the process just flips the tables, and the new one with no bloat becomes the active one. In many production environments, the **pg_repack** option is the chosen one. 

Remember that using **pg_repack** requires the table to have a PK, or at least a UNIQUE total index on a NOT NULL column. 

Also, suppose the table is the source of a logical replication (publication). 
In that case, it is recommended to stop the replication during the repack and resume it once the operation is finished to avoid unexpected effects. 

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }