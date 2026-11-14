##
## NOTE: All commands in this Makefile are intended to be run from the host system (outside of any container).
## Do not run these targets inside a Docker or PMM container.
##
## Usage:
## 1. Download the PMM Demo backup
##    Run:
##      make download-clickhouse-backup
##
##    The default backup is 2026-01-20.
##    All available backups can be found here:
##      https://github.com/Percona-Lab/pmm-demo-dump/releases/tag/pmm-demo
##
##    To download a different backup, set the Makefile variable BACKUP_NAME
##
## 2. Restore the PMM Demo backup
##    After downloading, the backup will be available in "dev/clickhouse-backups/"
##
##    Example:
##      dev/clickhouse-backups/20260120
##
##    Restore the backup by running:
##      make restore-clickhouse-backup
##
##    You must specify the latest migration applied in the backup.
##    For the default backup (20260120), this is migration 21 (default).
##
##    To restore a different backup, set the Makefile variable BACKUP_NAME and BACKUP_LAST_MIGRATION
##    For other backups, the latest applied migration can be found
##    in the release description:
##      https://github.com/Percona-Lab/pmm-demo-dump/releases/tag/pmm-demo
##
## 3. Run benchmarks
##    To run benchmarks for all main endpoints:
##      make bench
##
##    To run benchmarks for specific endpoints:
##      make bench-filters
##      make bench-report
##      make bench-metrics
##      make bench-example
##
##    Each benchmark runs 10 iterations and reports AVG, MIN, and MAX timings.
##
##    This allows you to compare performance before and after changes
##    and ensure there is no negative performance impact.
##
## 4. Summary
##
##    Run:
##      make download-clickhouse-backup
##      make restore-clickhouse-backup
##      make bench
.PHONY: download-clickhouse-backup restore-clickhouse-backup bench bench-filters bench-report bench-metrics bench-example
# Since the ClickHouse backup can come from a different migration state (e.g., created when the latest migration was 21),
# and the current latest migration is 22, we need to ensure that the remaining migrations are applied
# to make the system fully functional after restore.
#
# We also drop the database because, in the case of mismatched migrations, it would not be possible
# to apply them correctly. Therefore, we need to provide the backup name and the last migration
# applied to the backup.
#
# Default backup name and backup last migration.
BACKUP_NAME ?= 20260120
BACKUP_LAST_MIGRATION ?= 21
#

download-clickhouse-backup:	## Download the ClickHouse backup, unzip it and prepare the directory to restore.
	./download_clickhouse_backup.sh $(BACKUP_NAME)
#
# Default ClickHouse credentials.
CLICKHOUSE_USER ?= default
CLICKHOUSE_PASSWORD ?= clickhouse
#

restore-clickhouse-backup:	## Restore the ClickHouse backup into the pmm-server container and apply differential migrations if needed.
	docker exec pmm-server clickhouse-client --user=$(CLICKHOUSE_USER) --password=$(CLICKHOUSE_PASSWORD) --query="DROP DATABASE IF EXISTS pmm";
	docker exec pmm-server clickhouse-client --user=$(CLICKHOUSE_USER) --password=$(CLICKHOUSE_PASSWORD) --query="RESTORE DATABASE pmm FROM Disk('backup', '$(BACKUP_NAME)')";
	docker exec -u root pmm-server go run /root/go/src/github.com/percona/pmm/qan-api2/clickhouse_migrate/main.go --last-migration $(BACKUP_LAST_MIGRATION) --user CLICKHOUSE_USER=$(CLICKHOUSE_USER) --password CLICKHOUSE_PASSWORD=$(CLICKHOUSE_PASSWORD)
#
# Default time periods for benchmarking.
# NOTE: These defaults are aligned with the default demo backup (BACKUP_NAME=20260120).
# PMM_DEMO_BENCH_PERIOD_TO may be in the future relative to the current date and is
# intended for benchmarking with demo data that includes timestamps up to 2026-01-21.
# When using a different backup or real data, override these values so they match
# the actual time range of your data to avoid empty benchmark results.
PMM_DEMO_BENCH_PERIOD_FROM ?= 2025-12-27T00:00:00+01:00
PMM_DEMO_BENCH_PERIOD_TO ?= 2026-01-21T23:59:59+01:00
#

bench: bench-filters bench-report bench-metrics bench-example	## Run bench for all main endpoints.

bench-filters:	## Run bench for getFilters.
	@echo "Running Filters benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetFilters$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

bench-report:	## Run bench for getReport.
	@echo "Running Report benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetReport$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

bench-metrics:	## Run bench for getMetrics.
	@echo "Running Metrics benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetMetrics$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

bench-example:	## Run bench for getExample.
	@echo "Running Example benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetExample$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go
