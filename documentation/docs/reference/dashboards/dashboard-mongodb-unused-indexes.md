# MongoDB Unused Indexes

This dashboard helps you find MongoDB indexes that have not been used since the last `mongod` service restart. These indexes are candidates for review and possible removal to reduce write overhead, disk usage, and memory pressure.

Use the filters at the top to scope the view to a specific environment, cluster, replication set, MongoDB node, or database.

**Requirements:** the `indexstats` collector must be enabled on the MongoDB exporter. Index usage is recorded only on the node that executes the query, so review all replica set members before dropping an index.

## Overview

### About Unused Indexes

Explains how the dashboard interprets index usage and what to verify before dropping an index.

### Instance Uptime

Shows how long the selected MongoDB service has been running since the last restart, in days.

Use this to understand how long index usage has been tracked. A recently restarted instance may not have had enough time for indexes to be used yet.

### Unused Indexes

Shows the number of indexes with zero accesses since restart. The default `_id_` index is excluded.

A non-zero value means there are indexes that have not been used on the selected node(s). Review the table below before taking action.

### Indexes Monitored

Shows the total number of indexes reported by the `indexstats` collector for the selected filters.

Use this together with **Unused Indexes** to understand what share of monitored indexes appear unused.

## Unused Index Candidates

### Unused Indexes by Collection

Lists indexes with zero accesses since the last `mongod` restart. Columns include cluster, database, collection, index name, and service.

This panel uses the same logic as the MongoDB Unused Indexes advisor check:

```promql
avg by (cluster, collection, database, key_name, service_name) (
  mongodb_indexstats_accesses_ops{...}
) == 0
```

Review each index with your application team before dropping it. On replica sets, an index may appear unused on one member but still be required on another.

## Index Access Activity

### Index Accesses Since Restart

Shows index access counts over time for the selected filters. Flat lines at zero indicate unused indexes.

Use this to distinguish indexes that have never been used from indexes that were used earlier but are idle now.

### Least Used Indexes

Shows the 20 indexes with the lowest access counts since restart.

Use this to find rarely used indexes that are not yet at zero but may still be candidates for review.
