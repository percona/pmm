---
name: postgresql-maintenance
description: pmm_postgresql_high_table_bloat, pmm_postgresql_high_index_bloat, pmm_postgresql_high_number_dead_tuples, pmm_postgresql_not_autovacuumed, pmm_postgresql_not_autoanalyzed, pmm_postgresql_unused_replication_slots; vacuum lag, bloat, transaction ID wraparound risk.
---

# PostgreSQL maintenance: vacuum, bloat, wraparound, slots

## When to use
The alerts above, or: tables that keep growing, autovacuum not keeping up, a rising XID age, or `pg_wal` growing because of a replication slot.
Locks, connections, replica lag or archiving: use **postgresql-incidents**. Plan problems: use **postgresql-query-analysis**.

## Steps
1. Scope: alert labels give `datname`/`schemaname`/`relname` (table alerts), `idxname` (index bloat) or `slot_name`. `pmm_list_services(service_type=postgresql)` gives the `service_id`.
2. There is no map intent for vacuum. Use `pmm_list_dashboard_panels("postgresql-instance-summary")`: panels 1031 Dead Tuples %, 1029 Oldest Autovacuum, 1027 Transaction Wraparound. Run `pmm_metrics_snapshot` on these (add `service_id="<id>"`):
   - dead ratio: `topk(10, pg_stat_user_tables_n_dead_tup / (pg_stat_user_tables_n_live_tup + pg_stat_user_tables_n_dead_tup))`
   - vacuum recency and activity: `time() - pg_stat_user_tables_last_autovacuum`, `time() - pg_stat_user_tables_last_autoanalyze`, `increase(pg_stat_user_tables_autovacuum_count[1d])`, `pg_stat_user_tables_n_mod_since_analyze`
   - wraparound: `max by (datname) (pg_database_wraparound_age_datfrozenxid_seconds)` and `pg_database_wraparound_age_datminmxid_seconds`. The value is an XID age, not seconds. Compare it with `pg_settings_autovacuum_freeze_max_age`.
   - slots: `pg_replication_slots_active` and `pg_replication_slots_pg_wal_lsn_diff` (bytes of WAL retained, by `slot_name`; primary only)
   - settings: `pg_settings_autovacuum`, `pg_settings_autovacuum_max_workers`, `pg_settings_autovacuum_vacuum_scale_factor`, `pg_settings_autovacuum_vacuum_cost_limit`
   - bloat: `pg_bloat_table_bloat_pct`, `pg_bloat_table_real_size`, `pg_bloat_btree_bloat_pct`, `pg_bloat_btree_real_size`. These come from a custom query that the user added; the PMM client does not ship one. Check `pmm_list_metric_names(prefix="pg_bloat")`. If they are absent, the bloat alerts cannot fire: give the user the pgstattuple SQL below.
3. Logs (`pmm_clickhouse_query`, database=otel, node filter, `log.file.name ILIKE '%postg%'`): `Body ILIKE '%vacuum%' OR Body ILIKE '%wraparound%'`. This shows `canceling autovacuum task`, `skipping vacuum ... lock not available`, and `must be vacuumed within N transactions`.
4. Give the user the vacuum-blocker SQL (Remediation, first block). It decides between "autovacuum cannot clean" and "autovacuum is not running".

## Interpretation
Template thresholds:
- dead tuples: `n_dead_tup` > 10000 and at least 10% of live+dead, for 2m.
- not_autovacuumed / not_autoanalyzed: the last autovacuum or autoanalyze is older than 10 days, for 1h. Tables never processed (value 0) are excluded. A table with no writes legitimately is not vacuumed, so check `n_dead_tup` and `n_mod_since_analyze` first.
- table bloat: above 70% with a real size above 200 MB, for 1h. Index bloat: above 60% with a real size above 100 MB, for 1h.
- unused slot: `active == 0` for 1m. It fires as soon as a replica or CDC consumer disconnects.

Ranked causes of vacuum lag:
1. xmin held back by long or `idle in transaction` sessions, abandoned slots, prepared transactions, or standbys with `hot_standby_feedback`. Autovacuum runs but `n_dead_tup` stays high.
2. Thresholds too coarse for large tables (default scale factor 0.2 means 20% of the rows must change first).
3. Throttling or too few workers: autovacuum is always busy and `autovacuum_count` rises slowly. Raise the cost limit and workers.
4. Autovacuum disabled, globally (`pg_settings_autovacuum` = 0) or per table (`autovacuum_enabled=false`).
5. Lock conflicts that keep cancelling autovacuum (seen in the logs).

Wraparound:
- XID age above `autovacuum_freeze_max_age` (default 200M): aggressive anti-wraparound vacuums should already be running.
- Above about 1 billion: urgent.
- Near 2^31 (about 2.1 billion): PostgreSQL stops assigning XIDs, and writes fail.

Bloat: plain VACUUM makes the space reusable but does not shrink files. Shrinking needs pg_repack or VACUUM FULL (exclusive lock). Bloat comes back unless the vacuum cause is fixed.

Slots: an inactive slot retains all WAL from its `restart_lsn`. `pg_wal` then grows until the disk is full and the server stops. Logical slots also hold `catalog_xmin`, which bloats the catalogs and adds to wraparound pressure.

## Remediation
Vacuum blockers:
```sql
SELECT pid, datname, usename, state, age(backend_xmin) AS xmin_age, now() - xact_start AS xact_age, left(query, 100)
FROM pg_stat_activity WHERE backend_xmin IS NOT NULL ORDER BY age(backend_xmin) DESC LIMIT 10;
SELECT slot_name, slot_type, active, age(xmin) AS xmin_age, age(catalog_xmin) AS catalog_xmin_age,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS retained_wal FROM pg_replication_slots;
SELECT gid, prepared, age(transaction) AS xid_age FROM pg_prepared_xacts ORDER BY 3 DESC;
SELECT * FROM pg_stat_progress_vacuum;
```
End blockers with approval: `pg_terminate_backend(<pid>)`, or `ROLLBACK PREPARED '<gid>'`. Set `idle_in_transaction_session_timeout`.

Vacuum and analyze (online):
```sql
VACUUM (VERBOSE, ANALYZE) <schema>.<table>;   -- read "dead row versions cannot be removed yet" = xmin blocker
ANALYZE VERBOSE <schema>.<table>;
SELECT c.oid::regclass, age(c.relfrozenxid) AS xid_age FROM pg_class c WHERE c.relkind IN ('r','m','t') ORDER BY 2 DESC LIMIT 20;
VACUUM (FREEZE, VERBOSE) <schema>.<table>;     -- wraparound: oldest tables first
```
For many tables, run `vacuumdb -d <db> -j 4 --freeze --min-xid-age=1000000000` (PG12+) from any client host.

Per-table autovacuum tuning (works everywhere, needs the table owner):
```sql
ALTER TABLE <schema>.<table> SET (autovacuum_vacuum_scale_factor = 0.01, autovacuum_vacuum_threshold = 1000,
  autovacuum_analyze_scale_factor = 0.02, autovacuum_vacuum_cost_limit = 2000);
```
Global settings: `autovacuum_max_workers` (needs a restart), `autovacuum_vacuum_cost_limit`, `autovacuum_work_mem`. Change them with `ALTER SYSTEM` plus a reload (self-managed), the Operator CR (Kubernetes), or the parameter group (managed cloud).

Measure bloat exactly (scans the relation): `CREATE EXTENSION IF NOT EXISTS pgstattuple;`, then `SELECT * FROM pgstattuple_approx('<schema>.<table>');` and `SELECT * FROM pgstatindex('<schema>.<index>');`.

Index bloat (PG12+, no long lock, cannot run in a transaction block; on failure drop the leftover `*_ccnew` invalid index):
```sql
REINDEX INDEX CONCURRENTLY <schema>.<index>;
```
Table bloat:
- `pg_repack -d <db> -t <schema>.<table>` (or `--only-indexes`). Needs the pg_repack extension, a PK or unique key, and free disk about the size of the table. Self-managed, or managed clouds that offer the extension.
- `VACUUM FULL <schema>.<table>` takes an ACCESS EXCLUSIVE lock: maintenance window only.

Unused slot: first confirm that its consumer (replica, Debezium or other CDC, backup tool) is gone for good. Then:
```sql
SELECT pg_drop_replication_slot('<slot_name>');
```
Warning: if the consumer comes back, it cannot resume. A physical replica must be re-seeded (pg_basebackup), and a logical consumer loses its position and the changes in between. Patroni or Operator-managed permanent slots are recreated unless you remove them from the cluster config. On PG13+, `max_slot_wal_keep_size` caps the retained WAL (the slot is invalidated instead of filling the disk).

## Related skills
postgresql-incidents (WAL disk, archiving, locks), node-capacity-saturation (disk full).
