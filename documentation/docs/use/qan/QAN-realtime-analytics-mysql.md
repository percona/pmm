# Real-time Query Analytics for MySQL

While [Query Analytics (QAN) Stored metrics](../qan/QAN-stored-metrics.md) capture queries after they complete, Real-time Query Analytics (RTA) shows you what your MySQL server is executing right now — including which statements are stuck waiting on a lock, and what is holding that lock.

This is the view you want during an incident. A runaway `UPDATE`, a schema change parked behind a forgotten transaction, a pile-up of sessions queued behind a single row: QAN shows none of it until the queries finish, and a killed query never reaches QAN at all.

RTA updates every 1-5 seconds and keeps data in memory only. Use the **Pause** button to freeze the view for investigation or to export a snapshot.

## Before you start

### Supported servers

RTA works with **MySQL**, **Percona Server for MySQL**, and **MariaDB**.

Rather than checking a version number, RTA asks the server at session start whether it provides the tables it needs. If something is missing, the session reports a clear error instead of starting and showing nothing.

| Server | Status |
| ------ | ------ |
| MySQL 8.0 and later | Fully supported |
| Percona Server for MySQL 8.0 and later | Fully supported |
| MySQL 5.7 | Supported. The metadata-lock and transaction instrumentation is off by default, so a statement waiting on a metadata lock shows as **Blocked: unknown** without naming its blocker, and transaction details are empty until you enable it — see [Get complete data on MariaDB and MySQL 5.7](#get-complete-data-on-mariadb-and-mysql-57) |
| MariaDB 10.11, 11.4, 11.8, 12.3, 13.0 | Supported, with the lock-detail difference noted in [The two kinds of lock](#the-two-kinds-of-lock) |

Other MariaDB releases work if they provide the required tables, but only the versions in the table are tested.

Managed services such as Amazon RDS for MySQL have not been tested with RTA. They can work only when `performance_schema` is enabled on the instance, which you might need to turn on in the instance's parameter group.

### Service requirements

RTA requires at least one MySQL service monitored by PMM. If you haven't set this up yet, see [Connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md).

You need:

- **PMM Client 3.10.0 or later** on the monitored host. Run `pmm-admin status` to check.
- **`performance_schema` enabled** on the monitored server. RTA refuses to start without it.
- **The standard PMM monitoring user**. RTA reuses your existing MySQL exporter credentials and needs nothing beyond the grants PMM already documents:

    ```sql
    GRANT SELECT, PROCESS, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';
    ```

    The `PROCESS` privilege is what lets RTA read the lock tables. Without it, running queries still appear but lock information does not.

### Role requirements

Starting and stopping RTA sessions requires the **Admin** role. Users with other roles have View-only access to sessions that an Admin has already started. For details, see [Standard role permissions](../../admin/roles/index.md).

## Get complete data on MariaDB and MySQL 5.7

MySQL and Percona Server 8.0 and later enable everything RTA uses by default, so there is nothing to configure.

**On MySQL 5.7, the `wait/lock/metadata/sql/mdl` instrument, the `events_transactions_current` consumer and the `transaction` instrument are off by default.** Everything else works, but you lose two things:

- A statement waiting on a metadata lock shows as **Blocked: unknown** instead of naming its blocker. RTA recognises the wait from the statement's state (`Waiting for table metadata lock`), but can't see which connection holds the lock.
- Transaction details are empty: the `trx_state`, `trx_latency` and `trx_autocommit` fields in the **Raw data** tab, and how long a metadata-lock blocker has been idle in its transaction.

To enable all three, run the `setup_consumers` and `setup_instruments` statements in this section, or add the `events_transactions_current` and `performance_schema_instrument` lines from the configuration example to your MySQL configuration file and restart the server. RTA names each switch that is off at session start: hover over the session status in the sessions list to see them.

**MariaDB ships three Performance Schema switches turned off.** RTA still works without them — you get query text, user, database, command, state and elapsed time — but latency, row counts and metadata-lock detection are missing. RTA names each one at session start: hover over the session status in the sessions list to see them.

To get the complete picture, enable them:

```sql
UPDATE performance_schema.setup_consumers
   SET ENABLED = 'YES'
 WHERE NAME IN ('events_statements_current', 'events_transactions_current');

UPDATE performance_schema.setup_instruments
   SET ENABLED = 'YES', TIMED = 'YES'
 WHERE NAME IN ('transaction', 'wait/lock/metadata/sql/mdl');
```

These settings reset when the server restarts. To make them permanent, add the following to your MariaDB configuration file and restart:

```ini
[mysqld]
performance_schema = ON
performance_schema_consumer_events_statements_current = ON
performance_schema_consumer_events_transactions_current = ON
performance_schema_instrument = 'transaction=ON'
performance_schema_instrument = 'wait/lock/metadata/sql/mdl=ON'
```

Restart the RTA session afterwards so the agent picks up the change.

| Setting | What you lose without it |
| ------- | ------------------------ |
| `events_statements_current` | Statement latency, lock time, rows examined/sent, temporary table counts, full-scan detection, and the last statement of a blocker that is idle in its transaction |
| `events_transactions_current` | Transaction state, duration and autocommit |
| `wait/lock/metadata/sql/mdl` | Metadata-lock detection — a statement queued behind a DDL shows as **Blocked: unknown** instead of naming its blocker |

!!! caution alert alert-warning "Elapsed time is less precise without the statement consumer"
    With `events_statements_current` off, elapsed time comes from the process list, which counts whole seconds. A statement running for less than a second shows as `0s`.

## Start an RTA session

To start monitoring a MySQL service:
{.power-number}

1. Go to **Query Analytics** in the sidebar.
2. Select the **Real-time** tab.
3. Select a MySQL service from the **Cluster/Service** drop-down and click **Start session**.

The live query table appears and begins updating automatically.

If you have multiple services registered, click **+ New session** to run sessions simultaneously — useful for watching a primary and its replicas during the same incident.

## Read the overview

Each row is one statement currently executing. The default columns are:

| Column | Meaning |
| ------ | ------- |
| **Query text** | The statement as the server reports it, with SQL syntax highlighting |
| **Host** | The service the statement is running on |
| **Operation ID** | The connection ID — the value you pass to `KILL` |
| **Elapsed time** | How long the statement has been running |

Click **Show/Hide columns** to add **Database** and **User**. **Elapsed time** stays pinned to the right edge so it remains visible while you scroll.

State, command, rows examined, rows sent, full scan and program name are not table columns. To see them, click the row and open the **Details** tab.

!!! note alert alert-primary ""
    **Operation ID is the connection ID, not a per-statement identifier.** Consecutive statements on the same connection share it. This is inherent to how MySQL reports the process list.

### Filter the view

- **Blocked only** — show just the statements that are waiting on a lock. The count next to the toggle tells you how many there are without filtering. Statements that RTA could not judge, including those labelled **Blocked: unknown**, stay visible when the filter is on, so a statement that may be waiting is never hidden. Hover over the toggle to see why some statements could not be judged.
- **Hide BEGIN/COMMIT** — hide bare transaction-control statements (`BEGIN`, `START TRANSACTION`, `COMMIT` and `ROLLBACK`), which dominate the view under a transactional workload and rarely tell you anything.
- **Cluster/Service** — narrow to specific services when several sessions are running.

### Control the refresh rate

Use **Auto-refresh** to adjust how often the view updates, from 1 to 5 seconds. The default is 2 seconds. The refresh rate only controls how often your browser reads the data that PMM Server already holds, so a faster refresh adds no load to your database.

The database is polled by the PMM Client at its own collect interval, 2 seconds by default. To change it, run the following command on the monitored host:

```sh
pmm-admin inventory change agent rta-mysql-agent <agent-id> --collect-interval=5s
```

Run `pmm-admin list` to find the agent ID of the `rta_mysql_agent`.

RTA's own polling queries are marked with a `/* pmm-agent:rta */` comment and are left out of Query Analytics for the service, so they don't appear as part of your workload.

### Pause the stream

Click **Pause** to freeze the current view so an operation doesn't disappear on the next refresh. The agent keeps collecting in the background. Pausing also shows two icon buttons next to **Resume**:

- :material-refresh: **Refresh** reads the latest data once, without resuming live updates.
- :material-download: **Export to CSV** downloads the current view. See [Export RTA data](#export-rta-data).

Hover over an icon to see its name.

Opening a row also pauses the view, and closing the **Details** pane resumes live updates if they were running when you opened it. The pane covers the toolbar, so it has its own :material-refresh: **Refresh** button next to the previous and next arrows.

If the statement you opened finishes and you refresh, the **Details** pane keeps showing it and marks it as no longer running. It does not switch to the next statement that the same connection runs. The previous and next arrows still work: they move to the statements above and below the position the finished statement was last seen at.

## Find out what is blocking a query

This is what RTA is for. A blocked statement carries a **Blocked by** badge naming the connection holding it up. Click the row to open the **Details** tab and see the full picture:

| Field | Meaning |
| ----- | ------- |
| **Blocker's statement** | What the blocking connection is running. When the blocker is idle, this is the last statement it ran, which is often, but not always, the statement that took the lock: an earlier statement in the same transaction may have taken it. When the server keeps no text for an idle blocker, the field reads **No current statement (connection idle in transaction)**. This is the default on MariaDB, which records the last statement only with the `events_statements_current` consumer enabled |
| **Blocker state** | The blocker's current command — `Sleep` here means an open transaction sitting idle |
| **Blocker user** | Who owns the blocking connection |
| **Locked table** / **Locked index** | What is contended |
| **Lock type** | **Row lock (InnoDB)** or **Metadata lock (MDL)** |
| **Mode requested** / **Blocker holds** | The lock modes on each side |

**Root** marks a blocker that is not itself waiting on anything — the transaction at the head of the chain, and the one to resolve.

### The two kinds of lock

RTA reports two separate mechanisms, and telling them apart decides what you do next:

- **Row lock (InnoDB)** — a transaction holds a row another statement needs. Committing or rolling back the blocking transaction releases it.
- **Metadata lock (MDL)** — a DDL statement is waiting for every open transaction on a table to finish, and every later statement on that table is queued behind the DDL. This is the stall that makes an entire table appear to freeze after a schema change was started. The fix is to end the transaction the DDL is waiting on, or to cancel the DDL.

A statement waiting on a metadata lock appears in neither `SHOW ENGINE INNODB STATUS` nor the InnoDB lock tables, which is why this stall is so often misdiagnosed.

!!! note alert alert-primary ""
    **MariaDB reports lock modes less precisely.** On MySQL and Percona Server the mode distinguishes a wait on a row from a wait on the gap before it (`X,REC_NOT_GAP` against `X,GAP`) — the distinction that explains an otherwise impossible deadlock. MariaDB reports plain `X` or `S`. Everything else is the same.

### Blocked unknown

RTA does not claim a statement is healthy when it cannot tell. There are two cases, and they need different responses.

**A lock source could not be read.** If RTA cannot read one of its two lock sources, the **Blocked only** toggle says so in its tooltip, and if it can read neither, the toggle shows **Blocked unknown** and is disabled. The most common cause is the `wait/lock/metadata/sql/mdl` instrument being disabled, which is the default on MySQL 5.7 and MariaDB — see [Get complete data on MariaDB and MySQL 5.7](#get-complete-data-on-mariadb-and-mysql-57). To see what RTA cannot collect, hover over the session status in the sessions list, or check the PMM Client log.

A statement whose state says it is waiting for a lock, such as `Waiting for table metadata lock`, carries a **Blocked: unknown** label when RTA cannot read that kind of lock. Hover over the label to see why. RTA can't name the blocker, and doesn't count the statement in **Blocked only (N)**, but the filter keeps it visible. Other statements that RTA could not judge carry no label: most of them are not waiting at all.

**The connection moved on.** A single statement can carry a **Blocked: unknown** label for one refresh. RTA reads running statements and locks with two separate queries. If a connection moves on to its next statement in between and that statement waits for a lock, RTA does not attach the lock to the statement that did not request it, and labels the statement **Blocked: unknown** instead. Nothing needs fixing: the next refresh reads both again. The **Blocked only** tooltip counts these statements separately from a missing lock source.

### Stop a problematic query

!!! caution alert alert-warning "Caution"
    Killing a statement rolls back its transaction. Use carefully, especially for writes.

{.power-number}

1. Click the row to open the **Details** tab.
2. Copy the **Operation ID**.
3. Connect to your MySQL instance and run:

    ```sql
    KILL <operation_id>;
    ```

To clear a lock pile-up, kill the **Root** blocker rather than the statements queued behind it.

## Investigate a query

### Trace it to its source

In the **Details** tab, **Program name**, **User name** and **Client address** identify which application or host sent the statement. **Database name** tells you which part of your schema is affected. The same table appearing repeatedly in slow statements is a hotspot worth indexing.

### Spot inefficient queries

**Rows examined** far exceeding **Rows sent** means the server is reading much more than it returns. Combined with **Full scan**, that usually points at a missing index.

**Lock time** in the **Details** tab is how long the statement has waited for table locks, in milliseconds.

### Long statements are truncated

MySQL keeps only the beginning of a long statement, so RTA can show only that part. A statement that was cut short carries a **Truncated** label in the table, in the **Details** tab and in the blocker's statement.

How much MySQL keeps depends on server settings:

- The process list keeps 1024 bytes of a running statement.
- When the `events_statements_current` consumer is enabled, RTA reads up to `performance_schema_max_sql_text_length` bytes instead. The default is 1024 bytes. To see longer statements, raise this variable in the MySQL configuration file and restart the server, because it can't be changed at runtime.

RTA shows at most 64 KiB of any statement.

### View raw data

The **Raw data** tab shows the complete process list row for the statement, exactly as the server reported it — including fields not surfaced in **Details**. Use it when you need something RTA does not display, or to confirm what the server actually said.

## Export RTA data

You can export a snapshot of the current view to CSV. This is useful for capturing queries that are still in progress and may never appear in QAN — a long-running statement that is killed before it completes never reaches QAN, because QAN only records queries that finish.

The :material-download: **Export to CSV** icon button is hidden while auto-refresh is active and appears once you pause.

{.power-number}

1. Go to **Query Analytics > Real-time**.
2. Click **Pause**.
3. Apply any filters or sort order you want reflected in the export.
4. Click :material-download: **Export to CSV**.

The export includes all records across all pages and respects active filters and sort order. For MySQL, the `lock_time_ms` column holds the lock time in milliseconds, and `query_text_truncated` says whether the statement text was cut short. When one transaction is blocking a statement, `blocking_conn_id` and `blocking_query` name it, and `blocking_query_truncated` says whether its statement text was cut short.

## Known limitations

RTA reads what MySQL exposes while a statement runs, which has the following limits:

- **Rows examined and Rows sent show `0` while a statement is running.** MySQL fills these counters when the statement ends, so a running statement reports `0` even if it has read millions of rows.
- **A finished statement can stay in the view for up to about 30 seconds on an idle server.** The PMM Client sends no update while nothing is running, so the last snapshot stays visible until the data expires.

## Privacy considerations

!!! caution alert alert-warning "Sensitive data may be visible"

    RTA displays statement text as MySQL reports it, which may include:

    - literal values in `WHERE` clauses and `INSERT`/`UPDATE` statements
    - credentials passed inside a statement
    - personal or confidential data used in query conditions

    This data is visible to any PMM user who can access the QAN Real-time page. Consider your security requirements before enabling RTA in production.

RTA displays what the server returns and exposes nothing beyond what `SHOW PROCESSLIST` and the Performance Schema already provide. Connection credentials and TLS material are redacted.

## Troubleshooting

### Session won't start

Check each requirement:

- **PMM Client version**: 3.10.0 or later. Run `pmm-admin status` on the monitored host. A service monitored by an older PMM Client does not appear in the **Cluster/Service** drop-down, and adding the RTA agent with `pmm-admin inventory add agent rta-mysql-agent` or the inventory API is refused with an error that names the required version. If an RTA agent was added before its PMM Client reported a version, the sessions list shows the session as **Error** with the same explanation.
- **`performance_schema`**: must be enabled. Run `SELECT @@performance_schema;` — it must return `1`. This is set at startup and cannot be changed at runtime.
- **Grants**: the monitoring user needs `SELECT` and `PROCESS`. See [Service requirements](#service-requirements).
- **Admin role**: only users with the **Admin** [role](../../admin/roles/index.md) can start or stop sessions.

If a session can't start, its status shows **Error**. Hover over the status in the sessions list to see the reason, which names the check that failed. The Real-time view shows the same reason instead of an empty table.

### No queries appear

RTA shows only statements that are executing at the moment of collection. Idle connections and statements that finish between collection intervals never appear. If the server is genuinely busy but the view stays empty, lower the **Auto-refresh** interval.

### Row counts and full-scan show "Unavailable"

The `events_statements_current` consumer is disabled, so the server never measured them. This is the default on MariaDB — see [Get complete data on MariaDB and MySQL 5.7](#get-complete-data-on-mariadb-and-mysql-57).

RTA reports these as unavailable rather than as zero, so a statement nobody measured is never mistaken for a cheap, well-indexed one. Elapsed time still works: it falls back to the process list, which counts whole seconds.

### A blocked query doesn't name its blocker

Either the metadata-lock instrument is off (see above), or the monitoring user lacks `PROCESS`. Hover over the session status in the sessions list: RTA names what it cannot collect at session start.

### Lock information stops updating during a large pile-up

RTA bounds the lock graph it collects, because contention grows quadratically with the number of waiters. During a very large pile-up some waiters are described and the rest are not; the agent logs a warning when this happens. The statements themselves are still listed.

## See also

- [Real-time Query Analytics for MongoDB](QAN-realtime-analytics.md)
- [Query Analytics overview](index.md)
- [Connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md)
- [Stored metrics for MySQL](mysql.md)
- [Configuration issues](../../troubleshoot/config_issues.md)
