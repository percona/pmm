# Replication privileges

## Description

The **REPLICATION_SLAVE** privileges enable a user account to connect from a replica to a replication source server.

A standard user should not be granted **REPLICATION SLAVE** but instead be granted the **REPLICATION CLIENT** to access replication information.

The check lists the users assigned the **REPLICATION SLAVE** grant along with other grants.

## Resolution

Revoke **REPLICATION SLAVE** from a standard user or any additional grant from the replication user.

For example, to see the assigned grants for a replica and to revoke a grant:

```sql
mysql> SHOW GRANTS for replica@'%';
+---------------------------------------------------------------------+
| Grants for replica@%                                                |
+---------------------------------------------------------------------+
| GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO **replica**@**%** |
+---------------------------------------------------------------------+

mysql> REVOKE REPLICATION SLAVE on *.* from 'replica'@'%';
Query OK, 0 rows affected (0.10 sec)
```

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
