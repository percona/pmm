# MySQL InnoDB table space has a max cap and cannot auto-extend

## Description
Some of the InnoDB Tablespace has a max size limit that means the file size can not exceed that limit. That could cause production problems if that limit is reached. More [details](https://dev.mysql.com/doc/refman/8.0/en/innodb-system-tablespace.html)

## Rule
`SELECT * from performance_schema.global_variables where VARIABLE_NAME in ('innodb_data_file_path');`


## Resolution
In most of the cases we do not recommend to have any max size limit on InnoDB Tablespaces. 

