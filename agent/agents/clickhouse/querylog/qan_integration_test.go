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

// Integration tests for the ClickHouse QAN agent (ClickHouseQueryLog) against
// real servers. The agent reads completed query executions from
// system.query_log and folds them into one MetricsBucket per fingerprinted
// query class.
//
// The endpoint matrix mirrors collector_integration_test.go's, parsed from the
// CLICKHOUSE_TEST_ENDPOINTS variable ("name=dsn" pairs, comma-separated). The
// matrixEndpoints helper is redefined here because this is a different package
// (querylog) and the collector's helper cannot cross the package boundary.
//
//	CLICKHOUSE_TEST_ENDPOINTS="single-25.3=clickhouse://default:clickhouse@127.0.0.1:9000/default" \
//	  go test -tags clickhouse_integration ./agent/agents/clickhouse/...
//
// Driving approach: the test exercises the agent's own collection path — the
// unexported preflight + collect methods — for exactly ONE interval, rather
// than starting the full Run loop. Run schedules collection on the next minute
// boundary, which would make a test wait up to ~60s and depend on wall-clock
// timing; calling collect directly is deterministic and asserts the same code
// path (preflight -> readRows -> makeBuckets) that Run uses. The test is in
// package querylog precisely so it can reach those methods.
//
// Query-log flush: ClickHouse buffers system.query_log rows and flushes them
// asynchronously (default flush_interval_milliseconds = 7500). The agent never
// issues SYSTEM FLUSH LOGS, and neither does this test — instead it polls
// system.query_log until the expected rows appear, which is exactly how a real
// deployment behaves and keeps the assertion free of a privileged statement.

package querylog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // database/sql driver "clickhouse"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// flushPollTimeout bounds how long the test waits for buffered system.query_log
// rows to be flushed and become visible.
const flushPollTimeout = 30 * time.Second

// selectQueryCount is the number of literal-varying SELECTs the test runs; they
// share a fingerprint and must collapse into a single bucket.
const selectQueryCount = 5

// matrixEndpoints parses CLICKHOUSE_TEST_ENDPOINTS into "name -> dsn" pairs,
// matching the collector test's helper. When unset, a single local default is
// used so the test is runnable without the driver script.
func matrixEndpoints() map[string]string {
	raw := os.Getenv("CLICKHOUSE_TEST_ENDPOINTS")
	if strings.TrimSpace(raw) == "" {
		return map[string]string{
			"single-local": "clickhouse://default:clickhouse@127.0.0.1:9000/default",
		}
	}
	endpoints := make(map[string]string)
	for pair := range strings.SplitSeq(raw, ",") {
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

// TestClickHouseQANMatrix validates the QAN agent against every configured
// endpoint. For each one it generates a known workload, waits for the
// query-log flush, drives one collection interval and asserts the resulting
// buckets. An unreachable endpoint is skipped so the matrix can be run
// incrementally.
func TestClickHouseQANMatrix(t *testing.T) {
	endpoints := matrixEndpoints()
	require.NotEmpty(t, endpoints)

	for name, dsn := range endpoints {
		t.Run(name, func(t *testing.T) {
			db, err := sql.Open("clickhouse", dsn)
			require.NoError(t, err)
			defer db.Close()

			pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer pingCancel()
			pingErr := db.PingContext(pingCtx)
			if pingErr != nil {
				t.Skipf("endpoint %q unreachable, skipping: %v", name, pingErr)
			}

			// A unique marker keeps this run's query_log rows distinct from any
			// other test's, so the flush poll and assertions are not polluted
			// by concurrent or previous workloads.
			marker := fmt.Sprintf("pmm_qan_it_%d", time.Now().UnixNano())
			t.Logf("endpoint %q: workload marker %s", name, marker)

			// The agent only reads rows whose event_time is at or after its
			// watermark. system.query_log.event_time has 1-second granularity,
			// so the watermark is pinned to the start of the current second
			// before the workload runs: every query below then lands in a
			// second >= the watermark and is captured by the single collection
			// cycle. (In production the watermark is sub-second precise but
			// collection happens 60s later, so this boundary is immaterial.)
			agent, err := New(Params{
				DSN:            dsn,
				AgentID:        "qan-integration-test",
				MaxQueryLength: 0,
			}, logrus.WithField("test", t.Name()))
			require.NoError(t, err)
			defer agent.db.Close()
			agent.watermark = time.Now().Truncate(time.Second)

			table := "default." + marker
			generateWorkload(t, db, marker, table)
			waitForQueryLog(t, db, marker)
			buckets := collectOnce(t, agent)

			selectBucket, insertBucket := classifyBuckets(t, buckets, table)

			// The 5 literal-varying SELECTs share one fingerprint and must
			// collapse into exactly one bucket counting all of them.
			require.NotNil(t, selectBucket, "the SELECT class must produce a bucket")
			assert.InDelta(t, float32(selectQueryCount), selectBucket.Common.NumQueries, 1e-6,
				"the %d literal-varying SELECTs must collapse into one bucket", selectQueryCount)
			assert.Equal(t, "Select", selectBucket.Clickhouse.QueryKind)

			// The INSERT has a different shape and must be its own bucket.
			require.NotNil(t, insertBucket, "the INSERT class must produce a distinct bucket")
			assert.InDelta(t, float32(1), insertBucket.Common.NumQueries, 1e-6)
			assert.Equal(t, "Insert", insertBucket.Clickhouse.QueryKind)

			// Fingerprints must be literal-free: the per-query integer literal
			// is replaced by a placeholder, so the fingerprint carries "n = ?"
			// and never a concrete value. (The table name is an identifier,
			// not a literal, so it is preserved by design.)
			assert.Contains(t, selectBucket.Common.Fingerprint, "n = ?",
				"literal-varying SELECTs must produce a placeholder fingerprint")
			for i := range selectQueryCount {
				assert.NotContains(t, selectBucket.Common.Fingerprint, fmt.Sprintf("n = %d", i),
					"the fingerprint must not contain a concrete query literal")
			}

			// ClickHouse metrics and the common query-time stats must be
			// populated from the real server columns.
			assert.Positive(t, selectBucket.Common.MQueryTimeCnt, "query-time count must be populated")
			require.NotNil(t, selectBucket.Clickhouse, "the ClickHouse metrics sub-message must be set")
			assert.InDelta(t, float32(selectQueryCount), selectBucket.Clickhouse.MReadRowsCnt, 1e-6,
				"m_read_rows must have one sample per observed SELECT")
			assert.GreaterOrEqual(t, selectBucket.Clickhouse.MReadRowsSum, float32(0),
				"m_read_rows_sum must be populated from system.query_log")

			// The agent fills the period bookkeeping on every bucket.
			assert.Equal(t, "qan-integration-test", selectBucket.Common.AgentId)
			assert.Positive(t, selectBucket.Common.PeriodStartUnixSecs)
			assert.Positive(t, selectBucket.Common.PeriodLengthSecs)
		})
	}
}

// generateWorkload runs a deterministic set of queries that the agent must
// observe: selectQueryCount SELECTs that differ only by a literal (so they
// share a fingerprint) plus one INSERT into a per-run table. Every query
// touches the same per-run table, so the resulting buckets are identified by
// that table name — distinct from the test's own system.query_log poll queries.
func generateWorkload(t *testing.T, db *sql.DB, marker, table string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// log_queries must be on for system.query_log to be populated, and
	// log_comment tags every row with the run marker so the flush poll and
	// assertions ignore unrelated activity. The agent never changes server
	// settings, so the workload session enables these per statement; the
	// SETTINGS clause is placed where ClickHouse's grammar accepts it (after
	// the table definition / SELECT body, and before VALUES for an INSERT).
	settings := fmt.Sprintf(" SETTINGS log_queries = 1, log_comment = '%s'", marker)

	_, err := db.ExecContext(ctx,
		fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (n UInt64) ENGINE = Memory%s", table, settings))
	require.NoError(t, err, "creating the per-run table must succeed")
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_, _ = db.ExecContext(dropCtx, "DROP TABLE IF EXISTS "+table)
	})

	// Five SELECTs differing only by an integer literal — identical fingerprint.
	for i := range selectQueryCount {
		_, err = db.ExecContext(ctx,
			fmt.Sprintf("SELECT count() FROM %s WHERE n = %d%s", table, i, settings))
		require.NoError(t, err, "workload SELECT must succeed")
	}

	// One INSERT — a distinct query shape, hence a distinct bucket. For INSERT
	// the SETTINGS clause must precede VALUES.
	_, err = db.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (n)%s VALUES (1)", table, settings))
	require.NoError(t, err, "workload INSERT must succeed")
}

// waitForQueryLog polls system.query_log until this run's finished queries are
// visible, i.e. until ClickHouse has flushed its in-memory buffer. The test
// deliberately does not issue SYSTEM FLUSH LOGS — it waits for the same
// asynchronous flush the agent relies on in production.
func waitForQueryLog(t *testing.T, db *sql.DB, marker string) {
	t.Helper()
	// Expected finished rows: selectQueryCount SELECTs + 1 CREATE + 1 INSERT.
	const wantRows = selectQueryCount + 2

	deadline := time.Now().Add(flushPollTimeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var got int
		err := db.QueryRowContext(ctx,
			"SELECT count() FROM system.query_log "+
				"WHERE type = ? AND log_comment = ?",
			queryLogTypeQueryFinish, marker).Scan(&got)
		cancel()
		require.NoError(t, err, "reading system.query_log must succeed")

		if got >= wantRows {
			t.Logf("system.query_log flushed: %d finished rows for marker", got)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("system.query_log did not flush %d rows within %s (got %d)",
				wantRows, flushPollTimeout, got)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// collectOnce drives the agent through exactly one collection interval —
// preflight then collect — and returns the buckets. It exercises the same code
// path Run uses, without the minute-boundary scheduling that would make the
// test slow and timing-dependent.
func collectOnce(t *testing.T, agent *ClickHouseQueryLog) []*agentv1.MetricsBucket {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	columns, err := agent.preflight(ctx)
	require.NoError(t, err, "preflight must succeed against a healthy server with log_queries on")

	// The agent's watermark was set at New() time, before the workload ran, so
	// a single collect picks up every generated query.
	start := time.Now()
	buckets, err := agent.collect(ctx, columns, start, 60)
	require.NoError(t, err, "one collection cycle must succeed")
	require.NotEmpty(t, buckets, "the generated workload must yield at least one bucket")
	return buckets
}

// classifyBuckets picks out this run's SELECT and INSERT buckets by matching
// the per-run table among each bucket's touched tables. Matching on the table
// (not on the example text) keeps the test's own system.query_log poll queries
// — which mention the marker only as a literal — out of the result, and folds
// only the workload's own classes in.
func classifyBuckets(
	t *testing.T, buckets []*agentv1.MetricsBucket, table string,
) (selectBucket, insertBucket *agentv1.MetricsBucket) {
	t.Helper()
	for _, b := range buckets {
		require.NotNil(t, b.Common)
		require.NotNil(t, b.Clickhouse)
		if !slices.Contains(b.Common.Tables, table) {
			continue
		}
		switch b.Clickhouse.QueryKind {
		case "Select":
			selectBucket = b
		case "Insert":
			insertBucket = b
		}
	}
	return selectBucket, insertBucket
}
