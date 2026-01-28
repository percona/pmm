#!/bin/bash
set -e

# Usage: ./download_clickhouse_backup.sh 20260120
if [ -z "$1" ]; then
	echo "Usage: $0 <date_postfix> (e.g., 20260120)"
	exit 1
fi

POSTFIX="$1"
BACKUP_PARENT="../dev/clickhouse-backups"
TARGET_DIR="$BACKUP_PARENT/$POSTFIX"
ZIP_FILE="pmm_backup_${POSTFIX}.zip"
URL="https://github.com/Percona-Lab/pmm-demo-dump/releases/download/pmm-demo/pmm_backup_${POSTFIX}.zip"

# Download
curl -fSsL -o "$ZIP_FILE" "$URL"

# Ensure parent directory exists and is accessible
mkdir -p "$BACKUP_PARENT"
chmod 755 "$BACKUP_PARENT"

# Prepare target directory
mkdir -p "$TARGET_DIR"
chmod 755 "$TARGET_DIR"

# Extract
unzip -o "$ZIP_FILE" -d "$TARGET_DIR"
rm "$ZIP_FILE"

# Fix permissions recursively on target
chmod -R u+rwX,go+rX "$TARGET_DIR"

echo "Backup downloaded and extracted to $TARGET_DIR"
