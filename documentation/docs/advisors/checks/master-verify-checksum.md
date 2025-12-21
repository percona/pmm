# Master not verifying checksums

## Description

The source verifies binary log events by examining the checksums and stops with an error if there is a mismatch with the **master_verify_checksum** variable enabled. This variable is disabled by default. 


For more information, see [master_verify_checksum in the MuSQL documentation](https://dev.mysql.com/doc/refman/8.4/en/replication-options-binary-log.html#sysvar_master_verify_checksum)

## Resolution

Consider setting **master_verify_checksum=1** to avoid corrupt binary logs and the chance of breaking replication or silently introducing differences in the replicas.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
