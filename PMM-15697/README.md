# PMM-15697 live evidence

Evidence for the percona/pmm PR from branch `PMM-15697-qan-lbac`. Not for merge; delete this branch after review.

Setup: CHAOS VM, `perconalab/pmm-server:3-dev-latest` (main @ 294e9ba8c), access control on,
MySQL 8.4 + PostgreSQL 17 (pg_stat_monitor, plan capture on), Viewer `restricted` with Access Role `{service_type="mysql"}`.
"After" is the same server with pmm-managed and qan-api2 replaced by builds of 9d6dbaf57.

- `api-*.txt`: query-details endpoints called with curl as `restricted` and as admin.
- `PMM-15697-before-*.png` / `PMM-15697-after-*.png`: QAN as `restricted`, PostgreSQL queryid selected.
- `explain-check.txt`: MySQL EXPLAIN through pmm-managed after the fix.
