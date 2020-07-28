# Removing monitoring services with `pmm-admin remove`

Use the `pmm-admin remove` command to remove monitoring services.

## USAGE

Run this command as root or by using the `sudo` command

```sh
pmm-admin remove [OPTIONS] [SERVICE-TYPE] [SERVICE-NAME]
```

When you remove a service,
collected data remains in Metrics Monitor on PMM Server.

## SERVICES

Service type can be mysql, mongodb, postgresql or proxysql, and service
name is a monitoring service alias. To see which services are enabled,
run `pmm-admin list`.

## EXAMPLES

```sh
# Removing MySQL service named mysql-sl
pmm-admin remove mysql mysql-sl

# remove MongoDB service named mongo
pmm-admin remove mongodb mongo

# remove PostgreSQL service named postgres
pmm-admin remove postgresql postgres

# remove ProxySQL service named ubuntu-proxysql
pmm-admin remove proxysql ubuntu-proxysql
```

For more information, run `pmm-admin remove --help`.
