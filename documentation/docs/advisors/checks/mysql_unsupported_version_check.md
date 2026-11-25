# Unsupported MySQL version (replaced)

In PMM 3.10 and earlier, the `mysql_unsupported_version_check` check returned a warning for MySQL, Percona Server for MySQL or Percona XtraDB Cluster 5.6, and an error for older versions, because they had reached end of life (EOL).

In PMM 3.11.0 and later, this check is replaced by the [MySQL version check](mysql-version.md). The older check covered only 5.6 and older, while the new check covers every series of MySQL, Percona Server for MySQL, Percona XtraDB Cluster, MariaDB and Amazon RDS for MySQL. The new check also warns you six months before your series reaches EOL, and gives you one finding for each database.

## Description

EOL versions no longer get regular bug or security fixes from their vendor. Some vendors offer fixes after EOL through extended support, such as the paid Amazon RDS Extended Support for Amazon RDS for MySQL versions past the end of standard support. 

To find out whether your distribution offers extended support, check its lifecycle policy. To find when your series reaches EOL, check the [MySQL end-of-life dates](mysql-version.md#end-of-life-dates).

## Resolution

To keep getting bug and security fixes, upgrade to a supported release series. For Percona Server for MySQL, follow the [upgrade instructions](https://docs.percona.com/percona-server/8.4/upgrade.html). 

You can't upgrade from 5.6 directly to 8.x: upgrade to the latest 5.7 first, following the [5.7 upgrade instructions](https://docs.percona.com/percona-server/5.7/upgrade.html). From 5.7, upgrade to 8.0, following the [8.0 upgrade instructions](https://docs.percona.com/percona-server/8.0/upgrade.html), and then to 8.4.

To find out how long Percona supports each version, check the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
