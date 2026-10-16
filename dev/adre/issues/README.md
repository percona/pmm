# Assistant test incidents

Scripts that make a PMM test VM look like a production system with problems, so Assistant chats and
investigations can be tried against known causes. They run on the Docker host of a CHAOS test VM that has the
pmm-qa containers (`ps_pmm_replication_8_4_1/2`, `pdpgsql_pmm_17_1`, the `mongos` + `rs1`/`rs2` sharded cluster).
Every workload runs inside the database containers. Never run them against a real system.

| Script | What it does |
|---|---|
| `alerts.sh create\|list\|delete` | Creates PMM alert rules from the built-in `pmm_*` templates in the folder *Assistant tests*; `list` also shows what is firing |
| `issues.sh` | Loads a `shop` dataset, runs steady traffic and incidents, and records each incident window with its expected evidence |

## Quick start

```sh
# on the VM, as root, from the directory holding the scripts
./alerts.sh create
./issues.sh setup            # users, data, log settings; idempotent (restarts PostgreSQL once for max_connections)
./issues.sh baseline start   # steady shop_web traffic on all three engines
./issues.sh chaos start      # one incident every 20 min: 10 min incident, 10 min calm, all scenarios in rotation
./issues.sh status           # what is running, last incidents
./issues.sh run pg_lock_waits 5   # or run one incident in the foreground
./issues.sh stop             # stop everything; ./issues.sh teardown also drops the data and restores settings
```

`chaos start <minutes> <gap> [scenario ...]` changes the timing or limits the rotation.

## Ground truth

Each finished incident appends a JSON line to `/root/adre-issues/incidents.jsonl`: scenario, UTC start and end,
target, and the evidence and cause an investigation should find. Pick a window from that file, ask the Assistant
about that time range (or investigate the alert it raised), and compare its answer with `expect`.
`./issues.sh list` prints every scenario with its expected evidence.

## What setup changes

- **MySQL primary:** `max_connections=200`, `log_error_verbosity=3`, `innodb_print_all_deadlocks=ON` (`SET PERSIST`);
  replica `log_error_verbosity=3`. Database `shop` (customers 200k, orders 2M, accounts 10) replicates to the replica.
- **PostgreSQL:** `max_connections=200` (restart), `log_lock_waits=on`, `log_temp_files=0`,
  `log_min_duration_statement=1000` (`ALTER SYSTEM`). Database `shop` with the same tables plus `order_events`
  (autovacuum disabled), `cart_items` (autovacuum logged) and `wal_churn`.
- **MongoDB:** `shop.orders` (1M documents, index on `order_no` only) and a capped `shop.audit_log`.
- Users `shop_web` (steady traffic) and `shop_app` (incidents) on all three, with the test-only password in
  `APP_PASSWORD` (default `shop_app_pw`).

No secondary indexes on `customer_id`, `note` or `status`: that is the bug most query incidents expose.

## Scenarios

Bad queries, load, and problems visible only in logs. Durations are per incident (default 10 minutes).

| Scenario | Kind | What happens | Main evidence |
|---|---|---|---|
| `mysql_full_scans` | bad query | 8 threads of unindexed `customer_id` lookups, `LIKE '%x%'` and a join over 2M rows | QAN rows examined, CPU |
| `mysql_lock_waits` | bad query | An open transaction locks every `orders` row (unindexed `UPDATE`), other updates time out | QAN lock time, 1205 errors, row lock waits |
| `mysql_deadlocks` | logs | Transfers update two `accounts` rows in random order | error log deadlock dumps, deadlock counter |
| `mysql_connection_storm` | load | 175 sleeping connections of 200, bursts refused | alert *MySQL connections near max_connections* |
| `mysql_auth_failures` | logs | Wrong password / unknown user twice a second | error log `Access denied` |
| `mysql_replica_break` | replication | Row written on the replica, then the same key on the source | alert *MySQL replication SQL thread stopped*, replica error log 1062 |
| `pg_seq_scans` | bad query | 8 clients of unindexed lookups, `LIKE`, join | QAN, `duration:` log lines |
| `pg_lock_waits` | bad query + logs | Idle-in-transaction session holds 20k row locks; updates wait and hit `statement_timeout` | log `still waiting for ShareLock`, `canceling statement due to statement timeout` |
| `pg_deadlocks` | logs | Opposite-order updates on `accounts` | log `deadlock detected` |
| `pg_connection_storm` | load | 175 sleeping connections of 200, bursts over the limit | alert *PostgreSQL connections near max_connections*, log `too many clients already` |
| `pg_auth_failures` | logs | Wrong password / unknown role | log `password authentication failed` |
| `pg_dead_tuples` | maintenance | Updates on a table with autovacuum disabled | alert *PostgreSQL high number of dead tuples* |
| `pg_vacuum_blocked` | maintenance + logs | A REPEATABLE READ transaction stays open while another table churns | dead tuples alert, autovacuum log `not yet removable` |
| `pg_rollbacks` | errors | Transactions that roll back, duplicate-key inserts | alert *PostgreSQL high transaction rollbacks*, log `duplicate key` |
| `pg_temp_files` | config | `ALTER ROLE shop_app SET work_mem='64kB'` and big sorts | log `temporary file`, temp bytes |
| `pg_checkpoint_storm` | config + logs | `max_wal_size=32MB` and bulk writes | log `checkpoints are occurring too frequently` |
| `mongo_collscans` | bad query | 6 clients of unindexed finds, counts and regex over 1M documents | QAN docs examined, mongod `Slow query` COLLSCAN |
| `mongo_big_sorts` | errors | In-memory sort over the whole collection without disk use | `Sort exceeded memory limit` errors |
| `mongo_timeouts` | errors | Unindexed finds with `maxTimeMS(30)` | `MaxTimeMSExpired` errors |
| `mongo_write_storm` | load | Unindexed `updateMany` plus bulk inserts | write tickets, dirty cache, CPU |
| `mongo_auth_failures` | logs | Wrong password / unknown user on mongos and rs101 | mongos/mongod log `Failed to authenticate` |
| `mongo_stepdown` | topology | `rs.stepDown` on the primary of the shard holding `shop` | alert *MongoDB primary changed*, election logs |
| `host_cpu_saturation` | load | MySQL, PostgreSQL and MongoDB scans at once | alert *Host CPU high* |

## Alert rules

`alerts.sh` creates 19 rules (agent, MySQL, PostgreSQL, MongoDB and host). Choices that differ from the template
defaults, and why:

- **MongoDB read/write tickets:** threshold 4 instead of 50. MongoDB 8.0 throughput probing keeps about 16
  tickets available at idle, so the default fires on an idle cluster.
- **Host CPU / memory:** filtered to node `pmm-server`. Every container node reports the host's CPU and memory, so
  without the filter each incident fires once per container.
- **Not created:** `pmm_postgresql_high_statement_timeouts` reads `postgresql_errors_total`, which no exporter on
  this VM provides, so it would never fire; bloat, backup, PBM, Valkey and ProxySQL rules have nothing to watch here.

## Auto-investigation

Auto-investigation of alerts runs only from alert messages posted to Slack (Slack integration enabled,
*Auto-investigate* on, alert notifications sent to a channel the bot reads). Without Slack, open an alert from the
alerts panel on the Assistant page and choose *Investigate*, or ask the Assistant in chat. Each investigation
costs model tokens, so with Slack set the hourly cap and minimum severity before enabling it.
