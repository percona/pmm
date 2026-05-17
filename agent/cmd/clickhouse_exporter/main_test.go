// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/agents/clickhouse"
)

// stubCollector is a minimal prometheus.Collector that emits a single gauge,
// standing in for the real ClickHouse collector so the HTTP layer can be
// exercised without a live database.
type stubCollector struct {
	desc *prometheus.Desc
}

func newStubCollector() *stubCollector {
	return &stubCollector{
		desc: prometheus.NewDesc("ClickHouseMetrics_Query", "stub metric", nil, nil),
	}
}

func (c *stubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *stubCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1)
}

func TestNewMetricsHandlerServesMetrics(t *testing.T) {
	handler, err := newMetricsHandler(newStubCollector())
	require.NoError(t, err)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "ClickHouseMetrics_Query")
}

func TestVersionLine(t *testing.T) {
	line := versionLine()
	assert.True(t, strings.HasPrefix(line, "clickhouse_exporter, version "),
		"unexpected version banner: %q", line)
}

func TestFlagParsingDefaults(t *testing.T) {
	t.Setenv(clickhouse.EnvDSN, "")
	t.Setenv(clickhouse.EnvListenAddress, "")
	cfg := clickhouse.LoadConfig()

	app := kingpin.New("clickhouse_exporter", "test")
	listenAddress := app.Flag("web.listen-address", "").Default(cfg.ListenAddress).String()
	telemetryPath := app.Flag("web.telemetry-path", "").Default(cfg.TelemetryPath).String()
	dsn := app.Flag("clickhouse.dsn", "").Default(cfg.DSN).String()

	_, err := app.Parse(nil)
	require.NoError(t, err)

	assert.Equal(t, clickhouse.DefaultListenAddress, *listenAddress)
	assert.Equal(t, clickhouse.DefaultTelemetryPath, *telemetryPath)
	assert.Equal(t, clickhouse.DefaultDSN, *dsn)
}

func TestFlagParsingOverrides(t *testing.T) {
	cfg := &clickhouse.Config{
		ListenAddress: clickhouse.DefaultListenAddress,
		TelemetryPath: clickhouse.DefaultTelemetryPath,
	}

	app := kingpin.New("clickhouse_exporter", "test")
	listenAddress := app.Flag("web.listen-address", "").Default(cfg.ListenAddress).String()
	telemetryPath := app.Flag("web.telemetry-path", "").Default(cfg.TelemetryPath).String()

	_, err := app.Parse([]string{
		"--web.listen-address=0.0.0.0:9999",
		"--web.telemetry-path=/ch",
	})
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9999", *listenAddress)
	assert.Equal(t, "/ch", *telemetryPath)
}
