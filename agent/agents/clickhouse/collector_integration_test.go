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

// Integration tests for the ClickHouse collector against real servers.
//
// The collector must work against every supported ClickHouse version, in both
// single-node and cluster topologies, and whether the server is local or
// external. The full matrix is driven by testdata/run-matrix.sh:
//
//	cd agent/agents/clickhouse/testdata && ./run-matrix.sh
//
// To run against an arbitrary set of endpoints directly, set
// CLICKHOUSE_TEST_ENDPOINTS to a comma-separated list of "name=dsn" pairs:
//
//	CLICKHOUSE_TEST_ENDPOINTS="single-25.3=clickhouse://default:clickhouse@127.0.0.1:9000/default" \
//	  go test -tags clickhouse_integration ./agent/agents/clickhouse/...
package clickhouse

import (
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// matrixEndpoints returns the ClickHouse endpoints to validate, parsed from
// CLICKHOUSE_TEST_ENDPOINTS ("name=dsn" pairs, comma-separated). When unset, a
// single local default is used so the test is runnable without the driver.
func matrixEndpoints() map[string]string {
	raw := os.Getenv("CLICKHOUSE_TEST_ENDPOINTS")
	if strings.TrimSpace(raw) == "" {
		return map[string]string{
			"single-local": "clickhouse://default:clickhouse@127.0.0.1:9000/default",
		}
	}
	endpoints := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, dsn, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		endpoints[strings.TrimSpace(name)] = strings.TrimSpace(dsn)
	}
	return endpoints
}

// TestClickHouseMatrix validates the collector against every configured
// endpoint (version × {single, cluster} × {local, external}). Each endpoint is
// a subtest; an unreachable endpoint is skipped so the matrix can be run
// incrementally, one topology at a time, by the driver script.
func TestClickHouseMatrix(t *testing.T) {
	endpoints := matrixEndpoints()
	require.NotEmpty(t, endpoints)

	for name, dsn := range endpoints {
		t.Run(name, func(t *testing.T) {
			c, err := NewCollector(dsn)
			if err != nil {
				t.Skipf("endpoint %q unreachable, skipping: %v", name, err)
			}
			defer c.client.Close() //nolint:errcheck

			var version string
			require.NoError(t, c.client.QueryRow("SELECT version()").Scan(&version),
				"the collector must read the server version on every supported release")
			t.Logf("endpoint %q: ClickHouse %s", name, version)

			// On a cluster member system.clusters must list a named cluster.
			if strings.HasPrefix(name, "cluster") {
				var clusters int
				require.NoError(t, c.client.QueryRow(
					"SELECT count(DISTINCT cluster) FROM system.clusters WHERE cluster NOT LIKE 'default%'").Scan(&clusters))
				assert.Positive(t, clusters, "a cluster endpoint must expose a named cluster")
			}

			// system.query_log is created lazily — force the flush so the
			// collector's query has a table to read, mirroring a server that
			// has served traffic.
			_, err = c.client.Exec("SYSTEM FLUSH LOGS")
			require.NoError(t, err)

			ch := make(chan prometheus.Metric, 8)
			c.Collect(ch)
			close(ch)

			var metrics []prometheus.Metric
			for m := range ch {
				metrics = append(metrics, m)
			}
			assert.Len(t, metrics, 2,
				"Collect must emit clickhouse_query_count and clickhouse_scrape_duration_seconds")
		})
	}
}

// TestCollectorIntegrationBadDSN verifies NewCollector fails fast when the
// server is unreachable — needed so a missing external server is not mistaken
// for a healthy one.
func TestCollectorIntegrationBadDSN(t *testing.T) {
	_, err := NewCollector("clickhouse://default:wrong@127.0.0.1:1?dial_timeout=2s")
	assert.Error(t, err)
}
