#!/bin/bash
# Wait for pmm-init to complete if it is currently running (e.g. during upgrade),
# so that Grafana does not scan the plugin directory while plugins are being copied.
for _ in {1..60}; do
    status=$(supervisorctl status pmm-init 2>/dev/null | awk '{print $2}')
    [ "$status" != "RUNNING" ] && [ "$status" != "STARTING" ] && break
    sleep 2
done

exec /usr/sbin/grafana server \
    --homepath=/usr/share/grafana \
    --config=/etc/grafana/grafana.ini \
    "$@"
