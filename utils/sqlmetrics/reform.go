// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlmetrics

import (
	"strings"
	"sync/atomic"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/reform.v1"
)

// Reform is a SQL logger with metrics.
type Reform struct {
	l          *reform.PrintfLogger
	requests   atomic.Int64
	mRequests  *prom.CounterVec
	mResponses *prom.SummaryVec
}

// NewReform creates a new logger with given parameters.
func NewReform(driver, dbName string, printf reform.Printf) *Reform {
	constLabels := prom.Labels{
		"driver": driver,
		"db":     dbName,
	}

	return &Reform{
		l: reform.NewPrintfLogger(printf),
		mRequests: prom.NewCounterVec(prom.CounterOpts{
			Namespace:   "go_sql",
			Subsystem:   "reform",
			Name:        "requests_total",
			Help:        "Total number of queries started.",
			ConstLabels: constLabels,
		}, []string{"statement"}),
		mResponses: prom.NewSummaryVec(prom.SummaryOpts{
			Namespace:   "go_sql",
			Subsystem:   "reform",
			Name:        "response_seconds",
			Help:        "Response durations in seconds.",
			ConstLabels: constLabels,
			Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"statement", "error"}),
	}
}

func statement(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	parts := strings.SplitN(query, " ", 2)
	if len(parts) != 2 {
		return query
	}
	return parts[0]
}

// Before implements reform.Logger.
func (r *Reform) Before(query string, args []any) {
	r.l.Before(query, args)

	r.requests.Add(1)

	r.mRequests.WithLabelValues(statement(query)).Inc()
}

// After implements reform.Logger.
func (r *Reform) After(query string, args []any, d time.Duration, err error) {
	r.l.After(query, args, d, err)

	e := "0"
	if err != nil {
		e = "1"
	}
	r.mResponses.WithLabelValues(statement(query), e).Observe(d.Seconds())
}

// Describe implements prom.Collector.
func (r *Reform) Describe(ch chan<- *prom.Desc) {
	r.mRequests.Describe(ch)
	r.mResponses.Describe(ch)
}

// Collect implements prom.Collector.
func (r *Reform) Collect(ch chan<- prom.Metric) {
	r.mRequests.Collect(ch)
	r.mResponses.Collect(ch)
}

// Requests returns a total number of queries started.
func (r *Reform) Requests() int {
	return int(r.requests.Load())
}

// Reset sets all metrics to 0.
func (r *Reform) Reset() {
	r.requests.Store(0)

	r.mRequests.Reset()
	r.mResponses.Reset()
}

var (
	// Check interfaces.
	_ reform.Logger  = (*Reform)(nil)
	_ prom.Collector = (*Reform)(nil)
)
