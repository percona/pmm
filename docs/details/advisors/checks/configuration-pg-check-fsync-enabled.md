# fsync is disabled on PostgreSQL configuration

## Description
If this parameter is enabled (which it is by default), the PostgreSQL server will try to make sure that updates are physically written to disk, by issuing fsync() system calls or various equivalent methods mentioned in wal_sync_method. This ensures that the database cluster can recover to a consistent state after an operating system or hardware crash. Even though this requires extra effort, it is a cost worth paying for critical environments.

Turning off fsync can give some performance performance improvement with the risk of an unrecoverable data corruption in the event of a power failure or system crash. Thus it is only advisable to turn off fsync if you can easily recreate your entire database from external data.
High quality hardware alone is not a sufficient justification for turning off fsync.

## Resolution
Please Perform the steps mentioned below to enable fsync. 

- Alter the configuration parameter as superuser
```
ALTER SYSTEM SET fsync = on;
```

- Signal the PostgreSQL server to reload the configuration
```
SELECT pg_reload_conf();
```

- Flush all the pages safely to disk using the initdb utility
```
$ initdb -D $PGDATA --sync-only
syncing data to disk … ok
```

- 
Run sync on the filesystem
```
$ sync
```
 