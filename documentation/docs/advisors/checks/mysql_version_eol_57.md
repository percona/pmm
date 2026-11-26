# MySQL 5.7 end of life (replaced)

In PMM 3.10 and earlier, the `mysql_version_eol_57` check warned you that MySQL, Percona Server for MySQL or Percona XtraDB Cluster 5.7 was about to reach end of life (EOL), and returned an error once that date was less than 30 days away or had passed.

In PMM 3.11.0 and later, this check is replaced by the [MySQL version check](mysql-version.md). The older check covered only 5.7, while the new check covers every series of MySQL, Percona Server for MySQL, Percona XtraDB Cluster, MariaDB and Amazon RDS for MySQL. 

The new check also warns you six months before your series reaches EOL, and gives you one finding for each database.

## Description

Percona Server for MySQL 5.7 reached EOL in October 2023, so it no longer gets regular bug or security fixes. To find the exact date for your distribution, check the [MySQL end-of-life dates](mysql-version.md#end-of-life-dates).

## Resolution

[Contact Percona](https://www.percona.com/about/contact) to get extended support for version 5.7, or to get help planning the upgrade to a supported series. You can't upgrade from 5.7 directly to 8.4: upgrade to 8.0 first, following the [8.0 upgrade instructions](https://docs.percona.com/percona-server/8.0/upgrade.html), and then to 8.4.

To find out how long Percona supports each version, check the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-services-lifecycle-policy/).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


