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

// Package irt provides Instrumented http.RoundTrippers.
package irt

import (
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// WithMetrics returns http.RoundTripper instrumented with returned Prometheus metrics.
func WithMetrics(t http.RoundTripper, subsystem string) (http.RoundTripper, prom.Collector) {
	m := &metrics{
		inflight: prom.NewGauge(prom.GaugeOpts{
			Namespace: "promhttp",
			Subsystem: subsystem,
			Name:      "requests_in_flight",
			Help:      "Current number of in-flight requests.",
		}),
		counter: prom.NewCounterVec(prom.CounterOpts{
			Namespace: "promhttp",
			Subsystem: subsystem,
			Name:      "responses_count",
			Help:      "Number of responses received.",
		}, []string{"code", "method"}),
		duration: prom.NewHistogramVec(prom.HistogramOpts{
			Namespace: "promhttp",
			Subsystem: subsystem,
			Name:      "responses_seconds",
			Help:      "Histogram of response latency (seconds).",
			Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 3.0},
		}, []string{"code", "method"}),
	}

	t = promhttp.InstrumentRoundTripperInFlight(m.inflight, t)
	t = promhttp.InstrumentRoundTripperCounter(m.counter, t)
	t = promhttp.InstrumentRoundTripperDuration(m.duration, t)
	// TODO InstrumentRoundTripperTrace

	return t, m
}

type metrics struct {
	inflight prom.Gauge
	counter  *prom.CounterVec
	duration prom.ObserverVec
}

// Describe implements prom.Collector.
func (m *metrics) Describe(ch chan<- *prom.Desc) {
	m.inflight.Describe(ch)
	m.counter.Describe(ch)
	m.duration.Describe(ch)
}

// Collect implements prom.Collector.
func (m *metrics) Collect(ch chan<- prom.Metric) {
	m.inflight.Collect(ch)
	m.counter.Collect(ch)
	m.duration.Collect(ch)
}

// check interfaces.
var (
	_ prom.Collector = (*metrics)(nil)
)
