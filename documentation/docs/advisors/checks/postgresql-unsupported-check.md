# Unsupported PostgreSQL version (replaced)

In PMM 3.10 and earlier, the `postgresql_unsupported_check` check returned an error when your PostgreSQL major version was older than 11, because those versions had reached end of life (EOL).

In PMM 3.11.0 and later, this check is replaced by the [PostgreSQL version check](postgresql-version-check.md). The older check covered only major versions older than 11, while the new check knows the EOL date of every major version. The new check also warns you six months before your major version reaches EOL.

## Description

EOL versions no longer get bug or security fixes from their vendor. To find the EOL date of a later major version, check the [PostgreSQL end-of-life dates](postgresql-version-check.md#end-of-life-dates).

## Resolution

To keep getting bug and security fixes, upgrade to a supported major version, following the [major upgrade instructions](https://docs.percona.com/postgresql/18/major-upgrade.html).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


