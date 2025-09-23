# MySQL InnoDB password lifetime

## Description

Passwords are set to never expire if the default password expiry time is set to **0**.
This makes passwords more vulnerable to force attacks, and increases the likelihood of the password being leaked somewhere else and being used to attack the database.

## Resolution

By default, the **default_password_lifetime** setting is set to **360** (days). Percona strongly recommends keeping a positive integer value, to force users to periodically change their passwords. 
This is an online change that you can apply with:

`SET GLOBAL default_password_lifetime=120;`


## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
