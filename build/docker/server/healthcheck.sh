#!/bin/bash
#
# The container's HEALTHCHECK. Runs the readiness probe the inline HEALTHCHECK command used
# to run and, only when PMM_ENABLE_SEP is set, additionally requires that this start's SEP
# provisioning run finished. A side-car gated on depends_on: {condition: service_healthy}
# would otherwise be released while grafana-sep is still minting the Grafana token that
# side-car reads once, at process start.
#
# Deliberately not under errexit: Docker reserves exit code 2, so every failure returns 1
# rather than letting a command's own status escape.

declare READYZ_URL="http://127.0.0.1:8080/v1/server/readyz"
# Removed by entrypoint.sh before supervisord starts and written by grafana-sep only on a
# successful run, so its presence describes this start rather than any file left on /srv.
declare SEP_MARKER="/srv/.sep_provisioned"

is_enabled() { [ "$1" = "1" ] || [ "$1" = "true" ]; }

curl -sf "$READYZ_URL" || exit 1

if is_enabled "$PMM_ENABLE_SEP" && [ ! -e "$SEP_MARKER" ]; then
    # Leading newline because curl's unredirected body lands in the same Health.Log entry.
    printf '\nSEP provisioning has not completed on this start yet.\n' >&2
    exit 1
fi

exit 0
