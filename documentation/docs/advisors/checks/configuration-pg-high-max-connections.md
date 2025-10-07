# Advisor check: PostgreSQL max_connections set too high

## Description

PostgreSQL doesn't cope well with having many connections, even if they are idle. 

The recommended value is below 300.

Even if there are currently fewer connections than the **max_connections** value configured, the recommendation is to set a hard limit. 

Connection spikes and new applications will eventually move the number of connections higher than an acceptable threshold. 

If a significant number of connections is required, a pooling solution should be used.

This limitation comes from the fact that PostgreSQL maintains snapshots for each connection. Each new transaction will have to perform operations on the snapshots, and the more connections (and thus snapshots) there are, the higher the impact on TPS. 

## Resolution

To optimize PostgreSQL connection management and maintain performance:

- Analyze the number of connections that applications require during peak usage
- Set **max_connections** to 300 if your peak requirements are below the recommended threshold
- Consider changing how applications interact with the database if peaks exceed 300 connections
- Allocate fewer connections per application when using application-side pooling
- Implement a solution like PgBouncer when application-side pooling isn't available

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
