# MySQL Temporary tables dimension is capped by max_heap_table_size

## Description

MySQL can create internal in-memory temporary tables as part of normal query execution. If an internal in-memory temporary table grows larger than the defined size, it is converted to an on-disk internal temporary table automatically, which can impact performance.

The size limit of an in-memory temporary table is defined by the smaller value of either **tmp_table_size** and **max_heap_table_size**.

Consider setting these two variables to the same value.

## Resolution

Set the **tmp_table_size** to a value that is equal to or less than the **max_heap_table_size** value.
Or increase the value of the **max_heap_table_size** value to match **tmp_table_size** value. 

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }