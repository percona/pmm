# Advisors

## InnoDB Flush Method May Not Be Optimal

### Problem Description

By default, InnoDB's log buffer is written out to the log file at each transaction commit and a flush-to-disk operation is performed on the log file, which enforces ACID compliance. In the event of a crash, if you can afford to lose a second's worth of transactions, you can achieve better performance by setting `__innodb_flush_log_at_trx_commit__` to either 0 or 2. If you set the value to 2, then only an operating system crash or a power outage can erase the last second of transactions. This can be very useful on slave servers, where the loss of a second's worth of data can be recovered from the master server if needed.

### Links and Further Reading

[Variables](http://dev.mysql.com/doc/mysql/en/innodb-parameters.html#optvar_innodb_flush_log_at_trx_commit) MySQL Manual: [InnoDB Performance Tuning Tips](http://dev.mysql.com/doc/mysql/en/optimizing-innodb.html)

### Expression

```
(%innodb_flush_log_at_trx_commit% == THRESHOLD)
```