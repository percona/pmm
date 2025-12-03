# MySQL binary logs automatically removed too quickly

## Description

Checks that binary logs are kept for at least one day before being purged.

For more information, see the [MySQL documentation](https://dev.mysql.com/doc/refman/8.4/en/replication-options-binary-log.html#sysvar_binlog_expire_logs_seconds).

## Resolution

Consider increasing binlog retention period by increasing **binlog_expire_logs_seconds**/**expire_logs_days**.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
