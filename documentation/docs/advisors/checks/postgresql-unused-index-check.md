# Check for relations with unused indexes for PostgreSQL


## Description

This check scans all databases in the cluster and lists relations with indexes that have not been used since the last statistics reset.


## Resolution

Connect to a database and run the following query:

```
SELECT * FROM pg_stat_user_indexes 
```

The output lists the indexes for the current database. The column `idx_scan` indicates the number of times that the index has been used.

If you find unused indexes, check whether these indexes are needed, and take appropriate actions.  

Keep in mind that some indexes may be needed to address foreign key performance. For example deleting or updating a key would force Postgres to validate the constraint. Without an index, this could result in sequential scans. 


## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
