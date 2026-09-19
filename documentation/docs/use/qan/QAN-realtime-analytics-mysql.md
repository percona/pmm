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
| MariaDB 11.8, 12.3, 13.0 | Supported, with the lock-detail difference noted below |

Other MariaDB releases work if they provide the required tables, but only the versions above are regularly tested.

### Service requirements

RTA requires at least one MySQL service monitored by PMM. If you haven't set this up yet, see [Connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md).

You need:

- **PMM Client 3.9.0 or later** on the monitored host. Run `pmm-admin status` to check.
- **`performance_schema` enabled** on the monitored server. RTA refuses to start without it.
- **The standard PMM monitoring user**. RTA reuses your existing MySQL exporter credentials and needs nothing beyond the grants PMM already documents:

    ```sql
    GRANT SELECT, PROCESS, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';
    ```

    The `PROCESS` privilege is what lets RTA read the lock tables. Without it, running queries still appear but lock information does not.

### Role requirements

Starting and stopping RTA sessions requires the **Admin** role. Users with other roles have View-only access to sessions that an Admin has already started. For details, see [Standard role permissions](../../admin/roles/index.md).

## Get complete data on MariaDB

MySQL and Percona Server enable everything RTA uses by default, so there is nothing to configure.

**MariaDB ships three Performance Schema switches turned off.** RTA still works without them — you get query text, user, database, command, state and elapsed time — but latency, row counts and metadata-lock detection are missing. The agent writes a warning to its log at session start naming each one.

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
| `events_statements_current` | Statement latency, lock time, rows examined/sent, temporary table counts, full-scan detection |
| `events_transactions_current` | Transaction state, duration and autocommit |
| `wait/lock/metadata/sql/mdl` | Metadata-lock detection — a statement queued behind a DDL shows as **Blocked unknown** instead of naming its blocker |

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

Click **Show/Hide columns** to add **Database**, **User**, **State**, **Command**, **Rows examined**, **Rows sent**, **Full scan** and **Program name**. Columns can be pinned left or right so they stay visible while you scroll.

!!! note alert alert-primary ""
    **Operation ID is the connection ID, not a per-statement identifier.** Consecutive statements on the same connection share it. This is inherent to how MySQL reports the process list.

### Filter the view

- **Blocked only** — show just the statements that are waiting on a lock. The count next to the toggle tells you how many there are without filtering.
- **Hide transaction control** — hide bare `BEGIN`, `COMMIT` and `ROLLBACK` statements, which dominate the view under a transactional workload and rarely tell you anything.
- **Cluster/Service** — narrow to specific services when several sessions are running.

### Control the refresh rate

Use **Auto-refresh** to adjust how often the view updates, from 1 to 5 seconds. The default is 2 seconds. Faster updates show more activity but add a small load to your database.

### Pause the stream

Click **Pause** to freeze the current view so an operation doesn't disappear on the next refresh. The agent keeps collecting in the background. Pausing also makes the **Export** button available.

## Find out what is blocking a query

This is what RTA is for. A blocked statement carries a **Blocked by** badge naming the connection holding it up. Click the row to open the **Details** tab and see the full picture:

| Field | Meaning |
| ----- | ------- |
| **Blocker's statement** | What the blocking connection is running. For the head of a chain this is often the statement that took the lock, not what it is doing now |
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

A statement shown as **Blocked unknown** means RTA could not read one of its two lock sources, so it will not claim the statement is healthy. The most common cause is the `wait/lock/metadata/sql/mdl` instrument being disabled — see [Get complete data on MariaDB](#get-complete-data-on-mariadb). Check the PMM Client log for the reason.

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

### View raw data

The **Raw data** tab shows the complete process list row for the statement, exactly as the server reported it — including fields not surfaced in **Details**. Use it when you need something RTA does not display, or to confirm what the server actually said.

## Export RTA data

You can export a snapshot of the current view to CSV. This is useful for capturing queries that are still in progress and may never appear in QAN — a long-running statement that is killed before it completes never reaches QAN, because QAN only records queries that finish.

The **Export** button is hidden while auto-refresh is active and appears once you pause.

{.power-number}

1. Go to **Query Analytics > Real-time**.
2. Click **Pause**.
3. Apply any filters or sort order you want reflected in the export.
4. Click **Export**.

The export includes all records across all pages and respects active filters and sort order.

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

- **PMM Client version**: 3.9.0 or later. Run `pmm-admin status` on the monitored host.
- **`performance_schema`**: must be enabled. Run `SELECT @@performance_schema;` — it must return `1`. This is set at startup and cannot be changed at runtime.
- **Grants**: the monitoring user needs `SELECT` and `PROCESS`. See [Service requirements](#service-requirements).
- **Admin role**: only users with the **Admin** [role](../../admin/roles/index.md) can start or stop sessions.

The session error message names the specific check that failed.

### No queries appear

RTA shows only statements that are executing at the moment of collection. Idle connections and statements that finish between collection intervals never appear. If the server is genuinely busy but the view stays empty, lower the **Auto-refresh** interval.

### Row counts and full-scan show "Unavailable"

The `events_statements_current` consumer is disabled, so the server never measured them. This is the default on MariaDB — see [Get complete data on MariaDB](#get-complete-data-on-mariadb).

RTA reports these as unavailable rather than as zero, so a statement nobody measured is never mistaken for a cheap, well-indexed one. Elapsed time still works: it falls back to the process list, which counts whole seconds.

### A blocked query doesn't name its blocker

Either the metadata-lock instrument is off (see above), or the monitoring user lacks `PROCESS`. Check the PMM Client log: the agent reports the reason once at session start rather than on every collection.

### Lock information stops updating during a large pile-up

RTA bounds the lock graph it collects, because contention grows quadratically with the number of waiters. During a very large pile-up some waiters are described and the rest are not; the agent logs a warning when this happens. The statements themselves are still listed.

## See also

- [Real-time Query Analytics for MongoDB](QAN-realtime-analytics.md)
- [Query Analytics overview](index.md)
- [Connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md)
- [Stored metrics for MySQL](mysql.md)
- [Configuration issues](../../troubleshoot/config_issues.md)
