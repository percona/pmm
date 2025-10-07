# Adviosor check: MongoDB active vs available connections

## Description
This check returns a warning if the ratio between active and available connections is higher than 75%. 

This check is relevant because of the risk of running out of connections. If this happens, application won't be able to connect to MongoDB.

## Resolution
We recommend increasing the ULIMIT to accept more connections or to evaluate the current workload from applications.

An unexpected spike on the workload or not optimized queries could be the root cause of more active connections.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
