# MySQL innodb_file_per_table configuration is enabled

## Description

When **innodb_file_per_table=ON** is set, InnoDB uses one tablespace file per InnoDB table. This is the default since MySQL 5.6.7. 

After changing the variable ON, make sure that the tables are rebuilt using a dummy alter to pull them out from the system tablespace to their dedicated tablespace.

## Resolution

Set **innodb_file_per_table=ON** in configuration and reboot the instance.
Run dummy alters (**ALTER TABLE table_name ENGINE=InnoDB**) for every InnoDB table.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
