# MySQL General log is active

## Description
The general log contains every single query that is hitting the database. These queries are logged before execution. Having this log can mean a very serious overhead in terms of disk space and overall performance. 

## Rule
`SELECT @@global.general_log;`


## Resolution
Turn off the general_log in the configuration file and restart the instance. 

