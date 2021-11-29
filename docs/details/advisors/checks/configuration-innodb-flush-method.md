# MySQL InnoDB file format in use

## Description
Before MySQL version 8 InnoDB had two file formats Antelope and Barracuda. Barracuda is the preferred file format.
From MySQL 8 The following InnoDB file format variables were removed:
- Innodb_file_format
- Innodb_file_format_check
- Innodb_file_format_max
- innodb_large_prefix

File format variables were necessary for creating tables compatible with earlier versions of InnoDB in MySQL 5.1. Now that MySQL 5.1 has reached the end of its product lifecycle, these options are no longer required.


## Rule
`SELECT * from performance_schema.global_variables where VARIABLE_NAME in ('innodb_file_format','innodb_file_format_max','innodb_flush_method','innodb_data_file_path');`


## Resolution
Barracuda is the recommended file format, support for Antelope is removed from MySQL 8.


