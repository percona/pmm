#!/bin/bash
set -euo pipefail

# Usage: ./run_clickhouse_migrations.sh 21
if [ -z "$1" ]; then
	echo "Usage: $0 <start_number> (e.g., 21)"
	exit 1
fi

# Runs all ClickHouse .up.sql migrations with a number greater than <start_number>
start=${1:-0}
dir="migrations/sql"

while IFS= read -r file; do
    num=$(printf '%s\n' "$file" | cut -d'_' -f1)
    if [ "$num" -gt "$start" ]; then
        echo "Running migration: $file"
        sql=$(sed 's/ALTER TABLE metrics/ALTER TABLE pmm.metrics/g' "$dir/$file")
        docker exec pmm-server clickhouse-client --password=clickhouse --query="$sql"
    fi
done < <(find "$dir" -maxdepth 1 -type f -regextype posix-extended -regex '.*/[0-9]+_.*\.up\.sql$' -printf '%f\n' | sort)
