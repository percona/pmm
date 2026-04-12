#!/usr/bin/env bash
set -o xtrace

sysbench \
    --db-driver=mysql \
    --mysql-host=ps \
    --mysql-port=3306 \
    --mysql-user=root \
    --mysql-password=secret \
    --mysql-db=sbtest \
    --table-size=1000000 \
    oltp_read_write \
    prepare

sysbench \
    --rate=200 \
    --threads=64 \
    --report-interval=10 \
    --time=0 \
    --events=0 \
    --rand-type=pareto \
    --db-driver=mysql \
    --mysql-host=ps \
    --mysql-port=3306 \
    --mysql-user=root \
    --mysql-password=secret \
    --mysql-db=sbtest \
    --table-size=1000000 \
    oltp_read_only \
    run
