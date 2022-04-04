# PostgreSQL stale replication slot

## Description

Stale replication slots will lead to WAL file accumulation and can result in a DB server outage.

A stale replication slot is a slot that satisfies the following criteria:
- Not a temporary slot.
- Not an active slot.
- WAL distance between current WAL position and slot’s restart LSN is more than the current setting for the max_wal_size configuration option.

## Resolution

Review the output of `SELECT * FROM pg_replication_slots` and identify the slots that are inactive and have an old `restart_lsn`. Drop such slots as soon as possible. You can recreate the slot, but note that the receiving end might need to be resynchronized.
