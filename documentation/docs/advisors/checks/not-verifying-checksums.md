# Checksum in the binary log not enabled

## Description

From MySQL 8.0.26, use **source_verify_checksum** rather than **master_verify_checksum**, which is deprecated from that release. 

In releases before MySQL 8.0.26, use **master_verify_checksum**.

Enabling **source_verify_checksum** causes the source to verify events read from the binary log by examining checksums, and to stop with an error in the event of a mismatch. 

The **source_verify_checksum** variable is disabled by default; in this case, the source uses the event length from the binary log to verify events, so that only complete events are read from the binary log.

## Resolution

Activate checksum by modifying the value of  **source_verify_checksum = 1**.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
