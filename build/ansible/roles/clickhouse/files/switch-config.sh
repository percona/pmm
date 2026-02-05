#!/bin/bash
# Usage: switch-config.sh [low|default]
# Switches /etc/clickhouse/config.xml

set -e
CONFIG_DIR="./"
PROFILE="$1"

if [ -z "$PROFILE" ]; then
  echo "Usage: $0 [low|default]" >&2
  exit 1
fi

case "$PROFILE" in
  low)
    TARGET="low-memory-config.xml"
    ;;
  default)
    TARGET="default-config.xml"
    ;;
  *)
    echo "Usage: $0 [low|default]" >&2
    exit 1
    ;;
esac

if [ ! -e "$CONFIG_DIR/$TARGET" ]; then
  echo "Config profile $TARGET does not exist in $CONFIG_DIR." >&2
  exit 2
fi

echo "Stopping clickhouse..."
if ! supervisorctl stop clickhouse; then
  echo "Failed to stop clickhouse!" >&2
  exit 3
fi

ln -sf "$TARGET" "$CONFIG_DIR/config.xml"
echo "Switched config.xml to $TARGET."

echo "Starting clickhouse..."
if ! supervisorctl start clickhouse; then
  echo "Failed to start clickhouse!" >&2
  exit 4
fi

exit 0
