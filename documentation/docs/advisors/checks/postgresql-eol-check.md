# PostgreSQL end of life (replaced)

In PMM 3.10 and earlier, the `postgresql_eol_check` check returned a warning when your PostgreSQL major version had reached end of life (EOL).

In PMM 3.11.0 and later, this check is replaced by the [PostgreSQL version check](postgresql-version-check.md). 

The older check gave a separate end-of-life finding next to the newer-version finding, while the new check gives you one finding for each database. The new check also warns you six months before your major version reaches EOL, and returns an error once it has.

## Description

EOL versions no longer get bug or security fixes from their vendor. 

To find when your major version reaches EOL, check the [PostgreSQL end-of-life dates](postgresql-version-check.md#end-of-life-dates).

## Resolution

To keep getting bug and security fixes, upgrade to the latest version of Percona Distribution for PostgreSQL, following the [upgrade instructions](https://docs.percona.com/postgresql/18/major-upgrade.html).

To find out how long Percona supports each version, check the [Percona release lifecycle](https://www.percona.com/release-lifecycle-of-percona-software-for-postgresql/).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
