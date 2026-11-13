# Percona Distribution for PostgreSQL End Of Life

!!! note
    The `postgresql_eol_check` check was removed in PMM 3.11.0. The [`postgresql_version_check`](postgresql-version-check.md) check now reports major versions that reached or are about to reach end of life, so each service gets one version finding instead of two.

The `postgresql_eol_check` check reported when the major version of PostgreSQL or Percona Distribution for PostgreSQL has reached its end of life (EOL), or reaches it within six months.

## Description

EOL versions no longer receive bug or security fixes from the vendor.

The check returned an error if the major version has reached EOL, and a warning if it reaches EOL within the next six months. For the EOL dates, see [PostgreSQL version](postgresql-version-check.md#end-of-life-dates).

## Resolution

Upgrade to the latest version of Percona Distribution for PostgreSQL, following the [upgrade instructions](https://docs.percona.com/postgresql/15/major-upgrade.html).

For more information, check out the [Percona software support lifecycle information](https://www.percona.com/services/policies/percona-services-lifecycle-policy/)

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
