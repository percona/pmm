# Unsupported MongoDB version (replaced)

In PMM 3.10 and earlier, the `mongodb_unsupported_version` check returned an error when your MongoDB or Percona Server for MongoDB release series was older than 4.4, and so no longer supported.

In PMM 3.11.0 and later, this check is replaced by the [MongoDB version check](mongodb-version.md). 

The older check covered only release series older than 4.4, while the new check knows the end-of-life (EOL) date of every release series. The new check also warns you six months before your release series reaches EOL.

## Description

A release series is unsupported once it reaches its EOL: it no longer gets bug or security fixes from its vendor. Percona Server for MongoDB follows the same EOL dates as the MongoDB release series that it's based on.

To check whether Percona still supports your release series, look it up in the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle). 

To find the EOL date of a later release series, check the [MongoDB end-of-life dates](mongodb-version.md#end-of-life-dates).

## Resolution

To keep getting bug and security fixes, upgrade to a supported release series. Don't skip major versions: for example, to go from 4.2 to 5.0, upgrade to 4.4 first, and continue until you reach a supported release series. To plan each step, follow the [recommended upgrade process](mongodb-version.md#recommended-upgrade-process).

Before you upgrade a production environment, test the new version in a development or staging environment, and check that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
