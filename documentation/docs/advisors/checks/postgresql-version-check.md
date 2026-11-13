# PostgreSQL version

The `postgresql_version_check` check reports when the major version of PostgreSQL or Percona Distribution for PostgreSQL has reached its end of life (EOL) or reaches it within six months, and otherwise when a newer minor or major version is available.

## Description

The check reads the server version from the `server_version_num` setting and returns at most one finding per service:

| Major version | Finding | Severity |
| :------------ | :------ | :------- |
| Has reached EOL | **PostgreSQL `<major>` reached end of life** | Error |
| Reaches EOL within the next six months (183 days), up to and including the EOL date | **PostgreSQL `<major>` reaches end of life on `<date>`**. If a newer minor version exists, the finding also names it. | Warning |
| Supported, with a newer minor or major version available | **Newer version of PostgreSQL is available**. The finding names the newer version and the EOL date of the installed major version. | Warning |

A major version past its EOL gets only the EOL error. A newer minor version of it doesn't fix the problem, so the check doesn't suggest one.

### End-of-life dates

EOL versions no longer receive bug or security fixes from the vendor.

The check uses these EOL dates from the [PostgreSQL versioning policy](https://www.postgresql.org/support/versioning/):

| Major version | End of life |
| :------------ | :---------- |
| 9.0 | October 8, 2015 |
| 9.1 | October 27, 2016 |
| 9.2 | November 9, 2017 |
| 9.3 | November 8, 2018 |
| 9.4 | February 13, 2020 |
| 9.5 | February 11, 2021 |
| 9.6 | November 11, 2021 |
| 10 | November 10, 2022 |
| 11 | November 9, 2023 |
| 12 | November 21, 2024 |
| 13 | November 13, 2025 |
| 14 | November 12, 2026 |
| 15 | November 11, 2027 |
| 16 | November 9, 2028 |
| 17 | November 8, 2029 |
| 18 | November 14, 2030 |

The check doesn't report a major version that isn't in this table, such as a newer one that PMM doesn't know yet.

The EOL finding's labels show the running version (`current`), its major version (`series`) and the EOL date (`eol_date`). The finding links to the [PostgreSQL versioning policy](https://www.postgresql.org/support/versioning/).

### Newer versions

For a supported major version, the check compares the version with the latest minor version of the same major version, and then with the latest major version. The finding's labels show the running version (`current`) and the newer version (`latest`), and the finding links to the [release notes](https://www.postgresql.org/docs/release/) of the newer version.

Newer releases include security fixes, so the newer-version finding also links to the [PostgreSQL security information page](https://www.postgresql.org/support/security/), where you can check which vulnerabilities affect your current version.

## Resolution

If your major version has reached EOL, or reaches it soon, upgrade to a supported major version. For more information, check out the [Percona software support lifecycle information](https://www.percona.com/services/policies/percona-services-lifecycle-policy/).

To upgrade PostgreSQL, follow these instructions:

- [Upgrade only the minor version](https://docs.percona.com/postgresql/15/minor-upgrade.html)
- [Upgrade major versions](https://docs.percona.com/postgresql/15/major-upgrade.html)

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

