# MySQL Temporary tables dimension is capped by max_heap_table_size

## Description
MySQL can create temporary tables as part of normal query execution or with the CREATE TEMPORARY TABLE sql command. The maximum size of an in-memory temporary table is controlled by the smallest of tmp_table_sizeand max_heap_table_size. When the smaller of the two is exceeded, the temporary table is converted to an on-disk one, which has a performance impact. Consider setting these 2 variables the same value.


## Rule
`SELECT @@global.max_heap_table_size, @@global.tmp_table_size`


## Resolution
Set the tmp_table_size to a value that is equal to or less than max_heap_table_size value.
Or increase the value of max_heap_table_size value to match tmp_table_size value. 

