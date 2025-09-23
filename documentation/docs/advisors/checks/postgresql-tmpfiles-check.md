# PostgreSQL temporary files written to disk check
## Description
This check reports details about the temporary files and total space used. You can use this information for fine-tuning the **work_mem** parameter.


## Resolution
There are numerous tunable parameters that can affect the number of temporary files created. However, the **work_mem** parameter is the most commonly targeted one. This parameter defaults to **4MB** and sets the base maximum amount of memory to be used by a query operation before writing to temporary disk files. For example, Sort or Hash table operations.

For complex queries, several Sort or Hash operations could run in parallel. 
Before starting to write data into temporary files, each operation will generally be allowed to use as much memory as this value specifies. 

In addition, several running sessions could be doing such operations concurrently. Therefore, the total memory used could be many times the value of **work_mem**. Make sure to consider this when choosing the value. 

Sort operations are used for ORDER BY, DISTINCT, and merge joins. Hash tables are used in hash joins, hash-based aggregation, and hash-based processing of IN subqueries.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
