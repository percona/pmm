// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

const baseURL = "http://127.0.0.1/v1/"

func getPeriodFromEnv(b *testing.B) (string, string) {
	b.Helper()
	from := os.Getenv("PMM_DEMO_BENCH_PERIOD_FROM")
	to := os.Getenv("PMM_DEMO_BENCH_PERIOD_TO")
	if from == "" {
		b.Fatalf("PMM_DEMO_BENCH_PERIOD_FROM is not set")
	}
	if to == "" {
		b.Fatalf("PMM_DEMO_BENCH_PERIOD_TO is not set")
	}

	return from, to
}

// buildPayload builds a JSON payload with custom fields and always appends period_start_from and period_start_to.
func buildPayload(b *testing.B, customFields string) []byte {
	b.Helper()
	from, to := getPeriodFromEnv(b)
	// Minimize JSON: remove all whitespace, newlines, and tabs using strings.Builder
	var sb strings.Builder
	for _, c := range customFields {
		if c != ' ' && c != '\n' && c != '\t' {
			sb.WriteByte(byte(c))
		}
	}
	trimmed := sb.String()
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '}' {
		trimmed = trimmed[:len(trimmed)-1]
	}
	payload := fmt.Appendf(nil, trimmed+",\"period_start_from\":\"%s\",\"period_start_to\":\"%s\"}", from, to)

	return payload
}

func benchmarkRequest(b *testing.B, url string, params string) time.Duration {
	b.Helper()
	payload := buildPayload(b, params)
	req, err := http.NewRequestWithContext(b.Context(), http.MethodPost, url, bytes.NewBuffer(payload))
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

func benchmarkWithStats(b *testing.B, url string, payload string) {
	b.Helper()
	durations := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
		duration := benchmarkRequest(b, url, payload)
		durations[i] = duration
		b.Logf("iteration %d took %v", i, duration)
	}

	if b.N > 0 {
		var sum time.Duration
		for _, d := range durations {
			sum += d
		}
		avgDuration := sum / time.Duration(b.N)
		minDuration := slices.Min(durations)
		maxDuration := slices.Max(durations)

		b.Logf("avg=%v min=%v max=%v\n", avgDuration, minDuration, maxDuration)
	}
}

func BenchmarkGetFilters(b *testing.B) {
	url := baseURL + "qan/metrics:getFilters"
	payload := `{
		"labels": [],
		"main_metric_name": "load"}`
	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetReport(b *testing.B) {
	url := baseURL + "qan/metrics:getReport"
	payload := `{
		"group_by": "queryid",
		"include_only_fields": [],
		"keyword": "",
		"labels": [],
		"limit": "25",
		"main_metric": "load",
		"offset": 0,
		"order_by": "-load",
		"search": ""}`
	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetMetrics(b *testing.B) {
	url := baseURL + "qan:getMetrics"
	payload := `{
		"filter_by": "0D1A4A519E3B08C0EADA79DF0F2034C7",
		"group_by": "queryid",
		"labels": [],
		"tables": [],
		"totals": false}`
	benchmarkWithStats(b, url, payload)
}

func BenchmarkGetExample(b *testing.B) {
	url := baseURL + "qan/query:getExample"
	payload := `{
		"filter_by": "9AD8CA7F8CAC1812CC0F42D4205D5441",
		"group_by": "queryid",
		"labels": [],
		"tables": []}`
	benchmarkWithStats(b, url, payload)
}
