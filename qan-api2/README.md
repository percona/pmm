# qan-api2

QAN API for PMM 2.x.

## Get Report

Examples:

```bash

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "queryid"}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "client_host"}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq

curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z",  "labels": [{"key": "client_host", "value": ["10.11.12.4", "10.11.12.59"]}]}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "group_by": "client_host", "offset": 10}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries"}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq

```

```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries", "columns": ["lock_time", "sort_scan"], "group_by": "server"}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq
```

```bash
curl -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z"}'  http://127.0.0.1:9922/v1/qan/metrics:getFilters 
```

## Get list of available metrics.

```bash
curl -s -X POST -d '{}' http://127.0.0.1:9922/v1/qan/metrics:getNames | jq`
```

```json
{
  "data": {
    "application_name": "Name provided by pg_stat_monitor",
    "bytes_sent": "Bytes Sent",
    "cmd_type": "Type of SQL command used in the query",
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
    "tmp_tables": "Tmp Tables",
    "top_query": "Top query plain text",
    "top_queryid": "Top parent query ID"    
  }
}
```

## Get Query Examples

`curl 'http://localhost:9922/v1/qan/query:getExample ' -X POST -d '{"filter_by":"1D410B4BE5060972","group_by":"queryid","limit":5,"period_start_from":"2018-12-31T22:00:00+00:00","period_start_to":"2019-01-01T06:00:00+00:00"}' -s | jq`

```json
{
  "query_examples": [
    {
      "example": "Ping",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_type": "RANDOM"
    },
    {
      "example": "Ping",
      "example_type": "RANDOM"
    }
  ]
}
```

## Get metrics

```bash
curl -X POST -s -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "1D410B4BE5060972", "group_by": "queryid"}' http://127.0.0.1:9922/v1/qan:getMetrics
```

```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "order_by": "num_queries", "columns": ["lock_time", "sort_scan"], "group_by": "server"}' http://127.0.0.1:9922/v1/qan/metrics:getReport | jq '.rows[].load'
```

```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01T00:00:00Z", "period_start_to": "2019-01-01T10:00:00Z", "filter_by": "1D410B4BE5060972", "group_by": "queryid"}' http://127.0.0.1:9922/v1/qan:getLabels | jq
```
