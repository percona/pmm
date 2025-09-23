# Redo log is disabled in this instance

## Description

The MySQL InnoDB Redo log, is one of the core components to fulfil the ACID paradigm in MySQL. This element is currently OFF, the setting is highly insecure.

__Disabling Redo Logging__

As of MySQL 8.0.21, you can disable redo logging using the ALTER INSTANCE DISABLE INNODB REDO_LOG statement. This functionality is intended for loading data into a new MySQL instance. Disabling redo logging speeds up data loading by avoiding redo log writes and doublewrite buffering.

This feature is intended only for loading data into a new MySQL instance. Do not disable redo logging on a production system. It is permitted to shutdown and restart the server while redo logging is disabled, but an unexpected server stoppage while redo logging is disabled can cause data loss and instance corruption.

Attempting to restart the server after an unexpected server stoppage while redo logging is disabled is refused with the following error:
```
[ERROR] [MY-013598] [InnoDB] Server was killed when Innodb Redo 
logging was disabled. Data files could be corrupt. You can try 
to restart the database with innodb_force_recovery=6
```
In this case, initialize a new MySQL instance and start the data loading procedure again.

## Resolution

Enable the REDO LOG: 
```sql
ALTER INSTANCE ENABLE INNODB REDO_LOG;
```

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
