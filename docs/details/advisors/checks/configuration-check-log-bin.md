# MySQL Redo log (binary log) not enabled

## Description
The binary logs are the change stream of the MySQL instance. It is used for replication and for point in time recovery. It’s recommended to have them on for point in time recovery purposes, and so replicas can be promoted to primaries. This database instance cannot be a replication source, nor could it provide the needed logs for point in time recovery. Consider enabling the binary logs.

[MySQL documentation](https://dev.mysql.com/doc/refman/8.0/en/binary-log.html)


## Rule
`SELECT @@global.log_bin;`


## Resolution
Turn on binary logging by having the log_bin option on the configuration file, and restart the MySQL instance. 

