# MySQL InnoDB Metrics

Collecting metrics and statistics for graphs increases overhead.  You can keep
collecting and graphing low-overhead metrics all the time, and enable
high-overhead metrics only when troubleshooting problems.

InnoDB metrics provide detailed insight about InnoDB operation.  Although you
can select to capture only specific counters, their overhead is low even when
they all are enabled all the time. To enable all InnoDB metrics, set the
global variable `innodb_monitor_enable` to `all`:

```sql
SET GLOBAL innodb_monitor_enable=all
```

!!! seealso "See also"

    [MySQL Server 5.7 Documentation: innodb_monitor_enable](https://dev.mysql.com/doc/refman/5.7/en/innodb-parameters.html#sysvar_innodb_monitor_enable)
