# MongoDB versions

The `mongodb_version` check reports when the release series of MongoDB or Percona Server for MongoDB (PSMDB) has reached its end of life (EOL) or reaches it within six months, and otherwise when a newer minor or major version is available.

## Description

The check reads the server version with the `buildInfo` command and returns at most one finding per service. In the following table, `<product>` is **Percona Server for MongoDB** or **MongoDB**:

| Release series | Finding | Severity |
| :------------- | :------ | :------- |
| Has reached EOL | **`<product>` `<series>` reached end of life** | Error |
| Reaches EOL within the next six months (183 days), up to and including the EOL date | **`<product>` `<series>` reaches end of life on `<date>`**. If a newer minor version of the series exists, the finding also names it. | Warning |
| Supported, with a newer minor or major version available | **Newer version of `<product>` is available**. The finding names the newer version and the EOL date of the running release series. | Warning |

The check tells the two products apart by the `psmdbVersion` field, which only PSMDB returns in `buildInfo`. It compares each product with its own releases and links to that product's release notes and lifecycle policy.

A release series past its EOL gets only the EOL error. A newer minor version of that series doesn't fix the problem, so the check doesn't suggest one.

### End-of-life dates

EOL versions no longer receive bug or security fixes from the vendor. PSMDB follows the same EOL timeline as the MongoDB release series that it's based on.

The check uses these EOL dates from the **MongoDB Server** table of the [MongoDB lifecycle schedules](https://www.mongodb.com/legal/support-policy/lifecycles), which the [Percona Server for MongoDB lifecycle](https://www.percona.com/release-lifecycle-percona-software-for-mongodb/) follows:

| Release series | End of life |
| :------------- | :---------- |
| 3.0 | February 28, 2018 |
| 3.2 | September 30, 2018 |
| 3.4 | January 31, 2020 |
| 3.6 | April 30, 2021 |
| 4.0 | April 30, 2022 |
| 4.2 | April 30, 2023 |
| 4.4 | February 29, 2024 |
| 5.0 | October 31, 2024 |
| 6.0 | July 31, 2025 |
| 7.0 | August 31, 2027 |
| 8.0 | October 31, 2029 |
| 8.2 | July 31, 2026 |
| 8.3 | October 31, 2029 |
| 9.0 | October 31, 2031 |

The check doesn't report EOL for a release series that isn't in this table, such as a newer one that PMM doesn't know yet. A release series is added to the table once MongoDB publishes its EOL date. For Percona's own support dates, see the [Percona software support lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle).

The EOL finding's labels show the running version (`current`), its release series (`series`) and the EOL date (`eol_date`).

### Newer versions

For a supported release series, the check compares the version with the latest minor version of the same series, and with the latest major version, of the same product:

- PSMDB is compared with the releases in the [Percona Server for MongoDB release notes](https://docs.percona.com/percona-server-for-mongodb/8.0/release_notes/index.html). A MongoDB major version that has no PSMDB release yet isn't suggested to PSMDB users.
- MongoDB is compared with the releases in the [MongoDB release notes](https://www.mongodb.com/docs/manual/release-notes/).

The finding links to the release notes of the version it suggests. Its labels show the running version (`current`) and the suggested version (`latest`). If a newer release series is more than one major version ahead, the finding reminds you to upgrade one major version at a time. If a newer minor version of your series exists, the finding also links to the [MongoDB security alerts](https://www.mongodb.com/resources/products/alerts), where you can check which vulnerabilities affect your current version.

## Resolution

If your release series has reached EOL, or reaches it soon, upgrade to a supported release series. Upgrade one major version at a time, as the following upgrade path shows: for example, from 4.2 to 4.4, then to 5.0.

If you are running PSMDB/MongoDB version lower than the latest minor/major patch, then we recommend upgrading to the latest version gradually. Make sure not to skip a major version release when upgrading:
{.power-number}

1. Upgrade to the latest minor patch of the current version. This will fix all the bugs/changes in that specific version.
2. Plan the gradual upgrade to the major version. We recommend upgrading to the latest major patch once it becomes stable.

## Recommended upgrade process
We recommend following an upgrade path similar to the one below: 
- Current minor version: 7.0.40
- Desired version: 9.0.8
- Recommended Upgrade Path: 7.0.40 => 7.0.48 => 8.0.38 => 9.0.8.
- Do NOT skip major version 8.0.x on the way to major version 9.0.x


> **Important**
> 
>Before performing any major upgrades on the production environment: 
>- Test the newer version in the lower environment (dev, staging), and verify the compatibility of that version with your application.
>- Make sure the drivers are compatible with the newer version.

If you are running **Percona Server for MongoDB (PSMDB)**, see the [Upgrade Procedure for PSMDB](https://www.percona.com/blog/upgrade-process-of-percona-server-for-mongodb-replica-set-and-shard-cluster/) for minor/major upgrades of a ReplicaSet or Sharded cluster.

If you are running **MongoDB**, see the [Upgrade Procedure for MongoDB](https://www.mongodb.com/docs/manual/tutorial/upgrade-revision/) for minor/major upgrades of a ReplicaSet or Sharded cluster.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

