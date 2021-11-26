# Non-root accounts with SUPER privileges

## Description
The _Super_ privilege grants administrator privileges to a user and should be granted only to users that are supposed to act at MySQL instance level. 
The _Super_ privilege grants:
- Enables server configuration changes by modifying global system variables. For some system variables, setting the session value also requires the SUPER privilege. If a system variable is restricted and requires a special privilege to set the session value, the variable description indicates that restriction. Examples include binlog_format, sql_log_bin, and sql_log_off. See also Section 5.1.8.1, “System Variable Privileges”.
- Enables changes to global transaction characteristics (see Section 13.3.6, “SET TRANSACTION Statement”).
- Enables the account to start and stop replication.
- Enables use of the CHANGE MASTER TO statement.
- Enables binary log control by means of the PURGE BINARY LOGS and BINLOG statements.
- Enables setting the effective authorization ID when executing a view or stored program. A user with this privilege can specify any account in the DEFINER attribute of a view or stored program.
- Enables use of the CREATE SERVER, ALTER SERVER, and DROP SERVER statements.
- Enables use of the mysqladmin debug command.
- Enables reading the DES key file by the DES_ENCRYPT() function.
- Enables control over client connections not permitted to non-SUPER accounts:
	- Enables use of the KILL statement or mysqladmin kill command to kill threads belonging to other accounts. (An account can always kill its own threads.)
	- The server does not execute init_connect system variable content when SUPER clients connect.
	- The server accepts one connection from a SUPER client even if the connection limit configured by the max_connections system variable is reached.
	- Updates can be performed even when the read_only system variable is enabled. This applies to explicit table updates, and to use of account-management statements such as GRANT and REVOKE that update tables implicitly.



## Resolution
Revoke Super grants from the users that are not supposed to be MySQL instance administrators. 
Revoke super on *.* from user@'host';
