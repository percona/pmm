# Replica without replication slave account

## Description

If the source fails, you may want to use a replica as the new source. An account with the **REPLICATION SLAVE** privilege must exist for a server to act as a replication source and let a replica can connect to it.

Therefore, it is a good idea to create this account on your replicas to prepare them to take over for a source, if needed.

## Resolution

```sql
CREATE USER 'replication'@'192.168.0.%' IDENTIFIED BY 'password';
GRANT REPLICATION SLAVE ON *.* to `replication'@'192.168.0.%';
```

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
