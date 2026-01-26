package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseURL = "http://127.0.0.1/v1/"

func getPeriodFromEnv() (string, string) {
	from := os.Getenv("PMM_DEMO_BENCH_PERIOD_FROM")
	to := os.Getenv("PMM_DEMO_BENCH_PERIOD_TO")
	if from == "" {
		from = "2025-12-27T00:00:00+01:00"
	}
	if to == "" {
		to = "2026-01-21T23:59:59+01:00"
	}

	return from, to
}

func benchmarkRequest(b *testing.B, url string, payload []byte) time.Duration {
	b.Helper()
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:admin")))

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		b.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	return duration
}

func benchmarkWithStats(b *testing.B, url string, payload []byte) {
	durations := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
		duration := benchmarkRequest(b, url, payload)
		durations[i] = duration
		fmt.Printf("iteration %d took %v\n", i, duration)
	}

	if b.N > 0 {
		var sum time.Duration
		min := durations[0]
		max := durations[0]
		for _, d := range durations {
			sum += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}
		avg := sum / time.Duration(b.N)
		fmt.Printf("avg=%v min=%v max=%v\n", avg, min, max)
	}
}

func BenchmarkGetFilters(b *testing.B) {
	url := baseURL + "qan/metrics:getFilters"
	from, to := getPeriodFromEnv()
	payload := []byte(fmt.Sprintf(`{
		   "labels": [],
		   "main_metric_name": "load",
		   "period_start_from": "%s",
		   "period_start_to": "%s"
	   }`, from, to))

	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetReport(b *testing.B) {
	url := baseURL + "qan/metrics:getReport"
	from, to := getPeriodFromEnv()
	payload := []byte(fmt.Sprintf(`{
		   "group_by": "queryid",
		   "include_only_fields": [],
		   "keyword": "",
		   "labels": [],
		   "limit": "25",
		   "main_metric": "load",
		   "offset": 0,
		   "order_by": "-load",
		   "period_start_from": "%s",
		   "period_start_to": "%s",
		   "search": ""
	   }`, from, to))

	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetMetrics(b *testing.B) {
	url := baseURL + "qan:getMetrics"
	from, to := getPeriodFromEnv()
	payload := []byte(fmt.Sprintf(`{
				 "filter_by": "0D1A4A519E3B08C0EADA79DF0F2034C7",
				 "group_by": "queryid",
				 "labels": [],
				 "period_start_from": "%s",
				 "period_start_to": "%s",
				 "tables": [],
				 "totals": false
			}`, from, to))

	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetExample(b *testing.B) {
	url := baseURL + "qan/query:getExample"
	from, to := getPeriodFromEnv()
	payload := []byte(fmt.Sprintf(`{
		   "filter_by": "9AD8CA7F8CAC1812CC0F42D4205D5441",
		   "group_by": "queryid",
		   "labels": [],
		   "period_start_from": "%s",
		   "period_start_to": "%s",
		   "tables": []
	   }`, from, to))

	benchmarkWithStats(b, url, payload)
}
