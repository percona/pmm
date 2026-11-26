# InnoDB redo log size advisor

## Description

The InnoDB redo log needs to be large enough to hold at least one hour of write traffic. When it fills up, InnoDB switches to synchronous flushing, which causes latency spikes and degrades performance.

This check raises a **warning** when write traffic exceeds the configured redo log capacity, and an **error** when the checkpoint age is close enough to the flush threshold that synchronous flushing is imminent.

## Resolution

**If you see a warning**, investigate what caused the spike and consider increasing the redo log capacity if it recurs.

**If you see an error**, increase the redo log capacity immediately:

- **MySQL 8.0.30 and newer**: adjust `innodb_redo_log_capacity`.
- **Older versions**: adjust `innodb_log_file_size` (and `innodb_log_files_in_group` if applicable).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
