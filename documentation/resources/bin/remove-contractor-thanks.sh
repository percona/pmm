#!/bin/bash
# Percona contractors who create tickets don't need to be thanked in the same
# way as community contributors are. Jirabot can't easily find out whether a
# contributor is a Percona contractor or not. This script deletes them by name.
# USAGE: resources/bin/remove-contractor-thanks.sh docs/release-notes/3.x.0.md

PATTERN="(Thanks to %s for reporting this issue)"
NAMES=(
    "Jiří Čtvrtka"
    )

FILE=$1

for n in "${NAMES[@]}"; do
    PATT=$(printf "s/${PATTERN}//\n" "${n}")
    echo "Removing thanks for $n"
    sed -i "${PATT}" $FILE
done
