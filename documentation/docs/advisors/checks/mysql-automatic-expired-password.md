# MySQL automatic password expiration

The `mysql_automatic_expired_password` check warns if MySQL doesn't expire passwords automatically, or if the password lifetime is too long.

## Description

The `default_password_lifetime` system variable sets the global automatic password expiration policy: the number of days after which passwords expire. The default value, `0`, disables automatic expiration, so passwords never expire unless you expire them manually. Passwords that never change are more likely to be leaked and reused in an attack on the database.

The check reads the server variables and returns a warning if `default_password_lifetime` is one of the following:

- `0`: Automatic password expiration is inactive.
- 365 or more: The password lifetime is too long.

The warning also shows the `disconnect_on_expired_password` setting, which controls how the server treats clients whose password expired. If the variable is `ON`, the default, the server disconnects clients that can't handle expired passwords. If it is `OFF`, the server puts them in sandbox mode, where they can only reset the password.

## Resolution

Set `default_password_lifetime` to a positive number of days that is less than 365, for example:

```sql
SET PERSIST default_password_lifetime = 180;
```

`SET PERSIST` applies the value right away and keeps it after a restart. On MySQL 5.7, run `SET GLOBAL` instead, and add the setting to the `[mysqld]` section of the configuration file.

To set a different lifetime for a specific account, use `ALTER USER`:

```sql
ALTER USER 'app'@'10.0.1.15' PASSWORD EXPIRE INTERVAL 90 DAY;
```

For details, see [Password Expiration Policy](https://dev.mysql.com/doc/refman/8.4/en/password-management.html#password-expiration-policy) and [Server Handling of Expired Passwords](https://dev.mysql.com/doc/refman/8.4/en/expired-password-handling.html) in the MySQL documentation.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
