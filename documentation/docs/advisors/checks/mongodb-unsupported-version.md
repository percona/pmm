# Unsupported MongoDB version

The `mongodb_unsupported_version` check returns an error if the release series of MongoDB or Percona Server for MongoDB (PSMDB) is no longer supported.

## Description

Percona doesn't provide support or issue hotfixes for release series that reached their end of life (EOL). An unsupported version no longer gets bug fixes or security fixes. PSMDB follows the same EOL timeline as the MongoDB release series that it's based on.

The check reads the server version with the `buildInfo` command and returns an error for release series older than 4.4.

For the support status of each release series, see the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle). For the EOL dates of later release series, see [MongoDB end-of-life version](mongodb-eol.md).

## Resolution

Upgrade to a supported release series. Upgrade one major version at a time: for example, from 4.2 to 4.4, then to 5.0. For the recommended upgrade path, see [MongoDB versions](mongodb-version.md).

Before you upgrade a production environment, test the new version in a development or staging environment, and verify that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
