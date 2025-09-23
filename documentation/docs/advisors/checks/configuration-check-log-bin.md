# MySQL Redo log (binary log) not enabled

## Description

The binary logs are the change stream of the MySQL instance. It is used for replication and for point-in-time recovery.

It’s recommended to have them on for point-in-time recovery purposes, and so replicas can be promoted to primaries.

This database instance cannot be a replication source, nor could it provide the needed logs for point-in-time recovery. Consider enabling the binary logs.

For more information, see the [MySQL documentation](https://dev.mysql.com/doc/refman/8.0/en/binary-log.html).

## Resolution

Turn on binary logging by having the **log_bin** option on the configuration file, and restart the MySQL instance.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
