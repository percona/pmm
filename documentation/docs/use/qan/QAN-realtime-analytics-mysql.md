# Real-time Query Analytics for MySQL

Use Real-time Query Analytics (RTA) to see what your MySQL, Percona Server for MySQL and MariaDB services are doing right now. RTA shows the statements that are running, including the ones waiting for a lock and the sessions holding those locks.

RTA also supports MongoDB, but shows different data for it. For MongoDB, [follow the MongoDB instructions](QAN-realtime-analytics.md) instead.

## Choose between RTA and QAN

Use RTA when you're troubleshooting a problem as it happens. For example, if a table seems frozen or your application stops responding, RTA shows you the stuck query, what it's waiting for and what's blocking it. This helps you find problems such as a runaway `UPDATE`, a schema change waiting behind an open transaction, or several sessions queued behind a single row.

[Query Analytics (QAN)](../qan/QAN-stored-metrics.md) works differently. It stores metrics for queries after they finish, so you can find slow, frequent or resource-intensive queries and spot trends over time. Because it relies on finished queries, QAN can't show you what's stuck right now.

In short, use QAN to understand query performance over time, and RTA to investigate what's happening right now.

## Prerequisites

### Check that your server is supported

RTA doesn't check the server version. When a session starts, RTA asks the server whether it provides the tables RTA needs. If something is missing, the session reports an error that says what's missing, so you don't end up with an empty view.

How much data RTA shows depends on the server and its default settings:

| Server | What RTA shows | What to do |
| ------ | -------------- | ---------- |
| MySQL 8.0 and later, Percona Server for MySQL 8.0 and later | All data, including blockers for metadata locks and transaction details | Nothing. Everything RTA uses is on by default. |
| MySQL 5.7 | All data except blockers for metadata locks and transaction details. A statement waiting for a metadata lock shows as **Blocked: unknown**. | [Turn on the missing settings](#get-complete-data-on-mysql-57-and-mariadb). |
| MariaDB | Query text, user, database, command, state and elapsed time. Latency, row counts, metadata lock detection and transaction details are missing, and lock modes are [less precise](#find-out-why-the-blocker-holds-the-lock). | [Turn on the missing settings](#get-complete-data-on-mysql-57-and-mariadb). Lock modes stay less precise. |
| Managed services, such as Amazon RDS for MySQL | Not tested. RTA can work only when `performance_schema` is enabled on the instance. | On Amazon RDS, turn on `performance_schema` in the instance's parameter group. |

### Check the service requirements

RTA needs at least one MySQL service monitored by PMM. If you haven't set one up yet, [connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md).

You need:

- **PMM Client 3.11.0 or later** on the monitored host. Run `pmm-admin status` to check.
- **`performance_schema` enabled** on the monitored server. RTA doesn't start without it.
- **The standard PMM monitoring user**. RTA reuses your existing MySQL exporter credentials and needs nothing beyond the grants PMM already documents:

    ```sql
    GRANT SELECT, PROCESS, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';
    ```

    The `PROCESS` privilege lets RTA read the lock tables. Without it, running queries still appear, but lock information doesn't.

### Get complete data on MySQL 5.7 and MariaDB

On MySQL 5.7 and MariaDB, some Performance Schema settings are off by default, and RTA shows less data until you turn them on. To see which settings are off on your server, start a session and hover over its status in the sessions list.

To turn on all the settings RTA uses:
{.power-number}

1. Run these statements on the monitored server:

    ```sql
    UPDATE performance_schema.setup_consumers
       SET ENABLED = 'YES'
     WHERE NAME IN ('events_statements_current', 'events_transactions_current');

    UPDATE performance_schema.setup_instruments
       SET ENABLED = 'YES', TIMED = 'YES'
     WHERE NAME IN ('transaction', 'wait/lock/metadata/sql/mdl');
    ```

2. To keep the settings after the server restarts, add these lines to your MySQL or MariaDB configuration file:

    ```ini
    [mysqld]
    performance_schema = ON
    performance_schema_consumer_events_statements_current = ON
    performance_schema_consumer_events_transactions_current = ON
    performance_schema_instrument = 'transaction=ON'
    performance_schema_instrument = 'wait/lock/metadata/sql/mdl=ON'
    ```

3. Restart the RTA session so the agent picks up the change.

Without these settings, you miss part of the data:

- **`events_statements_current` consumer** (off by default on MariaDB): without it, you don't see statement latency, lock time, rows examined and sent, temporary table counts, full scan detection, or the last statement of a blocker that's idle in its transaction.

    Elapsed time is also less precise, because it comes from the process list, which counts whole seconds. A statement that has run for less than a second shows as `0.000 s`.

- **`events_transactions_current` consumer and `transaction` instrument** (off by default on MySQL 5.7 and MariaDB): without them, you don't see transaction state, duration or autocommit.

    This includes the `trx_state`, `trx_latency` and `trx_autocommit` fields in the **Raw data** tab, and how long a metadata lock blocker has been idle in its transaction.

- **`wait/lock/metadata/sql/mdl` instrument** (off by default on MySQL 5.7 and MariaDB): without it, RTA can't detect metadata locks.

    A statement queued behind a DDL statement shows as **Blocked: unknown** instead of naming its blocker. RTA recognizes the wait from the statement's state, `Waiting for table metadata lock`, but can't see which connection holds the lock.

### Check your role

Starting and stopping RTA sessions requires the **Admin** role. Users with other roles can view sessions that an Admin has already started.

To check what each role can do, [review the standard role permissions](../../admin/roles/index.md).

### Review privacy implications

RTA displays statement text as MySQL reports it, which may include:

- literal values in `WHERE` clauses and `INSERT` or `UPDATE` statements
- credentials passed inside a statement
- personal or confidential data used in query conditions

Any PMM user who can open the **Query Analytics > Real-time** page can see this data. Consider your security requirements before you use RTA in production.

RTA displays what the server returns and exposes nothing beyond what `SHOW PROCESSLIST` and the Performance Schema already provide. Connection credentials and TLS material are redacted.

## Start an RTA session

You can start RTA from the PMM UI or from the command line:

=== "From the UI"

    To start monitoring a MySQL service:
    {.power-number}

    1. Go to **Query Analytics** in the sidebar.
    2. Select the **Real-time** tab. If sessions are already running, the tab opens the sessions list. Click **+ New session** there.
    3. Select a MySQL service from the **Cluster/Service** drop-down and click **Start session**:

        ![Cluster/Service drop-down with services grouped under MongoDB and MySQL](../../images/RTA_MySQL_select_service.png)

        The live query table appears and starts updating automatically. If the sessions list opens instead, click the service name to open the live query table.

    4. (Optional) To watch more services at once, for example a primary and its replicas, click **+ New session** and repeat step 3 for each one.

=== "From the command line"

    To start RTA on a specific MySQL service from the command line:
    {.power-number}

    1. Get the IDs you need:

        ```bash
        # Get your PMM Agent ID
        pmm-admin status

        # Get the MySQL service ID
        pmm-admin inventory list services
        ```

    2. Prepare the command that adds the RTA agent. Unlike the UI, this command doesn't reuse the exporter's connection settings, so pass the MySQL user and password:

        ```bash
        pmm-admin inventory add agent rta-mysql-agent <pmm-agent-id> <service-id> <mysql-user> --password='<mysql-password>'
        ```

    3. If the server requires TLS, add `--tls` and the certificate options to the command, such as `--tls-ca-file`, `--tls-cert-file` and `--tls-key-file`.

    4. Run the command. The output includes the RTA agent ID. Save it, because you need it to stop RTA later.

## Find statements that need attention

When something is slow or stuck, start with the live query table. It lists the statements running right now, with the longest-running ones at the top, so the most likely causes come first:

![RTA overview for a MySQL service, with a Blocked by label and a Truncated label](../../images/RTA_MySQL_overview.png)

To find the statements behind your problem:
{.power-number}

1. Start at the top of the table. A statement with a high **Elapsed time** has been running for a long time and is a likely cause.
2. Look for a **Blocked by** label. It means that the statement is waiting for a lock. To find what's holding it, see [Find out what is blocking a query](#find-out-what-is-blocking-a-query).
3. If you're watching several services, check **Host** to see which one the statement is running on.
4. To see which database and user the statement belongs to, click :material-view-column: **Show/Hide columns** and add the **Database** and **User** columns:

    ![Show/Hide columns menu with the Database and User columns](../../images/RTA_MySQL_columns.png)

5. When you've found the statement, [investigate it](#investigate-a-query). If it has to stop right away, [kill it](#stop-a-problematic-query).

### Narrow down the statements

On a busy server, the live query table can list many statements at once. To narrow it down to the ones behind your problem:
{.power-number}

1. If several sessions are running, select only the services you're investigating in **Cluster/Service**.
2. Turn on **Hide BEGIN/COMMIT** to clear out bare transaction statements, such as `BEGIN`, `START TRANSACTION`, `COMMIT` and `ROLLBACK`. Under a transactional workload, they fill the view and rarely tell you anything.
3. If you're investigating a lock problem, turn on **Blocked only**. Statements that might be waiting, including those labeled **Blocked: unknown**, stay visible, so you don't miss one. To see why RTA couldn't tell whether they're waiting, hover over the toggle.
4. To find specific statements, click :material-filter-variant: **Show/Hide filters** and filter by query text, host, database, user, or a **Min** and **Max** elapsed time. To match several databases or users, separate them with commas.

### Pause the view

If a statement you want to check keeps moving or disappears on the next refresh, pause the view:
{.power-number}

1. Click **Pause**. The view stops updating, but PMM Client keeps collecting in the background.
2. Check the statements you need. To see the latest data without resuming live updates, click :material-refresh: **Refresh**. To keep a copy of what you see, [export it](#export-rta-data).
3. To go back to live updates, click **Resume**:

    ![Paused RTA toolbar with the Resume, Refresh and Export buttons highlighted](../../images/RTA_MySQL_paused_toolbar.png)

### Control the refresh rate

Use **Auto-refresh** to adjust how often the view updates. The refresh rate only controls how often your browser reads the data that PMM Server already holds, so a faster refresh adds no load to your database.

PMM Client reads statements from the database at its own collect interval, every 2 seconds by default. Its queries carry a `/* pmm-agent:rta */` comment, and Query Analytics leaves them out, so they don't show up as part of your workload.

To change the collect interval:
{.power-number}

1. On the monitored host, run `pmm-admin list` and find the agent ID of the `rta_mysql_agent`.
2. Set the new interval:

    ```sh
    pmm-admin inventory change agent rta-mysql-agent <agent-id> --collect-interval=5s
    ```

## Investigate a query

The **Details** tab of a statement shows what it's doing and how much work it has done so far:

![Details tab of a running statement, with its command, state, rows examined and sent, full scan and lock time](../../images/RTA_MySQL_details.png)

To check a statement:
{.power-number}

1. Click the statement's row. The view pauses until you close the pane, so the statement stays in place, and the **Details** tab shows the statement as it was when you clicked it.
2. To check whether the statement is still running, click :material-refresh: **Refresh**. If it has finished, the pane says so. It doesn't jump to the next statement on the same connection.
3. To see newer values for the statement, close the pane and click the row again.

### Find out who sent a statement

Killing a problem statement helps only until it runs again. To fix it at the source, find the application or person behind it:
{.power-number}

1. In the statement's **Details** tab, check **Program name**, **User name** and **Client address** to find the application and host that sent it.
2. Check **Database name** to see which part of your schema it affects. If the same database keeps showing up in slow statements, look there for tables that need an index.

### Spot inefficient queries

A statement that runs for a long time is either doing too much work or waiting for something. To tell which, check its **Details** tab:

- To check for a missing index, compare **Rows examined** with **Rows sent**. If the server examines far more rows than it returns, and **Full scan** is **Yes**, the statement probably needs an index. On a running statement, these counters can lag behind, as described in [Known limitations](#known-limitations).
- To check whether the statement is waiting for table locks, look at **Lock time**, in milliseconds. A high lock time means that the statement spent its time waiting rather than working. To find out what's holding the lock, see [Find out what is blocking a query](#find-out-what-is-blocking-a-query).

### See the full text of a long statement

If a statement has a **Truncated** label, RTA shows only its beginning, because MySQL keeps only the first 1024 bytes of a statement by default. To see more of long statements:
{.power-number}

1. Turn on the `events_statements_current` consumer, as described in [Get complete data on MySQL 5.7 and MariaDB](#get-complete-data-on-mysql-57-and-mariadb). It's on by default on MySQL and Percona Server for MySQL.
2. In the MySQL configuration file, raise `performance_schema_max_sql_text_length`. RTA shows up to 64 KiB of a statement, so a higher value has no effect.
3. Restart the server, because you can't change this variable while it's running.

### View raw data

View the raw data when you need a value that the **Details** tab doesn't show, or want to confirm exactly what the server reported. The raw data is the complete process list row for the statement, as the server returned it.

To view it, click the statement's row and open the **Raw data** tab.

## Find out what is blocking a query

When a statement is waiting for a lock, it has a **Blocked by** label that names the connection holding it up. That connection may itself be waiting for another one, so follow the chain to the blocker marked **Root**: the transaction at the head of the chain, and the one to resolve.

The **Details** tab of a blocked statement shows its blocker:

![Details tab of a blocked statement, showing the blocker's statement, state, user, locked table and index, lock type and lock modes](../../images/RTA_MySQL_blocked_by.png)

To find the root blocker:
{.power-number}

1. Click the row of a statement with a **Blocked by** label to open its **Details** tab.
2. Check whether the blocker is marked **Root**:

    - If it is, the blocker isn't waiting for anything itself. To release its lock, see [Get the waiting statements running again](#get-the-waiting-statements-running-again).
    - If it isn't, the **Details** tab says that the blocker is waiting too. Open the blocker's row and repeat this step.

If a statement shows **Blocked: unknown** instead of naming its blocker, see [A statement shows Blocked: unknown](#a-statement-shows-blocked-unknown).

### Find out why the blocker holds the lock

Before you end a blocking transaction, check what it's doing, so you know whether it's safe to end it and who to contact about it. To find out:
{.power-number}

1. Click the row of the blocked statement to open its **Details** tab.
2. Check **Blocker state**. `Sleep` means that the blocker has an open transaction and is sitting idle.
3. Check **Blocker's statement** to see which statement took the lock. For an idle blocker, this is the last statement it ran. That's usually the one that took the lock, but an earlier statement in the same transaction may have taken it instead. On MariaDB, the field reads **No current statement (connection idle in transaction)** unless the `events_statements_current` consumer is on.
4. If you're investigating a deadlock, compare **Mode requested** and **Blocker holds**. On MySQL and Percona Server for MySQL, the mode tells a wait on a row apart from a wait on the gap before it, for example `X,REC_NOT_GAP` and `X,GAP`. That difference can explain a deadlock that otherwise looks impossible. MariaDB reports only `X` or `S`, so you can't tell these waits apart on MariaDB.

### Get the waiting statements running again

Waiting statements continue as soon as the root blocker releases its lock. To release it:
{.power-number}

1. In the **Details** tab, check **Lock type**.
2. Release the lock:

    - For a **Row lock (InnoDB)**, a transaction holds a row that another statement needs. Commit or roll back the blocking transaction.
    - For a **Metadata lock (MDL)**, a DDL statement is waiting for every open transaction on the table to finish, and every later statement on that table is queued behind it. That's why a whole table can seem to freeze after someone starts a schema change. End the transaction that the DDL statement is waiting for, or cancel the DDL statement.

    If you can't end the blocking transaction from the application, [stop the root blocker](#stop-a-problematic-query) instead.

## Stop a problematic query

Killing a statement rolls back its transaction, so be careful, especially with writes. To stop a statement:
{.power-number}

1. Click the row to open the **Details** tab.
2. Copy the **Operation ID**. It's the connection ID, so consecutive statements on the same connection have the same ID.
3. Connect to your MySQL instance and run:

    ```sql
    KILL <operation_id>;
    ```

To clear a lock pile-up, kill the **Root** blocker, not the statements queued behind it. To find it, see [Find out what is blocking a query](#find-out-what-is-blocking-a-query).

## Export RTA data

You can export a snapshot of the current view to CSV. This captures statements that are still running and may never appear in QAN. For example, a long-running statement that's killed before it finishes never reaches QAN, because QAN only records queries that finish.

To export the current view:
{.power-number}

1. Go to **Query Analytics > Real-time**.
2. Click **Pause**.
3. Apply any filters or sort order you want reflected in the export.
4. Click **Export**.

The export includes all records across all pages and is saved as `mysql_rta_export_{timestamp}.csv`.

The file also includes these additional columns:

- `lock_time_ms`: the lock time, in milliseconds
- `query_text_truncated`: whether the statement text was cut short
- `blocking_conn_id` and `blocking_query`: the transaction blocking the statement, if any
- `blocking_query_truncated`: whether the blocking statement's text was cut short

## Stop an RTA session

Each running session adds load to the monitored database and to PMM Server, so stop sessions you no longer need.

=== "From the UI"

    Click **All sessions** in the **Real-time** view. To stop one session, click **Stop** next to it. To stop several, select them and click **Stop selected**, or click **Stop all**.

=== "From the command line"

    To stop RTA on a service, remove its RTA agent:
    {.power-number}

    1. Find the RTA agent ID:

        ```bash
        pmm-admin inventory list agents
        ```

    2. Remove the agent:

        ```bash
        pmm-admin inventory remove agent <rta-agent-id>
        ```

    This stops RTA for that service but doesn't affect the MySQL exporter or stored QAN metrics.

## Known limitations

RTA reads what MySQL exposes while a statement runs, which has these limits:

- **Rows examined and Rows sent lag behind while a statement is running.** MySQL updates these counters as the statement progresses or only when it ends, so a running statement can report `0` rows examined even after reading millions of rows.
- **A finished statement can stay in the view for up to about 30 seconds on an idle server.** PMM Client sends no update while nothing is running, so the last snapshot stays visible until the data expires.

## Troubleshooting

### Session won't start

When a session can't start, its status shows **Error**, and the Real-time view shows the reason instead of an empty table:

![Sessions list with a session in the Error status and its reason in the tooltip](../../images/RTA_MySQL_session_error.png)

To fix it:
{.power-number}

1. In the sessions list, hover over the **Error** status to see which check failed.
2. Check the PMM Client version: run `pmm-admin status` on the monitored host. If PMM Client is older than 3.11.0, upgrade it. An older PMM Client is also why a service is missing from the **Cluster/Service** drop-down.
3. Check that `performance_schema` is on: run `SELECT @@performance_schema;`. If it returns `0`, turn it on in the server configuration and restart the server. This setting applies only at startup, so you can't change it while the server is running.
4. Check the grants of the monitoring user: it needs `SELECT` to start a session and `PROCESS` to read lock information. If either is missing, grant it. For the full list, see [Check the service requirements](#check-the-service-requirements).
5. Check your role: only users with the **Admin** [role](../../admin/roles/index.md) can start or stop sessions. If you're not an Admin, ask one to start the session.
6. Start the session again.

### No queries appear

If your server is busy but the Real-time view stays empty, PMM Client is probably collecting too rarely to catch your statements. RTA sees only statements that are running at the moment PMM Client collects, so idle connections and quick statements that start and finish between two collections never show up.

To catch them, lower the collect interval, as described in [Control the refresh rate](#control-the-refresh-rate). Changing **Auto-refresh** doesn't help, because it only sets how often your browser shows data that PMM Client has already collected.

### Row counts and full-scan show "Unavailable"

If row counts and full-scan detection show **Unavailable**, your server isn't measuring them, because the `events_statements_current` consumer is off. That's the default on MariaDB. To get these values, [turn on the missing Performance Schema settings](#get-complete-data-on-mysql-57-and-mariadb).

RTA shows **Unavailable** instead of `0` on purpose, so that you don't mistake an unmeasured statement for a cheap, well-indexed one. Elapsed time still works, but only to the whole second.

### A statement shows Blocked: unknown

If a statement shows **Blocked: unknown**, RTA can't tell whether it's waiting for a lock:

![Blocked: unknown labels on metadata-lock waits on MySQL 5.7, with the label's tooltip open and the toggle showing Blocked unknown](../../images/RTA_MySQL_blocked_unknown.png)

To find out why:
{.power-number}

1. Wait for the next refresh. If the label disappears, the connection simply moved on while RTA was reading locks, and you don't need to fix anything.
2. If the label stays, turn on the metadata lock instrument, as described in [Get complete data on MySQL 5.7 and MariaDB](#get-complete-data-on-mysql-57-and-mariadb). It's off by default on MySQL 5.7 and MariaDB.
3. If lock information is missing on every statement, check that the monitoring user has the `PROCESS` privilege, as described in [Check the service requirements](#check-the-service-requirements).

To see everything RTA can't collect, hover over the session status in the sessions list.

### Lock details are missing for some statements during a large pile-up

During a very large pile-up, you might see lock details for some waiting statements but not for others. This is expected: the number of lock relationships grows quickly with the number of waiting statements, so RTA collects only part of them. All waiting statements are still listed, and PMM Client logs a warning.

## Related topics

- [Real-time Query Analytics for MongoDB](QAN-realtime-analytics.md)
- [Real-time Query Analytics for PostgreSQL](QAN-realtime-analytics-postgresql.md)
- [Query Analytics overview](index.md)
- [Connect MySQL to PMM](../../install-pmm/install-pmm-client/connect-database/mysql/mysql.md)
- [Stored metrics for MySQL](mysql.md)
- [Configuration issues](../../troubleshoot/config_issues.md)
