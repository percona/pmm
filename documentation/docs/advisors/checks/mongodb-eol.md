# MongoDB end-of-life version (replaced)

In PMM 3.10 and earlier, the `mongodb_EOL` check returned a warning for MongoDB or Percona Server for MongoDB 4.4, which was about to reach end of life (EOL), and an error for older release series, which had reached it.

In PMM 3.11.0 and later, this check is replaced by the [MongoDB version check](mongodb-version.md). The older check covered only release series up to 4.4, while the new check covers every release series. The new check also warns you six months before your release series reaches EOL, and gives you one finding for each database instead of separate end-of-life and newer-version findings.

## Description

EOL versions no longer get bug or security fixes from their vendor. Percona Server for MongoDB follows the same EOL dates as the MongoDB release series that it's based on. 

To find when your release series reaches EOL, check the [MongoDB end-of-life dates](mongodb-version.md#end-of-life-dates).

## Resolution

To keep getting bug and security fixes, upgrade to a supported release series. Don't skip major versions: for example, to go from 7.0 to 9.0, upgrade to 8.0 first. To plan each step, follow the [recommended upgrade process](mongodb-version.md#recommended-upgrade-process).

Before you upgrade a production environment, test the new version in a development or staging environment, and check that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
