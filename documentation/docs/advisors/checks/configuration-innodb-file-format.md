# MySQL InnoDB file format in use

## Description

Data file formats may not be compatible with earlier versions of InnoDB since they change to support new features. InnoDB uses named file formats to help manage this compatibility in upgrade or downgrade operations when systems run different versions of MySQL. They are the following:

* Antelope supports the REDUNDANT and COMPACT row formats. This is the original InnoDB file format.

* Barracuda supports all InnoDB row formats. This format is the newest one.

## Resolution

Barracuda is the recommended file format.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
