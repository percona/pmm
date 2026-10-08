# MySQL version

The `mysql_version` check warns if a newer release of the MySQL server's major version is available.

## Description

New releases of a major version fix bugs and security vulnerabilities. Running an older release exposes the server to issues that are already fixed.

The check reads the server variables to detect the distribution and the version. It then compares the version with the latest release of the same major version that the check knows of, for the following distributions:

- MySQL
- Percona Server for MySQL
- Percona XtraDB Cluster
- Amazon RDS for MySQL
- MariaDB

If a newer release is available, the check returns a warning that shows the current version and the latest version. The check skips Amazon Aurora and major versions that it has no release data for.

## Resolution

Upgrade to the latest release of your major version. To find the changes in each release, see the release notes of your distribution:

- [MySQL 8.0 Release Notes](https://dev.mysql.com/doc/relnotes/mysql/8.0/en/)
- [Percona Server for MySQL 8.0 release notes](https://docs.percona.com/percona-server/8.0/release-notes/release-notes_index.html)
- [Percona XtraDB Cluster 8.0 release notes](https://docs.percona.com/percona-xtradb-cluster/8.0/release-notes/release-notes_index.html)
- [MySQL on Amazon RDS versions](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MySQL.Concepts.VersionMgmt.html)
- [MariaDB release notes](https://mariadb.com/docs/release-notes)

If you run version 5.7, which reached its end of life, plan an upgrade to a supported major version. For details, see [Percona Server for MySQL 5.7 End-Of-Life](mysql_version_eol_57.md).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
