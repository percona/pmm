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

// Command clickhouse_exporter is a Prometheus exporter for ClickHouse. It is
// managed by pmm-agent for ClickHouse instances that do not expose the native
// <prometheus> endpoint, and emits the same metric families as that endpoint.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/percona/pmm/agent/agents/clickhouse"
	"github.com/percona/pmm/version"
)

// readHeaderTimeout bounds how long the metrics HTTP server waits for request headers.
const readHeaderTimeout = 5 * time.Second

// versionLine is the banner the pmm-agent version probe matches.
func versionLine() string {
	return "clickhouse_exporter, version " + version.Version
}

// newMetricsHandler builds the /metrics HTTP handler backed by the given
// collector, registered on a private registry.
func newMetricsHandler(collector prometheus.Collector) (http.Handler, error) {
	registry := prometheus.NewRegistry()
	regErr := registry.Register(collector)
	if regErr != nil {
		return nil, regErr
	}
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), nil
}

// run wires the collector and serves /metrics; it returns an error instead of
// exiting so deferred cleanup always runs.
func run(dsn, listenAddress, telemetryPath string) error {
	collector, err := clickhouse.NewCollector(dsn)
	if err != nil {
		return fmt.Errorf("cannot connect to ClickHouse: %w", err)
	}
	defer collector.Close() //nolint:errcheck

	handler, err := newMetricsHandler(collector)
	if err != nil {
		return fmt.Errorf("cannot build metrics handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(telemetryPath, handler)

	srv := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	log.Printf("%s — serving metrics on %s%s", versionLine(), listenAddress, telemetryPath)
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	cfg := clickhouse.LoadConfig()

	app := kingpin.New("clickhouse_exporter", "Prometheus exporter for ClickHouse metrics.")
	app.HelpFlag.Short('h')
	listenAddress := app.Flag("web.listen-address", "Address on which to expose metrics.").
		Default(cfg.ListenAddress).String()
	telemetryPath := app.Flag("web.telemetry-path", "Path under which to expose metrics.").
		Default(cfg.TelemetryPath).String()
	dsn := app.Flag("clickhouse.dsn", "ClickHouse DSN ["+clickhouse.EnvDSN+"].").
		Default(cfg.DSN).String()
	showVersion := app.Flag("version", "Print version information and exit.").Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *showVersion {
		fmt.Println(versionLine()) //nolint:forbidigo
		return
	}

	err := run(*dsn, *listenAddress, *telemetryPath)
	if err != nil {
		log.Fatalf("clickhouse_exporter: %v", err)
	}
}
