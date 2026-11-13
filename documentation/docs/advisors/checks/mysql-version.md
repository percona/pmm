# MySQL version

The `mysql_version` check reports when the release series of MySQL, Percona Server for MySQL, Percona XtraDB Cluster, MariaDB or Amazon RDS for MySQL has reached its end of life (EOL) or reaches it within six months, and otherwise when a newer release of the series is available.

## Description

The check works out the distribution and the running release series (`major.minor`, for example 8.0, 8.4 or 10.11), and returns at most one finding per service:

| Release series | Finding | Severity |
| :------------- | :------ | :------- |
| Has reached EOL | **`<Product> <series>` reached end of life** | Error |
| Reaches EOL within the next six months (183 days), up to and including the EOL date | **`<Product> <series>` reaches end of life on `<date>`**. If a newer release of the series exists, the finding also names it. | Warning |
| Supported, with a newer release of the series available | **Newer version of `<Product>` is available** | Warning |

A release series past its EOL gets only the EOL error. A newer release of that series doesn't fix the problem, so the check doesn't suggest one. For Amazon RDS for MySQL, the EOL findings say "end of standard support" instead of "end of life".

### Distributions

The check reads the server variables (`SHOW VARIABLES`) and identifies the distribution in this order:

1. **MariaDB**: `version` or `version_comment` contains `MariaDB`. A `5.5.5-` prefix on the version, which MariaDB adds for old replication clients, is ignored.
2. **Amazon Aurora**: a variable whose name starts with `aurora_` exists. Aurora isn't checked.
3. **Amazon RDS for MySQL**: `basedir` contains `rdsdbbin`, or `datadir` contains `rdsdbdata`.
4. **Percona XtraDB Cluster**: `version_comment` contains `Percona XtraDB Cluster`.
5. **Percona Server for MySQL**: `version_comment` contains `Percona Server`.
6. **MySQL** (Oracle): anything else.

### End-of-life dates

EOL versions no longer receive bug or security fixes from the vendor.

The check doesn't report EOL for a series that isn't in its tables, such as a newer one that PMM doesn't know yet. Short-lived series (MySQL Innovation releases and MariaDB short-term releases) are added to the tables only once they have reached EOL.

The EOL finding's labels show the running version (`current`), its series (`series`) and the EOL date (`eol_date`).

#### MySQL, Percona Server for MySQL and Percona XtraDB Cluster

These dates come from the [Oracle Lifetime Support Policy](https://www.oracle.com/us/support/library/lifetime-support-technology-069183.pdf) (end of Extended Support) for long-term series. An [Innovation release](https://dev.mysql.com/doc/refman/8.4/en/mysql-releases.html) is supported until the next Innovation release ships.

| Series | End of life |
| :----- | :---------- |
| 5.5 | December 31, 2018 |
| 5.6 | February 28, 2021 |
| 5.7 | October 31, 2023 |
| 8.0 | April 30, 2026 |
| 8.1 (Innovation) | October 25, 2023 |
| 8.2 (Innovation) | January 16, 2024 |
| 8.3 (Innovation) | April 30, 2024 |
| 8.4 | April 30, 2032 |
| 9.0 (Innovation) | October 15, 2024 |
| 9.1 (Innovation) | January 21, 2025 |
| 9.2 (Innovation) | April 15, 2025 |
| 9.3 (Innovation) | July 22, 2025 |
| 9.4 (Innovation) | October 21, 2025 |
| 9.5 (Innovation) | January 20, 2026 |
| 9.6 (Innovation) | April 21, 2026 |
| 9.7 | April 30, 2034 |
| 26.7 (Innovation) | October 20, 2026 |

Percona Server for MySQL and Percona XtraDB Cluster follow these dates, except where the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle) states a different one:

| Series | Percona Server for MySQL | Percona XtraDB Cluster |
| :----- | :----------------------- | :--------------------- |
| 5.5 | December 1, 2018 | September 22, 2016 |
| 5.6 | February 1, 2021 | February 1, 2021 |
| 5.7 | October 1, 2023 | October 1, 2023 |
| 8.0 | April 1, 2026 | April 1, 2026 |
| 9.7 | April 21, 2034 | Same as MySQL |

#### MariaDB

These dates come from the [MariaDB maintenance policy](https://mariadb.org/about/#maintenance-policy) (community end of life), and from [endoflife.date](https://endoflife.date/mariadb) for short-term releases:

| Series | End of life |
| :----- | :---------- |
| 5.1 | February 1, 2015 |
| 5.2 | November 10, 2015 |
| 5.3 | March 1, 2017 |
| 5.5 | April 11, 2020 |
| 10.0 | March 31, 2019 |
| 10.1 | October 17, 2020 |
| 10.2 | May 23, 2022 |
| 10.3 | May 25, 2023 |
| 10.4 | June 18, 2024 |
| 10.5 | June 24, 2025 |
| 10.6 | July 6, 2026 |
| 10.7 | February 9, 2023 |
| 10.8 | May 20, 2023 |
| 10.9 | August 22, 2023 |
| 10.10 | November 17, 2023 |
| 10.11 | February 16, 2028 |
| 11.0 | June 6, 2024 |
| 11.1 | August 21, 2024 |
| 11.2 | November 21, 2024 |
| 11.3 | May 29, 2024 |
| 11.4 | May 29, 2029 |
| 11.5 | November 21, 2024 |
| 11.6 | February 13, 2025 |
| 11.7 | May 12, 2025 |
| 11.8 | June 4, 2028 |
| 12.0 | November 18, 2025 |
| 12.1 | February 13, 2026 |
| 12.2 | May 28, 2026 |
| 12.3 | June 12, 2029 |

#### Amazon RDS for MySQL

For Amazon RDS for MySQL, the check uses the end of standard support from the [RDS for MySQL release calendar](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MySQL.Concepts.VersionMgmt.html#MySQL.Concepts.VersionMgmt.ReleaseCalendar), and the finding says "end of standard support" instead of "end of life". After that date, AWS runs the instance under [Amazon RDS Extended Support](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/extended-support.html), which is charged. RDS for MariaDB uses the MariaDB dates.

| Series | End of standard support |
| :----- | :---------------------- |
| 5.7 | February 29, 2024 |
| 8.0 | July 31, 2026 |
| 8.4 | July 31, 2029 |

### Newer releases

For a supported series, the check compares the version with the latest release of the same series that it knows of. The check never recommends a different series: a MySQL 8.4 server is compared only with the latest 8.4 release, and a MariaDB 10.11 server only with the latest 10.11 release. The finding shows the current version and the latest version (labels `current` and `latest`). It also says that the newer release includes security fixes and links to the vendor's security page: [Oracle security alerts](https://www.oracle.com/security-alerts/) for MySQL, Percona Server for MySQL, Percona XtraDB Cluster and Amazon RDS for MySQL, and the [MariaDB security policy](https://mariadb.org/about/security-policy/) for MariaDB.

The check knows the latest release of these series:

| Distribution | Series |
|---|---|
| MySQL | 5.7, 8.0, 8.4, 9.7 |
| Percona Server for MySQL | 5.7, 8.0, 8.4, 9.7 |
| Percona XtraDB Cluster | 5.7, 8.0, 8.4, 9.7 |
| Amazon RDS for MySQL | 5.7, 8.0, 8.4 |
| MariaDB | 10.6, 10.11, 11.4, 11.8, 12.3 |

The check skips Amazon Aurora.

## Resolution

If your series has reached its end of life, or reaches it soon, upgrade to a supported release series. For Percona Server for MySQL, follow the [upgrade instructions](https://docs.percona.com/percona-server/8.4/upgrade.html). You can't upgrade from 5.6 directly to 8.x: upgrade to the latest 5.7 first, following the [5.7 upgrade instructions](https://docs.percona.com/percona-server/5.7/upgrade.html). For more information, check out the [Percona software support lifecycle information](https://www.percona.com/services/policies/percona-software-support-lifecycle).

Otherwise, upgrade to the latest release of your release series. To find out what changed in each release, read the release notes of your distribution:

- [MySQL 8.0 Release Notes](https://dev.mysql.com/doc/relnotes/mysql/8.0/en/)
- [Percona Server for MySQL 8.0 release notes](https://docs.percona.com/percona-server/8.0/release-notes/release-notes_index.html)
- [Percona XtraDB Cluster 8.0 release notes](https://docs.percona.com/percona-xtradb-cluster/8.0/release-notes/release-notes_index.html)
- [MySQL on Amazon RDS versions](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MySQL.Concepts.VersionMgmt.html)
- [MariaDB release notes](https://mariadb.com/docs/release-notes)

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
