# MySQL binlog_row_image set to MINIMAL

## Description

This variable, in row-based replication, determines if row images are written to the blog as **full** (log all columns), **minimal** (Log only changed columns and columns used to identify rows), or **noblob** (log all columns except BLOB or TEXT columns).

Setting **binlog_row_image** to MINIMAL reduces the amount of data pushed into the binary log. However, this setting also skips essential data used to recover your database from data corruption, or human mistakes.

Manually change the value of **binlog_row_image** in the my.cnf file.
******text
[mysql]

binlog_format=ROW
binlog_row_image =FULL
******

Close sessions and restart the server.

To set the variable for a session, use the following command:

```sql

mysql> SET SESSION binlog_row_image='FULL';
```

Remember, if you close the session, the **binlog_row_image** setting returns to the server setting.

For more information, see [flashback recovery](https://mydbops.wordpress.com/2019/05/22/flashback-recovery-in-mariadb-mysql-percona/) and the [MySQL manual](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_binlog_row_image).

On the other side, setting the **binlog_row_image** to FULL can result in a significant increase of data if you use many and large BLOB/TEXT columns that do not change often.  
Therefore, although best practice recommends using FULL, in some cases using MINIMAL is fine.

## Resolution

Consider setting **binlog_row_image=FULL** to increase the chances of successful data recovery.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
