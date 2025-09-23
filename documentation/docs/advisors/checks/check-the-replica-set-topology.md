# MongoDB replica set topology

## Description
This advisor warns if the Replica Set cluster has less than three members.

## Resolution
Add at least another data-bearing member. This kind of topology ensures High Availability and resilience in case of network partitioning (“split-brain” condition).

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }