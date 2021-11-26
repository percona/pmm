# Anonymous user(s)

## Description
MySQL allows the creation of users without names, this is very insecure, it is best practice to remove anonymous users to secure the MySQL installation. 

## Rule
`Select user,host from mysql.user where user = ''`


## Resolution
Remove any user that does not have a name in the mysql.user table. 
```
Delete from mysql.user where user=’’;
FLUSH PRIVILEGES;
```
