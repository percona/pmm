# Checks if InnoDB flush method is set correctly

## Description
Dirty pages are flushed to disk using the type of IO specified using `innodb_flush_method`:
- The default flush method (NULL) opens redo logs and data files using buffered IO and calls fsync() to flush data to disk when necessary for ACID compliance.
- When set to `O_DSYNC`, InnoDB will flush file data and metadata, and uses buffered IO for data files.
- When set to `O_DIRECT`, InnoDB opens redo log files with buffered IO and uses direct (unbuffered synchronous) IO on data files.


## Rule
`SELECT * from performance_schema.global_variables where VARIABLE_NAME in ('innodb_flush_method');`

## Resolution
In most cases `O_DIRECT` is the recommended flush method.
