# MySQL binlog checksum not set 

## Description
Not having binlog checksums can lead to corruption and consequent loss of data, and is considered a risky practice for production servers..\
[See](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_binlog_checksum) 


## Rule
`SELECT IF(@@global.binlog_checksum='NONE', 1, 0);`


## Resolution
Please consider setting binlog_checksum=CRC32 to improve consistency and reliability.\
`SET GLOBAL binlog_checksum=CRC32;`
 

