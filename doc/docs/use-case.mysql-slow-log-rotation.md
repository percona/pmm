# Use **logrotate** instead of the slow log rotation feature to manage the MySQL Slow Log

By default, PMM manages the slow log for the added MySQL monitoring service on the computer where PMM Client is installed. This example demonstrates how to substitute **logrotate** for this default behavior.

## Disable the default behavior of the slow log rotation

The first step is to disable the default slow log rotation when adding the MySQL monitoring service.

For this, set the `--slow-log-rotation` to *false*.

Run this command as root or by using the **sudo** command

```
pmm-admin rm mysql:queries
pmm-admin add mysql:queries --slow-log-rotation=false
```

On PMM Server, you can check the value of the Slow logs rotation field on the QAN Settings page. It should be *OFF*.

On PMM Client (the host where you ran **pmm-admin add** command to add the MySQL monitoring service), use the **pmm-admin list** command to determine if the *slow log* rotation is disabled.

```
$ pmm-admin list

PMM Server      | 127.0.0.1
Client Name     | percona
Client Address  | 172.17.0.1
Service Manager | linux-systemd

-------------- -------- ----------- -------- ------------------------------------------- --------------------------------------------------------------------------------------
SERVICE TYPE   NAME     LOCAL PORT  RUNNING  DATA SOURCE                                 OPTIONS
-------------- -------- ----------- -------- ------------------------------------------- --------------------------------------------------------------------------------------
mysql:queries  percona  -           YES      root:***@unix(/var/run/mysqld/mysqld.sock)  query_source=slowlog, query_examples=true, slow_log_rotation=false, retain_slow_logs=1
```

## Set up **logrotate** to manage the slow log rotation

**logrotate** is a popular utility for managing log files. You can install it using the package manager (apt or yum, for example) of your Linux distribution.

After you add a MySQL with `--slow-log-rotation` set to **false**, you can run **logrotate** as follows.

Run this command as root or by using the **sudo** command

```
$ logrotate CONFIG_FILE
```

*CONFIG_FILE* is a placeholder for a configuration file that you should supply to **logrotate** as a mandatory parameter. To use **logrotate** to manage the *slow log* for PMM, you may supply a file with the following contents.

This is a basic example of **logrotate** for the MySQL slow logs at 1G for 30 copies (30GB).

```
/var/mysql/mysql-slow.log {
             nocompress
             create 660 mysql mysql
             size 1G
             dateext
             missingok
             notifempty
             sharedscripts
             postrotate
             /bin/mysql -e 'SELECT @@global.long_query_time INTO @LQT_SAVE; \
             SET GLOBAL long_query_time=2000; SELECT SLEEP(2); \
             FLUSH SLOW LOGS; SELECT SLEEP(2); SET GLOBAL long_query_time=@LQT_SAVE;'
             endscript
             rotate 30
}
```

For more information about how to use **logrotate**, refer to its documentation installed along with the program.
