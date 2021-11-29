# MySQL check binlog sync status

## Description
This means that it’s not guaranteed that for a transaction that is flushed to disk the binary logs are flushed as well. This means that on a crash, there might be transactions applied to the instance’s data which the instance has no binlog for.

[MySQL documentation](https://dev.mysql.com/doc/refman/5.7/en/replication-options-binary-log.html#sysvar_sync_binlog)
[Percona blog posts on the topic:](https://www.percona.com/blog/2018/05/04/how-binary-logs-and-filesystems-affect-mysql-performance/)



## Rule
`SELECT @@global.sync_binlog;`


## Resolution
Consider setting the sync_binlog variable to 1 with `SET GLOBAL sync_binlog=1`.
