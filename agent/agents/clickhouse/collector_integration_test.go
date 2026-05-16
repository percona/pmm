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

//go:build clickhouse_integration

// Integration tests for the ClickHouse collector against a real server.
//
// Run with a ClickHouse instance available (see agent/docker-compose.yml):
//
//	docker compose -f agent/docker-compose.yml up -d clickhouse
//	go test -tags clickhouse_integration ./agent/agents/clickhouse/...
package clickhouse

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integrationDSN returns the DSN for the test ClickHouse, overridable via
// CLICKHOUSE_DSN; the default matches agent/docker-compose.yml.
func integrationDSN() string {
	if dsn := os.Getenv("CLICKHOUSE_DSN"); dsn != "" {
		return dsn
	}
	return "clickhouse://default:clickhouse@127.0.0.1:9000/default"
}

func TestCollectorIntegrationNewCollector(t *testing.T) {
	c, err := NewCollector(integrationDSN())
	require.NoError(t, err, "NewCollector must connect and ping the server")
	require.NoError(t, c.client.Close())
}

func TestCollectorIntegrationCollect(t *testing.T) {
	c, err := NewCollector(integrationDSN())
	require.NoError(t, err)
	defer c.client.Close() //nolint:errcheck

	// system.query_log is created lazily by ClickHouse — it does not exist on a
	// server that has never flushed its logs. Force the flush so the test
	// mirrors a real monitored server that has served (and logged) traffic.
	_, err = c.client.Exec("SYSTEM FLUSH LOGS")
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 8)
	c.Collect(ch)
	close(ch)

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	assert.Len(t, metrics, 2, "Collect against a live server must emit both metrics")
}

func TestCollectorIntegrationBadDSN(t *testing.T) {
	_, err := NewCollector("clickhouse://default:wrong@127.0.0.1:1?dial_timeout=2s")
	assert.Error(t, err, "NewCollector must fail when the server is unreachable")
}
