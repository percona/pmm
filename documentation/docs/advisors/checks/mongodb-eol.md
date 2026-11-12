# MongoDB end-of-life version

The `mongodb_EOL` check reports when the release series of MongoDB or Percona Server for MongoDB (PSMDB) has reached its end of life (EOL), or reaches it within six months.

## Description

After a release series reaches EOL, it no longer gets bug fixes or security fixes. PSMDB follows the same EOL timeline as the MongoDB release series that it's based on.

The check reads the server version with the `buildInfo` command, looks up the EOL date of its release series, and returns:

- An error if the release series has reached EOL.
- A warning if the release series reaches EOL within the next six months (183 days), up to and including the EOL date itself.

The check uses these EOL dates from the [MongoDB lifecycle schedules](https://www.mongodb.com/legal/support-policy/lifecycles):

| Release series | End of life |
| :------------- | :---------- |
| 3.6 | April 30, 2021 |
| 4.0 | April 30, 2022 |
| 4.2 | April 30, 2023 |
| 4.4 | February 29, 2024 |
| 5.0 | October 31, 2024 |
| 6.0 | July 31, 2025 |
| 7.0 | August 31, 2027 |
| 8.0 | October 31, 2029 |

The check doesn't report a release series that isn't in this table, such as a newer one that PMM doesn't know yet. For Percona's own support dates, see the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle).

The finding's labels show the running version (`current`), its release series (`series`) and the EOL date (`eol_date`).

## Resolution

Upgrade to a supported release series. Upgrade one major version at a time, as the [recommended upgrade path](mongodb-version.md) shows: for example, from 4.2 to 4.4, then to 5.0.

Before you upgrade a production environment, test the new version in a development or staging environment, and verify that your drivers support it.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
