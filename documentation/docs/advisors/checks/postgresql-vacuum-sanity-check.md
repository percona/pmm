# PostgreSQL Vacuum sanity


## Description

This checks basic VACUUM parameters and notifies if default values are found. In addition, it alerts you if the _autovacuum_ feature is not enabled in your settings.

## Resolution
* Autovacuuming is critical for PostgreSQL to keep table bloat and transaction wraparound under control. Disabling autovacuum could lead to serious table bloat, performance issues, and an accelerated path to transaction wraparound. Typically, the autovacuum parameter is turned off only because there is a very specific and special reason for it, and most likely another task may be vacuuming on a different schedule. If this is not the case, consider enabling this as soon as possible.
*   If **autovacuum\_max\_workers** is set to the default value of **3**, and you find that the vacuum is not cleaning up as expected, adjust this parameter to increase the autovacuum process count.
*   The default value for **autovacuum\_scale\_factor** is 20%. This means that a table will not get autovacuumed until 20% of the data in it has been updated or deleted + a threshold.  For small tables, this is not too much of a concern. However, on large tables, this can be problematic. 

In order to mitigate this at the table level, identify large tables that are not being vacuumed frequently and apply custom storage parameters to the table.  For example: 

`ALTER TABLE foobar SET ( autovacuum\_vacuum\_scale\_factor = .1)`

The above will set the scale factor to 10% on the foobar table.  

Another option is to set the autovacuum to take place when _**n**_ number of rows have been updated or deleted.

`ALTER TABLE foobar SET ( autovacuum\_vacuum\_scale\_factor = 0, autovacuum\_vacuum\_threshold = 100000 )`

In the above example, the foobar table will autovacuum when 100,000 rows have been updated or deleted. 

For vacuum cost settings, keep the following in mind.

During the execution of VACUUM and ANALYZE commands, the system maintains an internal counter that keeps track of the estimated cost of the various I/O operations that are performed. When the accumulated cost reaches a limit specified in the **vacuum\_cost\_limit** parameter, the process performing the operation will sleep for a time, as specified by **vacuum\_cost\_delay**. Then it will reset the counter and continue execution.

These costs are determined using the following logic and the default settings which can be changed as shown below:

*   vacuum\_cost\_page\_hit = 1
*   vacuum\_cost\_page\_miss = 10
*   vacuum\_cost\_page\_dirty = 20

**How the costs are calculated.**

*   **vacuum\_cost\_page\_hit** is the cost for vacuuming a buffer found in the shared buffer cache. It represents the cost to lock the buffer pool, look up the shared hash table and scan the content of the page. Unless changed, the default value of 1 gets added to the counter.
*   **vacuum\_cost\_page\_miss** is the cost for vacuuming a buffer that has to be read from disk. This represents the effort to lock the buffer pool, look up the shared hash table, read the desired block in from the disk and scan its content. Unless changed, the default value of 10 gets added to the counter.
*   **vacuum\_cost\_page\_dirty** is the cost when vacuum modifies a block that was previously clean. It represents the extra I/O required to flush the dirty block out to disk again. Unless changed, the default value of 20 gets added to the counter.

This feature aims to enable administrators to reduce the I/O impact of these commands on concurrent database activity.

There are many situations where it is not important that maintenance commands like VACUUM and ANALYZE finish quickly. However, it is usually very important that these commands do not significantly interfere with the ability of the system to perform other database operations. Cost-based vacuum delay provides a way for administrators to achieve this.

This feature is disabled by default for manually issued VACUUM commands. To enable it, set the **vacuum\_cost\_delay** variable to a non-zero value.


## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
