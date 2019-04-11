# qan-api2

[![Build Status](https://travis-ci.org/percona/qan-api2.svg?branch=master)](https://travis-ci.org/percona/qan-api2)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/qan-api2)](https://goreportcard.com/report/github.com/percona/qan-api2)
[![pullreminders](https://pullreminders.com/badge.svg)](https://pullreminders.com?ref=badge)

qan-api for PMM 2.x.

## Get Report

Examples:
```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "d_client_host"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z",  "labels": [{"key": "d_client_host", "value": ["10.11.12.4", "10.11.12.59"]}]}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "d_client_host", "offset": 10}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

```

```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries", "columns": ["lock_time", "sort_scan"], "group_by": "d_server"}' http://127.0.0.1:9922/v1/qan/GetReport | jq
 ```

 ```bash
 curl -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}'  http://127.0.0.1:9922/v1/qan/Filters/Get
 ```

## Get list of availible metrics.

`curl -X POST -d '{}' http://127.0.0.1:9922/v1/qan/GetMetricsNames -s | jq`

```json
{
  "data": {
    "bytes_sent": "Bytes Sent",
    "count": "Count",
    "docs_returned": "Docs Returned",
    "docs_scanned": "Docs Scanned",
    "filesort": "Filesort",
    "filesort_on_disk": "Filesort on Disk",
    "full_join": "Full Join",
    "full_scan": "Full Scan",
    "innodb_io_r_bytes": "Innodb IO R Bytes",
    "innodb_io_r_ops": "Innodb IO R Ops",
    "innodb_io_r_wait": "Innodb IO R Wait",
    "innodb_pages_distinct": "Innodb Pages Distinct",
    "innodb_queue_wait": "Innodb Queue Wait",
    "innodb_rec_lock_wait": "Innodb Rec Lock Wait",
    "latancy": "Latancy",
    "load": "Load",
    "lock_time": "Lock Time",
    "merge_passes": "Merge Passes",
    "no_good_index_used": "No Good Index Used",
    "no_index_used": "No Index Used",
    "qc_hit": "Query Cache Hit",
    "query_length": "Query Length",
    "query_time": "Query Time",
    "response_length": "Response Length",
    "rows_affected": "Rows Affected",
    "rows_examined": "Rows Examined",
    "rows_read": "Rows Read",
    "rows_sent": "Rows Sent",
    "select_full_range_join": "Select Full Range Join",
    "select_range": "Select Range",
    "select_range_check": "Select Range Check",
    "sort_range": "Sort Range",
    "sort_rows": "Sort Rows",
    "sort_scan": "Sort Scan",
    "tmp_disk_tables": "Tmp Disk Tables",
    "tmp_table": "Tmp Table",
    "tmp_table_on_disk": "Tmp Table on Disk",
    "tmp_table_sizes": "Tmp Table Sizes",
    "tmp_tables": "Tmp Tables"
  }
}
```

## Get Query Exemples

`curl 'http://localhost:9922/v1/qan/ObjectDetails/GetQueryExample' -XPOST -d '{"filter_by":"1D410B4BE5060972","group_by":"queryid","limit":5,"period_start_from":"2018-12-31T22:00:00+00:00","period_start_to":"2019-01-01T06:00:00+00:00"}' -s | jq`

```json
{
  "query_examples": [
    {
      "example": "Ping",
      "example_format": "EXAMPLE",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_format": "EXAMPLE",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_format": "EXAMPLE",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_format": "EXAMPLE",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_format": "EXAMPLE",
      "example_type": "RANDOM"
    }
  ]
}
```

## Get metrics

`curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "db1", "group_by": "d_server"}' http://127.0.0.1:9922/v1/qan/ObjectDetails/GetMetrics`

```json
{
  "metrics": {
    "bytes_sent": {
      "rate": 60.038887,
      "cnt": 1834,
      "sum": 2161400,
      "max": 249,
      "avg": 1178.5168,
      "p99": 117.85169
    },
    "docs_returned": {},
    "docs_scanned": {},
    "filesort": {},
    "filesort_on_disk": {},
    "full_join": {},
    "full_scan": {
      "rate": 0.17,
      "sum": 6120
    },
    "innodb_io_r_bytes": {},
    "innodb_io_r_ops": {},
    "innodb_io_r_wait": {},
    "innodb_pages_distinct": {},
    "innodb_queue_wait": {},
    "innodb_rec_lock_wait": {},
    "lock_time": {
      "rate": 1.4918888e-05,
      "cnt": 1834,
      "sum": 0.53708,
      "max": 0.000179,
      "avg": 0.00029284623,
      "p99": 2.9284623e-05
    },
    "merge_passes": {
      "cnt": 1834
    },
    "no_good_index_used": {},
    "no_index_used": {},
    "qc_hit": {},
    "query_length": {},
    "query_time": {
MacBook2:qan-api2 als$ git diff Makefile
      "rate": 0.00030471332,
      "cnt": 1834,
      "sum": 10.96968,
      "min": 2e-06,
      "max": 0.003068,
      "avg": 0.0059812865,
      "p99": 0.0005981287
    },
    "response_length": {},
    "rows_affected": {
      "cnt": 1834
    },
    "rows_examined": {
      "rate": 187.64,
      "cnt": 1834,
      "sum": 6755040,
      "max": 1268,
      "avg": 3683.228,
      "p99": 368.32278
    },
    "rows_read": {},
    "rows_sent": {
      "rate": 0.3563889,
      "cnt": 1834,
      "sum": 12830,
      "max": 1,
      "avg": 6.995638,
      "p99": 0.6995638
    },
    "select_full_range_join": {},
    "select_range": {},
    "select_range_check": {},
    "sort_range": {},
    "sort_rows": {},
    "sort_scan": {},
    "tmp_disk_tables": {
      "cnt": 1834
    },
    "tmp_table": {
      "rate": 0.17,
      "sum": 6120
    },
    "tmp_table_on_disk": {},
    "tmp_table_sizes": {
      "cnt": 1834
    },
    "tmp_tables": {
      "rate": 0.17,
      "cnt": 1834,
      "sum": 6120,
      "max": 1,
      "avg": 3.3369684,
      "p99": 0.33369684
    }
  }
}

```
