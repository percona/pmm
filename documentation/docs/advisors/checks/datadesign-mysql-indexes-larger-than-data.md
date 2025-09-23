# Index size is larger than data size

## Description

InnoDB uses clustered PRIMARY KEY indexes, which adds the primary key to the end of all secondary indexes.

This action can make the overall index size, when it contains a large primary key, larger than the actual data. Redundant indexes can also make an index larger than the raw data. 

Generally, when an index data is larger than actual data, review tables since this may be caused by one of the following:

* Poor indexing

* Redundant indexes

* Large (or composite) PK

## Resolution

Review tables for redundant indexes or large primary keys.

In some cases, this size is unavoidable, but small indexes and not over-indexing tables is important in data design.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
