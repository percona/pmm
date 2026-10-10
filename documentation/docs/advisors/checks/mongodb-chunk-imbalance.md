# MongoDB balancer is disabled

The `mongodb_balancer` check warns if the balancer is disabled in a MongoDB sharded cluster.

## Description

The balancer is a background process of a sharded cluster that migrates chunks between shards, so that each shard holds an even share of the data. The balancer is enabled by default.

While the balancer is disabled, chunks don't move between shards. New data accumulates on some shards, which then serve more reads and writes than the others and run out of disk space sooner. Queries that target these shards slow down.

The check reads the balancer mode that PMM collects from the `mongos` routers with the `balancerStatus` command. If the mode isn't `full`, the balancer is disabled, and the check returns a warning that lists the affected clusters. An enabled balancer doesn't trigger the check, whether or not it is migrating chunks at the moment.

You can disable the balancer on purpose for a short time, for example during maintenance or a manual backup with `mongodump`. Enable it again when you finish.

## Resolution

To enable the balancer:
{.power-number}

1. Connect to any `mongos` of the cluster with `mongosh`.
2. Check the balancer state. The method returns `false` if the balancer is disabled:

    ```javascript
    sh.getBalancerState()
    ```

3. If no maintenance or backup requires the balancer to stay disabled, enable it:

    ```javascript
    sh.startBalancer()
    ```

    To enable the balancer from a driver, run the `balancerStart` command against the `admin` database instead: `db.adminCommand( { balancerStart: 1 } )`.

If chunk migrations affect performance during busy hours, schedule a balancing window instead of disabling the balancer. For details, see [Manage Sharded Cluster Balancer](https://www.mongodb.com/docs/manual/tutorial/manage-sharded-cluster-balancer/) in the MongoDB documentation.

### Data stays unevenly distributed

If the data stays unevenly distributed while the balancer is enabled, check the following common causes:

- **Shard key**: A shard key with low cardinality, or one that increases monotonically, such as a timestamp, sends most writes to the same shard. Choose a shard key with high cardinality that most of your queries use. Starting with MongoDB 5.0, you can reshard a collection to change its shard key. For details, see [Reshard a Collection](https://www.mongodb.com/docs/manual/core/sharding-reshard-a-collection/) in the MongoDB documentation.
- **Jumbo chunks**: The balancer can't move a chunk that grew beyond the maximum chunk size and can't be split. To find and clear jumbo chunks, see [Finding Undetected Jumbo Chunks in MongoDB](https://www.percona.com/blog/finding-undetected-jumbo-chunks-in-mongodb/).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
