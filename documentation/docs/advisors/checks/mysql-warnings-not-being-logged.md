# Warnings are not being logged

## Description

The **log_error_verbosity** setting determines if the error log contains ERROR, WARNING, and INFORMATION messages. If WARNING messages are not printed to the error log important information may be ignored.

See [log_error_verbosity](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_log_error_verbosity) for more information

## Resolution

Please consider setting **log_error_verbosity** to a value of 2 or larger (log_warnings >= 2 for MariaDB) to avoid ignoring WARNING messages.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
