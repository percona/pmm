# Percona Distribution for PostgreSQL End Of Life

The `postgresql_eol_check` check reports when the major version of PostgreSQL or Percona Distribution for PostgreSQL has reached its end of life (EOL), or reaches it within six months.

## Description

After a major version reaches EOL, it no longer gets bug fixes or security fixes. Percona does not provide support or issue hotfixes for end-of-life releases.

The check reads the server version from the `server_version_num` setting, looks up the EOL date of its major version, and returns:

- An error if the major version has reached EOL.
- A warning if the major version reaches EOL within the next six months (183 days), up to and including the EOL date itself.

The check uses these EOL dates, based on the [PostgreSQL versioning policy](https://www.postgresql.org/support/versioning/):

| Major version | End of life |
| :------------ | :---------- |
| 9.0 | October 9, 2015 |
| 9.1 | October 28, 2016 |
| 9.2 | November 10, 2017 |
| 9.3 | November 9, 2018 |
| 9.4 | February 10, 2020 |
| 9.5 | February 11, 2021 |
| 9.6 | November 11, 2021 |
| 10 | November 10, 2022 |
| 11 | November 9, 2023 |
| 12 | November 14, 2024 |
| 13 | November 13, 2025 |
| 14 | November 12, 2026 |
| 15 | November 11, 2027 |
| 16 | November 9, 2028 |
| 17 | November 8, 2029 |
| 18 | November 15, 2030 |

The check doesn't report a major version that isn't in this table, such as a newer one that PMM doesn't know yet.

The finding's labels show the running version (`current`), its major version (`series`) and the EOL date (`eol_date`).

## Resolution

Upgrade to the latest version of Percona Distribution for PostgreSQL, following the [upgrade instructions](https://docs.percona.com/postgresql/15/major-upgrade.html).

For more information, check out the [Percona software support lifecycle information](https://www.percona.com/services/policies/percona-services-lifecycle-policy/)

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

