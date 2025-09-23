# MySQL Replica node is not configured as READ-ONLY

## Description

To prevent accidental writes that may lead to data inconsistency, a replica node must have the READ-ONLY flag active.

The current node has a READ-ONLY value of 0 and is at high risk.

## Resolution

Set the value of READ-ONLY to 1, to prevent writes on this node.
**SET GLOBAL READ-ONLY=1;**

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
