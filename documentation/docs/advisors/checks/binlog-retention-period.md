# MySQL binlog_expire_logs_seconds too low

## Description

The **binlog_expire_logs_seconds** variable sets, measured in seconds, the expiration period for a binary log. Binary log files can be removed when the period ends. A binary log may also be removed at server startup and when flushing the binary log. 

Setting the **binlog_expire_logs_seconds** variable to a low value can cause short rotation cycles for binary logs. These rotations can make Point In Time Recovery impossible.

Having short rotation cycles can also make maintenance of a replica difficult since the maintenance can only be performed in the rotation cycle.  

For more information, see [**binlog_expire_logs_seconds** in the MySQL documentation](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_binlog_expire_logs_second). 


## Resolution

Consider increasing **binlog_expire_logs_seconds** to at least 604800 seconds (1 week). The default value is 2592000 seconds, which is 30 days.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
