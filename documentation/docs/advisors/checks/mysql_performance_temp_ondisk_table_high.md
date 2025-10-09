# Performance check for On-Disk Temporary MySQL tables

## Description

Temporary tables on disk are slower than in-memory tables since disk I/O operations are inherently slower than memory operations. This can be especially noticeable if the temporary table is being used heavily, as the frequent reads and writes to the disk can cause performance issues. 

This performance impact can often be attributed to an un-optimized query or the absence of an index, among other factors. In such situations a query review or configuration change can be useful.

## Resolution

Perform a query review to identify which queries are causing the temporary tables.
Query reviews will help identify poorly written queries, table design issues or missing indexes, and will help optimize any queries that cause temporary disk tables.

Review **tmp_table_size** and **max_heap_table_size** only when the query review isn’t yielding results, and raisin in-memory temporary table types is absolutely necessary.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
