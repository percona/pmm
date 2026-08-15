#!/bin/bash
#
# The container's HEALTHCHECK: the readiness probe the inline command used to run, plus the
# SEP provisioning marker grafana-sep writes and entrypoint.sh clears each start.
#
# No errexit, and every failure returns 1 explicitly: Docker reserves exit code 2.

declare READYZ_URL="http://127.0.0.1:8080/v1/server/readyz"
declare SEP_MARKER="/srv/.sep_provisioned"

is_enabled() { [ "$1" = "1" ] || [ "$1" = "true" ]; }

curl -sf "$READYZ_URL" || exit 1

if is_enabled "$PMM_ENABLE_SEP" && ! is_enabled "$PMM_HA_ENABLE" &&
    ! is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES" && [ ! -e "$SEP_MARKER" ]; then
    # Leading newline: curl's unredirected body shares the Health.Log entry.
    printf '\nSEP provisioning has not completed on this start yet.\n' >&2
    exit 1
fi

exit 0
