# The advisor checks for two replication specific configuration options. 


## Description

Normally, replication stops when an error occurs on the replica, which gives you the opportunity to resolve the inconsistency in the data manually. The variable  `[slave | replica]_skip_errors` causes the replication SQL thread to continue replication when a statement returns any of the errors listed in the variable value.

The configuration `[slave | replica]_exec_mode` controls the way a replication thread resolves either conflicts or errors during replication. The STRICT mode is the default value and does not suppress conflicts or errors. The IDEMPOTENT mode suppresses duplicate-key and key-not-found errors, and this mode should only be used if you are sure these errors can be ignored.
This check ensures recommended replication setup and suggests against using IDEMPOTENT mode or skipping any replication errors.



## Resolution

Implement best practice for `[slave | replica]_skip_errors` = OFF.

Implement best practice for `[slave | replica]_exec_mode` = STRICT.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
