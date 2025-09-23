# MongoDB Versions
## Description
This check returns the MongoDB or Percona Server for MongoDB versions currently used in your environment. It also provides information on other available minor or major versions to consider for upgrades.

## Resolution
If you are running PSMDB/MongoDB version lower than the latest minor/major patch, then we recommend upgrading to the latest version gradually. Make sure not to skip a major version release when upgrading. 

1. Upgrade to the latest minor patch of the current version. This will fix all the bugs/changes in that specific version.
2. Plan the gradual upgrade to the major version. We recommend upgrading to the latest major patch once it becomes stable.

## Recommended upgrade process
We recommend following an upgrade path similar to the one below: 
- Current minor version: 4.2.23
- Desired version: 5.0.14
- Recommended Upgrade Path: 4.2.23 => 4.4.18 => 5.0.14.
- Do NOT skip major version 4.4.x on the way to major version 5.0.x


> **Important**
> 
>Before performing any major upgrades on the production environment: 
>- Test the newer version in the lower environment (dev, staging), and verify the compatibility of that version with your application.
>- Make sure the drivers are compatible with the newer version.

If you are running **Percona Server for MongoDB (PSMDB)**, see the [Upgrade Procedure for PSMDB](https://www.percona.com/blog/upgrade-process-of-percona-server-for-mongodb-replica-set-and-shard-cluster/) for minor/major upgrades of a ReplicaSet or Sharded cluster.

If you are running **MongoDB**, see the [Upgrade Procedure for MongoDB](https://www.mongodb.com/docs/manual/tutorial/upgrade-revision/) for minor/major upgrades of a ReplicaSet or Sharded cluster.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
