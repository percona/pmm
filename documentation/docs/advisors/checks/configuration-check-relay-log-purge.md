# MySQL relay log on the replica node is not automatically purged

## Description

Disabling the automatic purging of relay logs can have the following results:

* Relay logs can take up an unnecessary disk space

* Also enabling the **--relay-log-recovery** option risks data consistency and is therefore not crash-safe

Change this global variable dynamically with **SET GLOBAL relay_log_purge = N**.

## Resolution

Set **relay_log_purge** to 1 to enable automatic purging.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
