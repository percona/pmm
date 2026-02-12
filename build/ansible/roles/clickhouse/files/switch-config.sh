#!/bin/bash
# Usage: switch-config.sh [low|default]
# Switches /etc/clickhouse-server/config.xml

set -e
CONFIG_DIR="/etc/clickhouse-server"
PROFILE="$1"

if [ -z "$PROFILE" ]; then
  echo "Usage: $0 [low|default]" >&2
  exit 1
fi

case "$PROFILE" in
  low)
    CONFIG_TARGET="low-memory-config.xml"
    USERS_TARGET="low-memory-users.xml"
    ;;
  default)
    CONFIG_TARGET="default-config.xml"
    USERS_TARGET="default-users.xml"
    ;;
  *)
    echo "Usage: $0 [low|default]" >&2
    exit 1
    ;;
esac

if [ ! -e "$CONFIG_DIR/$CONFIG_TARGET" ]; then
  echo "Config profile $CONFIG_TARGET does not exist in $CONFIG_DIR." >&2
  exit 2
fi
if [ ! -e "$CONFIG_DIR/$USERS_TARGET" ]; then
  echo "Users profile $USERS_TARGET does not exist in $CONFIG_DIR." >&2
  exit 2
fi

echo "Stopping clickhouse..."
if ! supervisorctl stop clickhouse; then
  echo "Failed to stop clickhouse!" >&2
  exit 3
fi

ln -sf "$CONFIG_TARGET" "$CONFIG_DIR/config.xml"
ln -sf "$USERS_TARGET" "$CONFIG_DIR/users.xml"
echo "Switched config.xml to $CONFIG_TARGET."
echo "Switched users.xml to $USERS_TARGET."

echo "Starting clickhouse..."
if ! supervisorctl start clickhouse; then
  echo "Failed to start clickhouse!" >&2
  exit 4
fi

exit 0
