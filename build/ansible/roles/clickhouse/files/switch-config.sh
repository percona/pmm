#!/bin/bash
# switch-config.sh is deprecated and will be removed in a future PMM release.
# Use the PMM_CLICKHOUSE_CONFIG environment variable instead.

echo "switch-config.sh is deprecated and will be removed in a future PMM release." >&2
echo "Set PMM_CLICKHOUSE_CONFIG=default|low-memory" >&2
exit 1
