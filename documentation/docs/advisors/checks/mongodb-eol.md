# MongoDB end-of-life version

!!! note
    The `mongodb_EOL` check was removed in PMM 3.11.0. The [`mongodb_version`](mongodb-version.md) check now reports release series that reached or are about to reach end of life, so each service gets one version finding instead of two.

The `mongodb_EOL` check reported when the release series of MongoDB or Percona Server for MongoDB (PSMDB) has reached its end of life (EOL), or reaches it within six months.

## Description

EOL versions no longer receive bug or security fixes from the vendor. PSMDB follows the same EOL timeline as the MongoDB release series that it's based on.

The check returned an error if the release series has reached EOL, and a warning if it reaches EOL within the next six months. For the EOL dates, see [MongoDB versions](mongodb-version.md#end-of-life-dates).

## Resolution

Upgrade to a supported release series. Upgrade one major version at a time, as the [recommended upgrade path](mongodb-version.md#recommended-upgrade-process) shows: for example, from 4.2 to 4.4, then to 5.0.

Before you upgrade a production environment, test the new version in a development or staging environment, and verify that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
