# Advisor check: MySQL enforced data integrity checking is disabled

## Description

The SQL thread is not verifying binary log checksums.

Setting **slave_sql_verify_checksum=1** causes the replication SQL (applier) thread to verify data using the checksums read from the relay log. 

In the event of a mismatch, the replica stops with an error. Setting this variable takes effect for all replication channels immediately, including running channels.

## Resolution

Set the **slave_sql_verify_checksum** variable to **1**.
For MySQL 8.x and major: Set the **replica_sql_verify_checksum** variable to **1**.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
