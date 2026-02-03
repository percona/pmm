#!/bin/bash
# Usage: switch-config.sh [low|high]
# Switches /etc/clickhouse-server/config.xml symlink to selected config profile

set -e
CONFIG_DIR="/etc/clickhouse-server"
PROFILE="$1"

if [ -z "$PROFILE" ]; then
  echo "Usage: $0 [low|high]" >&2
  exit 1
fi

case "$PROFILE" in
  low)
    TARGET="low-memory-config.xml"
    ;;
  high)
    TARGET="high-memory-config.xml"
    ;;
  *)
    echo "Usage: $0 [low|high]" >&2
    exit 1
    ;;
esac

if [ ! -e "$CONFIG_DIR/$TARGET" ]; then
  echo "Config profile $TARGET does not exist in $CONFIG_DIR." >&2
  exit 2
fi

echo "Stopping clickhouse-server..."
if ! supervisorctl stop clickhouse-server; then
  echo "Failed to stop clickhouse-server!" >&2
  exit 3
fi

ln -sf "$CONFIG_DIR/$TARGET" "$CONFIG_DIR/config.xml"
echo "Switched config.xml to $TARGET."

echo "Starting clickhouse-server..."
if ! supervisorctl start clickhouse-server; then
  echo "Failed to start clickhouse-server!" >&2
  exit 4
fi

exit 0