# There are users without passwords

## Description
MySQL allows user creation with an empty string. This is very insecure and should be correct to secure your installation.
To identify the presence of any user without password:
`SELECT User, Host, authentication_string FROM mysql.user where authentication_string = '';`
The presence of accounts with empty passwords means that your MySQL installation is unprotected until you do something about it.
For details about [authentication](https://dev.mysql.com/doc/refman/8.0/en/pluggable-authentication.html)   


## Resolution
Assign a password to each MySQL root account that does not have one.
To prevent clients from connecting as anonymous users without a password, either assign a password to each anonymous account or remove the accounts.
 