#!/bin/sh
# Fetch the build log from the webhook host once, place it where nginx will
# serve it, then hand off to nginx (the CMD).

set -eu

: "${LOG_UUID:?LOG_UUID env var is required (the UUID printed in the PR comment)}"
: "${WEBHOOK_URL:?WEBHOOK_URL env var is required (e.g. https://builds.example.com)}"

DEST=/usr/share/nginx/html/log.txt
URL="${WEBHOOK_URL%/}/logs/${LOG_UUID}"

# CURL_OPTS is intentionally word-split — set e.g. "-k" for self-signed certs
# or "--resolve host:port:ip" when bypassing DNS.
# shellcheck disable=SC2086
if curl -fsS ${CURL_OPTS:-} -o "${DEST}" "${URL}"; then
    bytes=$(wc -c <"${DEST}" | tr -d ' ')
    echo "Downloaded ${bytes} bytes from ${URL}"
else
    echo "ERROR: failed to download log from ${URL}" >&2
    cat >"${DEST}" <<EOF
log-viewer could not download the log from:
  ${URL}

Common causes:
  - LOG_UUID is wrong, or the log was pruned from the webhook host
  - WEBHOOK_URL is unreachable from this container
  - TLS verification failed (set CURL_OPTS=-k for self-signed certs)
EOF
fi

exec "$@"
