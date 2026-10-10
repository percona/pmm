# MySQL test database

The `mysql_test_database` check reports test databases on a MySQL server.

## Description

Some MySQL installations include a `test` database that, by default, all users can access, even anonymous users. They can also include privileges that enable anyone to access databases whose names start with `test_`. A test database that remains on a production server can expose data or let any user store data on the server.

The check runs `SHOW DATABASES` and lists the databases named `test`, or whose names match the `test_%` pattern. In this pattern, `_` matches any single character, so names such as `tests` match as well. The check returns its finding with the **Info** severity.

## Resolution

If you don't use the reported databases, remove them:
{.power-number}

1. Back up any data that you want to keep.
2. Run `mysql_secure_installation` and answer `y` when it asks whether to remove the test database and access to it. Alternatively, drop each database with `DROP DATABASE`, for example: `DROP DATABASE test;`

For details, see [mysql_secure_installation](https://dev.mysql.com/doc/refman/8.4/en/mysql-secure-installation.html) in the MySQL documentation.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
