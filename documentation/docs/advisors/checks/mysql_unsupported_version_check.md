# MySQL End Of Life

!!! note
    The `mysql_unsupported_version_check` check was removed in PMM 3.11.0. The [`mysql_version`](mysql-version.md) check now reports MySQL, Percona Server for MySQL, Percona XtraDB Cluster, MariaDB and Amazon RDS for MySQL series that reached or are about to reach end of life, so each service gets one version finding instead of two.

The `mysql_unsupported_version_check` check reported when the release series of MySQL, Percona Server for MySQL, Percona XtraDB Cluster, MariaDB or Amazon RDS for MySQL has reached its end of life (EOL), or reaches it within six months.

## Description

EOL versions no longer receive bug or security fixes from the vendor.

The check returned an error if the series has reached EOL, and a warning if it reaches EOL within the next six months. For Amazon RDS for MySQL, it reported the end of standard support instead. For the dates of each distribution, see [MySQL version](mysql-version.md#end-of-life-dates).

## Resolution

Upgrade to a supported release series. For Percona Server for MySQL, follow the [upgrade instructions](https://docs.percona.com/percona-server/8.4/upgrade.html). You can't upgrade from 5.6 directly to 8.x: upgrade to the latest 5.7 first, following the [5.7 upgrade instructions](https://docs.percona.com/percona-server/5.7/upgrade.html).

For more information, check out the [Percona software support lifecycle information](https://www.percona.com/services/policies/percona-software-support-lifecycle).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
