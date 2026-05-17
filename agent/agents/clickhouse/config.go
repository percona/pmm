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

package clickhouse

import "os"

// Exporter configuration defaults and environment-variable names.
const (
	// EnvDSN holds the ClickHouse connection string when no flag is given.
	EnvDSN = "CLICKHOUSE_EXPORTER_DSN"
	// EnvListenAddress overrides the HTTP listen address.
	EnvListenAddress = "CLICKHOUSE_EXPORTER_WEB_LISTEN_ADDRESS"

	// DefaultDSN points at a local ClickHouse over the native protocol.
	DefaultDSN = "clickhouse://default@127.0.0.1:9000/default"
	// DefaultListenAddress is the host:port the exporter serves /metrics on.
	DefaultListenAddress = "127.0.0.1:9116"
	// DefaultTelemetryPath is the HTTP path that exposes Prometheus metrics.
	DefaultTelemetryPath = "/metrics"
)

// Config holds clickhouse_exporter runtime configuration.
type Config struct {
	// DSN is the ClickHouse connection string for the clickhouse-go/v2 driver.
	DSN string
	// ListenAddress is the host:port the exporter serves metrics on.
	ListenAddress string
	// TelemetryPath is the HTTP path that exposes Prometheus metrics.
	TelemetryPath string
}

// LoadConfig builds a Config from the defaults overlaid with environment
// variables. Command-line flags, when present, take precedence and are applied
// by the exporter's main package on top of the returned Config.
func LoadConfig() *Config {
	cfg := &Config{
		DSN:           DefaultDSN,
		ListenAddress: DefaultListenAddress,
		TelemetryPath: DefaultTelemetryPath,
	}
	if v := os.Getenv(EnvDSN); v != "" {
		cfg.DSN = v
	}
	if v := os.Getenv(EnvListenAddress); v != "" {
		cfg.ListenAddress = v
	}
	return cfg
}
