#!/bin/bash
# Usage: switch-config.sh [low|high]
# Switches /etc/clickhouse-server/config.xml symlink to selected config profile

set -e
CONFIG_DIR="/etc/clickhouse-server"
PROFILE="$1"


case "$PROFILE" in
  low)
    TARGET="low-memory-config.xml"
    ;;
  high)
    TARGET="high-memory-config.xml"
    ;;
  *)
    echo "Usage: $0 [low|high]"
    exit 1
    ;;
esac

if [ ! -e "$CONFIG_DIR/$TARGET" ]; then
  echo "Config profile $TARGET does not exist in $CONFIG_DIR."
  exit 2
fi

ln -sf "$TARGET" "$CONFIG_DIR/config.xml"
echo "Switched config.xml to $TARGET."
