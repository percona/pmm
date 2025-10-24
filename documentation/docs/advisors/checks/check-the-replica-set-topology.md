# MongoDB replica set topology

## Description
This advisor warns if the Replica Set cluster has less than three members.

## Resolution
Add at least another data-bearing member. This kind of topology ensures High Availability and resilience in case of network partitioning (“split-brain” condition).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }