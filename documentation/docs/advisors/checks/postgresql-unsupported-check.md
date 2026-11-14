# PostgreSQL version compatibility

!!! note
    The `postgresql_unsupported_check` check was removed in PMM 3.11.0. The [`postgresql_version_check`](postgresql-version-check.md) check now reports major versions that reached or are about to reach end of life.

## Description
This check details the currently installed PostgreSQL version, and reports whether Percona supports it.

Percona supports all versions of Percona Distribution for PostgreSQ, including Community distribution. However, certain updates and services from Percona, such as Managed Services, only support versions 11 and higher. 

This check will advise if the currently installed version meets those requirements, and will also perform an EOL check.

## Resolution

Make sure to upgrade to a version of PostgreSQL supported by Percona.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>


