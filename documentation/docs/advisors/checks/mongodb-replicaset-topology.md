# MongoDB replica set topology

## Description
This check returns a warning if the replica set has less than 3 members.

## Resolution
The recommended configuration for a replica set is minimum 3 data bearing members. 

This kind of topology ensures high availability and resilience in the case of  network partitioning (the “split brain” condition).

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
