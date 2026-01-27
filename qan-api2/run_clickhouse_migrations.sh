#!/bin/bash
# Usage: ./run_clickhouse_migrations.sh <start_number>
# Runs all ClickHouse .up.sql migrations with a number greater than <start_number>

set -euo pipefail

start=${1:-0}
dir="migrations/sql"

for file in $(ls $dir | grep -E '^[0-9]+_.*\.up\.sql$' | sort); do
    num=$(echo $file | cut -d'_' -f1)
    if [ "$num" -gt "$start" ]; then
        echo "Running migration: $file"
        sql=$(sed 's/ALTER TABLE metrics/ALTER TABLE pmm.metrics/g' "$dir/$file")
        docker exec pmm-server clickhouse-client --password=clickhouse --query="$sql"
    fi
done
