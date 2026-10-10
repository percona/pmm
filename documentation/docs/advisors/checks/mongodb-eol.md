# MongoDB end-of-life version

The `mongodb_EOL` check warns if the release series of MongoDB or Percona Server for MongoDB (PSMDB) has reached its end of life (EOL).

## Description

After a release series reaches EOL, it no longer gets bug fixes or security fixes. PSMDB follows the same EOL timeline as the MongoDB release series that it's based on.

The check reads the server version with the `buildInfo` command and returns:

- An error for release series older than 4.4.
- A warning for release series 4.4, which reached EOL on February 29, 2024.

The check doesn't report release series 5.0 and later. Some of them also reached EOL, for example 5.0 on October 31, 2024, and 6.0 on July 31, 2025. To find the EOL date of your release series, see the [MongoDB lifecycle schedules](https://www.mongodb.com/legal/support-policy/lifecycles) and the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle).

## Resolution

Upgrade to a supported release series. Upgrade one major version at a time: for example, from 4.2 to 4.4, then to 5.0. For the recommended upgrade path, see [MongoDB versions](mongodb-version.md).

Before you upgrade a production environment, test the new version in a development or staging environment, and verify that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
