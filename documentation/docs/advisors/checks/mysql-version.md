# MySQL version

The `mysql_version` check warns if a newer release of the MySQL server's release series is available.

## Description

New releases of a release series fix bugs and security vulnerabilities. Running an older release exposes the server to issues that are already fixed.

The check reads the server variables to detect the distribution and the version. It then compares the version with the latest release of the same release series (major and minor version, for example 8.0, 8.4 or 10.11) that the check knows of, for these distributions:

- MySQL
- Percona Server for MySQL
- Percona XtraDB Cluster
- Amazon RDS for MySQL
- MariaDB

If a newer release is available, the check returns a warning that shows the current version and the latest version of the same series. The check never recommends a different series: a MySQL 8.0 server is compared only with the latest 8.0 release, and a MariaDB 10.11 server only with the latest 10.11 release. The warning also says that the newer release includes security fixes and links to the vendor's security page: [Oracle security alerts](https://www.oracle.com/security-alerts/) for MySQL, Percona Server for MySQL, Percona XtraDB Cluster and Amazon RDS for MySQL, and the [MariaDB security policy](https://mariadb.org/about/security-policy/) for MariaDB.

The check knows these release series:

| Distribution | Series |
|---|---|
| MySQL | 5.7, 8.0, 8.4, 9.7 |
| Percona Server for MySQL | 5.7, 8.0, 8.4, 9.7 |
| Percona XtraDB Cluster | 5.7, 8.0, 8.4, 9.7 |
| Amazon RDS for MySQL | 5.7, 8.0, 8.4 |
| MariaDB | 10.6, 10.11, 11.4, 11.8, 12.3 |

The check skips Amazon Aurora and any series that it has no release data for, such as MySQL Innovation releases and MariaDB short-term releases. Whether a series has reached end of life is reported by the [MySQL version EOL](mysql_unsupported_version_check.md) check.

## Resolution

Upgrade to the latest release of your release series. To find out what changed in each release, read the release notes of your distribution:

- [MySQL 8.0 Release Notes](https://dev.mysql.com/doc/relnotes/mysql/8.0/en/)
- [Percona Server for MySQL 8.0 release notes](https://docs.percona.com/percona-server/8.0/release-notes/release-notes_index.html)
- [Percona XtraDB Cluster 8.0 release notes](https://docs.percona.com/percona-xtradb-cluster/8.0/release-notes/release-notes_index.html)
- [MySQL on Amazon RDS versions](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MySQL.Concepts.VersionMgmt.html)
- [MariaDB release notes](https://mariadb.com/docs/release-notes)

If your series has reached its end of life, the [MySQL version EOL](mysql_unsupported_version_check.md) check reports it. Plan an upgrade to a supported release series.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
