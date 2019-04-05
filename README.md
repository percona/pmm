# qan-api2

[![Build Status](https://travis-ci.org/percona/qan-api2.svg?branch=master)](https://travis-ci.org/percona/qan-api2)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/qan-api2)](https://goreportcard.com/report/github.com/percona/qan-api2)
[![pullreminders](https://pullreminders.com/badge.svg)](https://pullreminders.com?ref=badge)

qan-api for PMM 2.x.


# Get Report


Examples:
```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "d_client_host"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z",  "labels": [{"key": "d_client_host", "value": ["10.11.12.4", "10.11.12.59"]}]}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "d_client_host", "offset": 10}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "7DD5F6760F2D2EBB"}' http://127.0.0.1:9922/v1/qan/GetMetrics | jq

 ```

```
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries", "columns": ["lock_time", "sort_scan"], "group_by": "d_server"}' http://127.0.0.1:9922/v1/qan/GetReport | jq
 ```

 ```
 curl -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}'  http://127.0.0.1:9922/v1/qan/Filters/Get
 ```

# Get list of availible metrics.

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


```
curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "7DD5F6760F2D2EBB", "group_by": "queryid"}' http://127.0.0.1:9922/v1/qan/GetMetrics | jq
{
  "metrics": {
    "bytes_sent": {
      "rate": 137.38889,
      "cnt": 2473,
      "sum": 4946000,
      "min": 200,
      "max": 200,
      "avg": 2000,
      "p99": 200
    },
    "docs_returned": {},
    "docs_scanned": {},
    "filesort": {},
    "filesort_on_disk": {},
    "full_join": {},
    "full_scan": {
      "rate": 0.6869444,
      "sum": 24730
    },
    "innodb_io_r_bytes": {},
    "innodb_io_r_ops": {},
    "innodb_io_r_wait": {},
    "innodb_pages_distinct": {},
    "innodb_queue_wait": {},
    "innodb_rec_lock_wait": {},
    "lock_time": {
      "rate": 4.5558332e-05,
      "cnt": 2473,
      "sum": 1.6401,
      "min": 5.2e-05,
      "max": 0.000179,
      "avg": 0.0006632026,
      "p99": 6.632026e-05
    },
    "merge_passes": {
      "cnt": 2473
    },
    "no_good_index_used": {},
    "no_index_used": {},
    "qc_hit": {},
    "query_length": {},
    "query_time": {
      "rate": 0.0012054495,
      "cnt": 2473,
      "sum": 43.39618,
      "min": 0.001584,
      "max": 0.003068,
      "avg": 0.01754799,
      "p99": 0.001754799
    },
    "response_length": {},
    "rows_affected": {
      "cnt": 2473
    },
    "rows_examined": {
      "rate": 871.04553,
      "cnt": 2473,
      "sum": 31357640,
      "min": 1268,
      "max": 1268,
      "avg": 12680,
      "p99": 1268
    },
    "rows_read": {},
    "rows_sent": {
      "rate": 0.6869444,
      "cnt": 2473,
      "sum": 24730,
      "min": 1,
      "max": 1,
      "avg": 10,
      "p99": 1
    },
    "select_full_range_join": {},
    "select_range": {},
    "select_range_check": {},
    "sort_range": {},
    "sort_rows": {},
    "sort_scan": {},
    "tmp_disk_tables": {
      "cnt": 2473
    },
    "tmp_table": {
      "rate": 0.6869444,
      "sum": 24730
    },
    "tmp_table_on_disk": {},
    "tmp_table_sizes": {
      "cnt": 2473
    },
    "tmp_tables": {
      "rate": 0.6869444,
      "cnt": 2473,
      "sum": 24730,
      "min": 1,
      "max": 1,
      "avg": 10,
      "p99": 1
    }
  }
}
```


```
MacBook2:~ als$ curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "db1", "group_by": "d_server"}' http://127.0.0.1:9922/v1/qan/GetMetrics | jq
{
  "metrics": {
    "bytes_sent": {
      "rate": 59.971943,
      "cnt": 1832,
      "sum": 2158990,
      "max": 249,
      "avg": 1178.488,
      "p99": 117.8488
    },
    "docs_returned": {},
    "docs_scanned": {},
    "filesort": {},
    "filesort_on_disk": {},
    "full_join": {},
    "full_scan": {
      "rate": 0.16972223,
      "sum": 6110
    },
    "innodb_io_r_bytes": {},
    "innodb_io_r_ops": {},
    "innodb_io_r_wait": {},
    "innodb_pages_distinct": {},
    "innodb_queue_wait": {},
    "innodb_rec_lock_wait": {},
    "lock_time": {
      "rate": 1.4885e-05,
      "cnt": 1832,
      "sum": 0.53586,
      "max": 0.000179,
      "avg": 0.00029250002,
      "p99": 2.925e-05
    },
    "merge_passes": {
      "cnt": 1832
    },
    "no_good_index_used": {},
    "no_index_used": {},
    "qc_hit": {},
    "query_length": {},
    "query_time": {
      "rate": 0.00030420528,
      "cnt": 1832,
      "sum": 10.95139,
      "min": 2e-06,
      "max": 0.003068,
      "avg": 0.005977833,
      "p99": 0.0005977833
    },
    "response_length": {},
    "rows_affected": {
      "cnt": 1832
    },
    "rows_examined": {
      "rate": 187.36778,
      "cnt": 1832,
      "sum": 6745240,
      "max": 1268,
      "avg": 3681.8997,
      "p99": 368.18994
    },
    "rows_read": {},
    "rows_sent": {
      "rate": 0.35583332,
      "cnt": 1832,
      "sum": 12810,
      "max": 1,
      "avg": 6.992358,
      "p99": 0.6992358
    },
    "select_full_range_join": {},
    "select_range": {},
    "select_range_check": {},
    "sort_range": {},
    "sort_rows": {},
    "sort_scan": {},
    "tmp_disk_tables": {
      "cnt": 1832
    },
    "tmp_table": {
      "rate": 0.16972223,
      "sum": 6110
    },
    "tmp_table_on_disk": {},
    "tmp_table_sizes": {
      "cnt": 1832
    },
    "tmp_tables": {
      "rate": 0.16972223,
      "cnt": 1832,
      "sum": 6110,
      "max": 1,
      "avg": 3.3351529,
      "p99": 0.3335153
    }
  }
}
```
