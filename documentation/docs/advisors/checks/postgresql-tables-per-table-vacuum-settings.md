# Check the tables that have “per table” vacuum settings

## Description
This advisor returns the list of tables that specify table-level autovacuum parameters. Their associated table-level settings are also listed. 

Table-level settings override the global settings. This can lead to difficult troubleshooting, unexpected behaviors, and incidents. Therefore, it is very important to check if tables in the database have any specific, table-level settings. 

The table-level parameters can be adjusted according to the current autovacuum statistics.


## Rule
N/A (checks framework provides output for type POSTGRESQL_SHOW).

## Resolution
The results of this advisor check can be correlated with other sets of checks or behaviours.

In case some of the parameters values are off, they can be corrected using the following syntax:

``` yaml
ALTER TABLE <tablename> SET (autovacuum_analyze_scale_factor = <val>, autovacuum_vacuum_scale_factor = <val>, autovacuum_vacuum_threshold = <val>, autovacuum_analyze_threshold = <val>); 
```

Replace **tablename** with the actual table name, and **val** with the actual parameter value. 

For the full list of table-level autovacuum parameters, see [Automatic Vacuuming](https://www.postgresql.org/docs/current/runtime-config-autovacuum.html) in the PostgreSQL documentation.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }