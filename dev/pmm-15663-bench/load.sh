#!/bin/bash
# usage: load.sh [--stress] ; streams 8 generator shards into VictoriaMetrics inside the container
cd /root
NOW=$(date +%s%3N)
for i in 0 1 2 3 4 5 6 7; do
  ( FILLER_ONLY=${FILLER_ONLY:-0} SHARD=$i/8 python3 gen.py metrics $1 --now $NOW | gzip -1 | docker exec -i pmm-server curl -s --data-binary @- -H 'Content-Encoding: gzip' "http://127.0.0.1:9090/prometheus/api/v1/import" -o /dev/null -w "shard $i %{http_code}\n" ) 2>&1 &
done
wait
