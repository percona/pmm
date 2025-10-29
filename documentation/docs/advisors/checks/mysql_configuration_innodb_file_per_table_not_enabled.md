# MySQL innodb_file_per_table configuration is enabled

## Description

When **innodb_file_per_table=ON** is set, InnoDB uses one tablespace file per InnoDB table. This is the default since MySQL 5.6.7. 

After changing the variable ON, make sure that the tables are rebuilt using a dummy alter to pull them out from the system tablespace to their dedicated tablespace.

## Resolution

Set **innodb_file_per_table=ON** in configuration and reboot the instance.
Run dummy alters (**ALTER TABLE table_name ENGINE=InnoDB**) for every InnoDB table.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


