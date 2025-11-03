# MySQL enforced data integrity checking is disabled

## Description

Enable the **slave_sql_verify_checksum** causes the replication SQL (applier) thread to verify data using the checksums read from the relay log.

If there's mismatch, the replica stops with an error. 

Setting this variable takes effect for all replication channels immediately, including running channels.

## Resolution

Set the **slave_sql_verify_checksum** variable to **1**.

For MySQL 8.4 and later: Set the **replica_sql_verify_checksum** variable to **1**.  See the [MySQL documentation](https://dev.mysql.com/doc/refman/8.4/en/replication-options-replica.html#sysvar_replica_sql_verify_checksum) for details.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

