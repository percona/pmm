# Anonymous users

## Description

MySQL allows creating users without names, which can create important security issues. 
Best practices recommend removing anonymous users to secure the MySQL installation.

## Resolution

Remove any user that does not have a name in the mysql.user table.

```mysql
Delete from mysql.user where user=’’;
FLUSH PRIVILEGES;
```

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }