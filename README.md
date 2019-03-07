# qan-api

[![Build Status](https://travis-ci.org/Percona-Lab/qan-api.svg?branch=master)](https://travis-ci.org/Percona-Lab/qan-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/Percona-Lab/qan-api)](https://goreportcard.com/report/github.com/Percona-Lab/qan-api)
[![pullreminders](https://pullreminders.com/badge.svg)](https://pullreminders.com?ref=badge)

qan-api for PMM 2.x.


# Get Report


Examples:
```bash
curl -s -X POST -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00", "group_by": "d_client_host"}' http://127.0.0.1:9922/v1/qan/GetReport | jq
curl -X POST -s -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 23:00:00",  "labels": [{"key": "d_client_host", "value": ["10.11.12.4", "10.11.12.59"]}]}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00", "group_by": "d_client_host", "offset": 10}' http://127.0.0.1:9922/v1/qan/GetReport | jq

curl -s -X POST -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00", "order_by": "num_queries"}' http://127.0.0.1:9922/v1/qan/GetReport | jq

 curl -X POST -s -d '{"period_start_from": "2019-01-01 00:00:00", "period_start_to": "2019-01-01 01:00:00", "filter_by": "7DD5F6760F2D2EBB"}' http://127.0.0.1:9922/v1/qan/GetMetrics | jq
 ```
