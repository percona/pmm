# MySQL expired passwords

## Description

MySQL provides a password-expiration capability, which enables database administrators to require users to reset their passwords. When the password expires, the server either disconnects the client or restricts the client to a sandbox mode. In this mode, the client can only reset the expired password.
Such scenarios impact the application users and this check helps prevent them.


## Resolution

Change the user account password with [ALTER USER](https://dev.mysql.com/doc/refman/8.0/en/alter-user.html) or [SET PASSWORD](https://dev.mysql.com/doc/refman/8.0/en/set-password.html) command. 
This action restores normal access for the session, as well as for subsequent connections that use the account. 

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
