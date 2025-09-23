# MySQL check binary log sync status

## Description

If the binary log synchronization to disk is disabled, the operating system flushes the binary log to disk as it does any other file. If the operating system crashes, or experiences a power failure, committed transactions may not be synchronized with the binary log.

For more information, see the [MySQL documentation](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_sync_binlog)
and [Percona blog posts on the topic](https://www.percona.com/blog/2018/05/04/how-binary-logs-and-filesystems-affect-mysql-performance/).

## Resolution

Consider changing **SET GLOBAL sync_binlog=1**.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

