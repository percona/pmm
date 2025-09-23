# PostgreSQL Archiver is failing

## Description

For a robust and fault-tolerant database solution the implementation of PTIR is a cornerstone of design. For a PostgreSQL database, this is covered by the WAL archive process, which will capture the contents of each WAL segment file once it is filled, and save that data somewhere before the segment file is recycled for reuse. 

From the official documentation:

To enable WAL archiving, set the wal_level configuration parameter to replica or higher, archive_mode to on, and specify the shell command to use in the `archive_command` configuration parameter. In practice these settings will always be placed in the postgresql.conf file. In `archive_command`, `%p` is replaced by the path name of the file to archive, while `%f` is replaced by only the file name. (The path name is relative to the current working directory, i.e., the cluster's data directory.) Use `%%` if you need to embed an actual `%` character inthe command. The simplest useful command is something like:

```
archive_command = 'test ! -f /mnt/server/archivedir/%f && cp %p /mnt/server/archivedir/%f'  # Unix
archive_command = 'copy "%p" "C:\\server\\archivedir\\%f"'  # Windows
```

which will copy archivable WAL segments to the directory `/mnt/server/archivedir` in the above example.

The archive command should return a zero (0) code so PostgreSQL will assume the WAL was successfully archived and will remove it or recycle it, in this manner the WAL directory size will be consistent with no big differences. In the case the command returns a no zero code PostgreSQL won’t remove or recycle the WAL file and will retry the archive until it success, in the meantime the number of WAL files in the WAL directory will increase and thus the occupied space, with the risk of run out of space.


## Resolution

From the PostgreSQL log we should be able to get insight about the failure, we can even try the command manually to verify the result. 

Something to keep in mind is the archive command will be executed under the ownership of the same user that the PostgreSQL server is running as, so the paths, tools, scripts, etc., that are called or acceded from the archive command should be accessible for the user running the service (usually postgres).

In some situations where is critical to release space and avoid the catastrophic failure of running out of free space, the archive command can be set to the dummy value `/bin/true`. In a Linux/Unix environment, this returns a zero code with no actual action.

This is enough for PostgreSQL to consider the WAL segment archived and remove/recycle it. 

While it might help with urgent needs, it is very dangerous: **be aware that doing this will break the continuity of the archive**.

Basically the PITR is no longer an option unless the initial issue is solved and a new physical backup (filesystem snapshot, pg_basebackup) is taken.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }