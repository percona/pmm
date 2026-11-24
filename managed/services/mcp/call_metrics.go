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

package mcp

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "mcp"

	// The code label of a call that succeeded; a failed call carries its error code.
	codeOK = "ok"
)

// toolDurationBuckets span a catalog lookup of milliseconds up to a live inspection that runs into its timeout.
var toolDurationBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 60, 120}

// callMetrics counts the tool calls by tool and result code, and measures their duration and the calls in flight.
type callMetrics struct {
	calls    *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

func newCallMetrics() *callMetrics {
	return &callMetrics{
		calls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "tool_calls_total",
			Help:      "Number of MCP tool calls by tool and result code: ok, or the error code of a failed call.",
		}, []string{"tool", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "tool_duration_seconds",
			Help:      "Duration of MCP tool calls by tool.",
			Buckets:   toolDurationBuckets,
		}, []string{"tool"}),
		inFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "tool_calls_in_flight",
			Help:      "Number of MCP tool calls in flight by tool.",
		}, []string{"tool"}),
	}
}

// register registers the metrics on r.
func (m *callMetrics) register(r prometheus.Registerer) error {
	for _, c := range []prometheus.Collector{m.calls, m.duration, m.inFlight} {
		err := r.Register(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// begin counts a call of tool in flight, and returns the function that records its end with its result code and
// returns its duration.
func (m *callMetrics) begin(tool string) func(code string) time.Duration {
	start := time.Now()
	inFlight := m.inFlight.WithLabelValues(tool)
	inFlight.Inc()
	return func(code string) time.Duration {
		duration := time.Since(start)
		inFlight.Dec()
		m.duration.WithLabelValues(tool).Observe(duration.Seconds())
		m.calls.WithLabelValues(tool, code).Inc()
		return duration
	}
}
