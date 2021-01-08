# MongoDB ReplSet

This dashboard provides information about replica sets and their members.

[TOC]

## Replication Operations

This metric provides an overview of database replication operations by type and makes it possible to analyze the load on the replica in more granular manner. These values only appear when the current host has replication enabled.

## ReplSet State

This metric shows the role of the selected member instance (PRIMARY or SECONDARY).

## ReplSet Members

This metric the number of members in the replica set.

## ReplSet Last Election

This metric how long ago the last election occurred.

## ReplSet Lag

This metric shows the current replication lag for the selected member.

## Storage Engine

This metric shows the storage engine used on the instance

## Oplog Insert Time

This metric shows how long it takes to write to the oplog. Without it the write will not be successful.

This is more useful in mixed replica sets (where instances run different storage engines).

## Oplog Recovery Window

This metric shows the time range in the oplog and the oldest backed up operation.

For example, if you take backups every 24 hours, each one should contain at least 36 hours of backed up operations, giving you 12 hours of restore window.

## Replication Lag

This metric shows the delay between an operation occurring on the primary and that same operation getting applied on the selected member

## Elections

Elections happen when a primary becomes unavailable. Look at this graph over longer periods (weeks or months) to determine patterns and correlate elections with other events.

## Member State Uptime

This metric shows how long various members were in PRIMARY and SECONDARY roles.

## Max Heartbeat Time

This metric shows the heartbeat return times sent by the current member to other members in the replica set.

Long heartbeat times can indicate network issues or that the server is too busy.

## Max Member Ping Time

This metric can show a correlation with the replication lag value.
