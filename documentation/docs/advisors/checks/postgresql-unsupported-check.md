# PostgreSQL version compatibility

!!! note
    The `postgresql_unsupported_check` check was removed in PMM 3.11.0. The [`postgresql_version_check`](postgresql-version-check.md) check now reports major versions that reached or are about to reach end of life.

The `postgresql_unsupported_check` check returned an error if the PostgreSQL major version was older than 11.

## Description

The check read the server version from the `server_version_num` setting and returned an error for major versions older than 11. These versions reached end of life (EOL), so they no longer get bug or security fixes. Later major versions have their own [end-of-life dates](postgresql-version-check.md#end-of-life-dates).

## Resolution

Upgrade to a supported major version, following the [major upgrade instructions](https://docs.percona.com/postgresql/18/major-upgrade.html).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


