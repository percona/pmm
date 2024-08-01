#!/bin/bash

# This script checks if PMM Server has finished upgrading so pmm-agent can perform the `setup` command.
# If PMM Server is not ready, the script will wait for 30 seconds and then exit with an error.

if ! timeout 30 bash -c "until supervisorctl status pmm-update-perform-init | grep -q EXITED; do sleep 2; done"; then
    echo "FATAL: failed to set up pmm-agent."
    exit 1
fi
