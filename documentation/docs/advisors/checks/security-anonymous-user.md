# Advisor check: Anonymous users

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

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
