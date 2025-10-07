# Advisor check: MySQL InnoDB strict mode not correct

## Description

Disabling InnoDB Strict mode can lead to incompatible settings between KEY_BLOCK_SIZE, ROW_FORMAT, DATA DIRECTORY, TEMPORARY and TABLESPACE table options. 

Enabling Strict mode will force InnoDB to ensure compatibility between these create options and other settings.

In addition, enabling Strict mode checks row size when creating or altering a table. This prevents INSERT or UPDATE from failing due to the record being too large for the selected page size.


## Resolution

The **innodb_strict_mode** setting is enabled by default. Percona strongly recommends leaving this setting enabled.  
This is an online change that you can apply with:

`SET GLOBAL innodb_strict_mode=ON;`

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }