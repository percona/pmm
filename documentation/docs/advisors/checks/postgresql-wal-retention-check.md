# Postgres WAL retention check
## Description

This check analyses the number of WAL files being retained in the **pg_wal** directory and reports the number of WAL files and the disk space they consume. 

The check lists the WAL files with `pg_ls_waldir()`, so it works for a monitoring user with the `pg_monitor` role. PostgreSQL 9.x has no `pg_ls_waldir()` and has reached end of life, so on it the check returns a warning that it needs PostgreSQL 10 or later.

Additionally, the check provides information on the number of **wal_keep_segments** or **wal_keep_size** configured depending on the version of Postgres. A calculation is performed against the parameters mentioned, and the potential disk space used as a result of the parameter values and **wal_segment_size** for the running cluster.


## Resolution

There are various reasons WAL files can accumulate in the WAL directory and it is up to the administrator to decide if the reported number of WAL files is a concern. 

Typically, WAL file accumulation occurs when the following situations arise:

- A replica using replication slots is offline and the replication slot has not been removed.
- Archiving of WAL files is failing and the WAL files are being retained until resolved.
- **Wal_keep_segments** or **wal_keep_size** has been configured to a value other than **0**.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


