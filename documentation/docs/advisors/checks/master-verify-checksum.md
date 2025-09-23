# Master not verifying checksums

## Description

The source verifies binary log events by examining the checksums and stops with an error if there is a mismatch with the **master_verify_checksum** variable enabled. This variable is disabled by default. 


For more information, see [master_verify_checksum in the MuSQL documentation](https://dev.mysql.com/doc/refman/8.0/en/replication-options-binary-log.html#sysvar_master_verify_checksum)

## Resolution

Consider setting **master_verify_checksum=1** to avoid corrupt binary logs and the chance of breaking replication or silently introducing differences in the replicas.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
