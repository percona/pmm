# MySQL binlog row image set to FULL
## Description
Having binlog_row_image set to FULL creates unnecessarily large binary logs on disk, as every column in a row will be added to its image.  Setting to MINIMAL can help reduce the disk footprint of your binary logs by storing only those columns that have actually changed.  This is specially useful if you have rows with large BLOB/TEXT columns that don't change often.  
[See](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_binlog_row_image)

## Rule
`SELECT IF(@@global.binlog_row_image='FULL', 1, 0);`


## Resolution
Please consider setting binlog_row_image=MINIMAL to reduce the footprint of your binary logs on disk.
