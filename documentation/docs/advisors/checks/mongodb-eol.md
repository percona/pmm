# MongoDB end-of-life version

!!! note
    The `mongodb_EOL` check was removed in PMM 3.11.0. The [`mongodb_version`](mongodb-version.md) check now reports release series that reached or are about to reach end of life, so each service gets one version finding instead of two.

## Description

End-of-life (EOL) versions no longer receive bug or security fixes from the vendor. Percona Server for MongoDB (PSMDB) follows the same EOL timeline as the MongoDB release series that it's based on. For the EOL dates and the findings `mongodb_version` gives, see [MongoDB versions](mongodb-version.md#end-of-life-dates).

## Resolution

Upgrade to a supported release series. Upgrade one major version at a time, as the [recommended upgrade path](mongodb-version.md#recommended-upgrade-process) shows: for example, from 7.0 to 8.0, then to 9.0.

Before you upgrade a production environment, test the new version in a development or staging environment, and verify that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
