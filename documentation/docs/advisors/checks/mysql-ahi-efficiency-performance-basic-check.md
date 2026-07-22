# InnoDB Adaptive Hash Index (AHI) efficiency check

## Description

MySQL's InnoDB engine can automatically build a hash index in memory on frequently accessed data, this is called the Adaptive Hash Index (AHI). When it works well, the AHI lets MySQL answer queries with a fast hash lookup instead of traversing the full B-tree index, which means lower CPU usage and faster queries.

This check looks at two things:

- **how often the AHI is actually helping** (hit ratio): if most lookups still fall back to the B-tree, the AHI isn't doing much.
- **whether the AHI is causing contention** (latch wait load): under heavy concurrent workloads, multiple threads competing for the AHI's internal lock can slow things down more than the AHI speeds them up.

You'll see different alert levels depending on what's found:

- **Notice**: AHI is working normally, or it's disabled.
- **Warning**: contention is building up and starting to affect performance.
- **Major**: contention is high enough that the AHI is likely hurting more than helping. Increase AHI partitions or disable AHI altogether.

## Resolution

### If you're seeing Warning or Major alerts

Increase the number of AHI partitions. This splits the internal lock across more structures, reducing contention:

```sql
SET GLOBAL innodb_adaptive_hash_index_parts = 16;
```

To make it permanent, add it to your MySQL configuration file and restart:

```ini
[mysqld]
innodb_adaptive_hash_index_parts = 16
```

If contention is still high after increasing partitions, disabling the AHI entirely is worth testing:

```sql
SET GLOBAL innodb_adaptive_hash_index = OFF;
```

Benchmark your workload with it off to see if performance improves.

### If the AHI is disabled

Consider turning it on and watching the **InnoDB Adaptive Hash Index** panel on the **MySQL InnoDB Details** dashboard. The AHI is most useful for read-heavy workloads that repeatedly access the same index data.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
