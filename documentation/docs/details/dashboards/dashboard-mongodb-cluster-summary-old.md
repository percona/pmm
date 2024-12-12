# MongoDB Cluster Summary (OLD)

??? info "Dashboard update notice"
     A [new version of the MongoDB Sharded Cluster Summary dashboard](../../details/dashboards/dashboard-sharded-cluster-summary.md) is available. 
     This older version will be deprecated and removed from PMM in the near future. We encourage you to start using the new dashboard to benefit from its enhanced monitoring capabilities.

![!image](../../images/PMM_MongoDB_Cluster_Summary.jpg)

## Current Connections Per Shard

TCP connections (Incoming) in mongod processes.

## Total Connections

Incoming connections to mongos nodes.

## Cursors Per Shard

The Cursor is a MongoDB Collection of the document which is returned upon the find method execution.

## Mongos Cursors

The Cursor is a MongoDB Collection of the document which is returned upon the find method execution.

## Operations Per Shard

Ops/sec, classified by legacy wire protocol type (`query`, `insert`, `update`, `delete`, `getmore`).

## Total Mongos Operations

Ops/sec, classified by legacy wire protocol type (`query`, `insert`, `update`, `delete`, `getmore`).

## Change Log Events

Count, over last 10 minutes, of all types of configuration db changelog events.

## Oplog Range by Set

Timespan 'window' between oldest and newest ops in the Oplog collection.
