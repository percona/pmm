# Automatic_sp_privileges configuration for MySQL


## Description

This alerts if the **automatic_sp_privileges** variable is disabled. 
When the [automatic_sp_privileges](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_automatic_sp_privileges) variable is disabled, the server can no longer grant/revoke EXECUTE and ALTER ROUTINE privileges to the creator of a stored routine. 

## Resolution

Consider enabling the **automatic_sp_privileges** by setting this variable to **1**.


## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
