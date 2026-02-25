# Automatic_sp_privileges configuration for MySQL

## Description

This alerts if the **automatic_sp_privileges** variable is disabled. 
When the [automatic_sp_privileges](https://dev.mysql.com/doc/refman/8.4/en/server-system-variables.html#sysvar_automatic_sp_privileges) variable is disabled, the server can no longer grant/revoke EXECUTE and ALTER ROUTINE privileges to the creator of a stored routine. 

## Resolution

Consider enabling the **automatic_sp_privileges** by setting this variable to **1**.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


