# Advisor check: Settings changed on a instance that requires a restart

## Description

One or more parameter setting requires a server restart/reload following a recent change.

## Resolution

Use the query below to check the parameters:

```mysql
SELECT name, setting, short_desc, reset_val FROM pg_settings WHERE pending_restart IS true;
```

Restart the PostgreSQL server to apply the new value.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
