# There are users without passwords

## Description

MySQL allows creating users with an empty string. This can lead to important security issues and should be fixed to secure your installation.

To identify user without a password:

`SELECT User, Host, authentication_string FROM mysql.user where authentication_string = '';`

Having accounts with empty passwords means that your MySQL installation is unprotected until you fix this issue.
For more information, see [Pluggable Authentication in the MySQL documentation](https://dev.mysql.com/doc/refman/8.0/en/pluggable-authentication.html).

## Resolution

Assign a password to each MySQL root account that does not have one. 
To prevent clients from connecting as anonymous users without a password, you can:

- assign a password to each anonymous account 
- or remove the accounts
- or use the auth_socket plugin if the user is a local user [read here for detailed instructions](https://dev.mysql.com/doc/mysql-secure-deployment-guide/8.0/en/secure-deployment-configure-authentication.html#:~:text=The%20auth_socket%20plugin%20checks%20whether,authentication_string%20column%20of%20the%20mysql.)

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }