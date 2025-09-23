# InnoDB redo log size advisor

## Description

The configuration variable **innodb_log_file_size** is one of the most important settings with respect to innodb configuration.
The rule of the thumb is to keep at least one hour of traffic in those logs and let the checkpointing perform its work as smoothly as possible. If you don't do this, InnoDB will do synchronous flushing at the worst possible time.

This check does the following:

* Raises a warning if the data written is more than the configured redo log size

* Raises an error when the capacity is 80% and checkpoint switches to synchronous page flushing, which degrades performance

## Resolution

Upon Warning, review the **innodb_log_file_size** and reason that caused the spike. 
Upon Error, revise the **innodb_log_file_size**.


## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
