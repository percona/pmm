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

package querylog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestFingerprint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			"numeric literal",
			"SELECT * FROM t WHERE id = 42",
			"SELECT * FROM t WHERE id = ?",
		},
		{
			"string literal",
			"SELECT * FROM t WHERE name = 'alice'",
			"SELECT * FROM t WHERE name = ?",
		},
		{
			"string literal with doubled quote escape",
			"SELECT 'O''Brien'",
			"SELECT ?",
		},
		{
			"IN list collapses",
			"SELECT * FROM t WHERE id IN (1, 2, 3, 4)",
			"SELECT * FROM t WHERE id IN (?)",
		},
		{
			"array literal collapses",
			"SELECT [1, 2, 3] AS a",
			"SELECT [?] AS a",
		},
		{
			"tuple literal collapses",
			"SELECT (1, 2, 3)",
			"SELECT (?)",
		},
		{
			"LIMIT n",
			"SELECT * FROM t LIMIT 10",
			"SELECT * FROM t LIMIT ?",
		},
		{
			"LIMIT n, m",
			"SELECT * FROM t LIMIT 10, 20",
			"SELECT * FROM t LIMIT ?",
		},
		{
			"LIMIT n OFFSET m",
			"SELECT * FROM t LIMIT 10 OFFSET 20",
			"SELECT * FROM t LIMIT ?",
		},
		{
			"line comment stripped",
			"SELECT 1 -- a trailing comment\nFROM t",
			"SELECT ? FROM t",
		},
		{
			"block comment stripped",
			"SELECT /* hint */ 1 FROM t",
			"SELECT ? FROM t",
		},
		{
			"float and exponent literals",
			"SELECT 3.14, 2e10, 0xFF FROM t",
			"SELECT ? FROM t",
		},
		{
			"named query parameter preserved",
			"SELECT * FROM t WHERE id = {id:UInt64}",
			"SELECT * FROM t WHERE id = {id:UInt64}",
		},
		{
			"digit inside identifier is not a literal",
			"SELECT col1, col2 FROM t1",
			"SELECT col1, col2 FROM t1",
		},
		{
			"whitespace collapses",
			"SELECT   1    FROM     t",
			"SELECT ? FROM t",
		},
		{
			"literals-only differ produce same fingerprint",
			"SELECT * FROM t WHERE id = 999 AND name = 'bob'",
			"SELECT * FROM t WHERE id = ? AND name = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fingerprint(tt.query))
		})
	}
}

func TestFingerprintGrouping(t *testing.T) {
	t.Parallel()

	// Same shape, different literals -> identical fingerprint and hash.
	a := fingerprint("SELECT id FROM t WHERE id = 1")
	b := fingerprint("SELECT id FROM t WHERE id = 2")
	assert.Equal(t, a, b)
	assert.Equal(t, hashFingerprint(0, a), hashFingerprint(0, b))

	// Different shapes -> different hash.
	c := fingerprint("INSERT INTO t VALUES (1)")
	assert.NotEqual(t, hashFingerprint(0, a), hashFingerprint(0, c))
}

func TestHashFingerprint(t *testing.T) {
	t.Parallel()

	// A non-zero server hash is used verbatim (hex of the uint64).
	assert.Equal(t, "ff", hashFingerprint(255, "ignored"))
	// A zero server hash falls back to the client hash, which is stable.
	h1 := hashFingerprint(0, "SELECT ? FROM t")
	h2 := hashFingerprint(0, "SELECT ? FROM t")
	assert.Equal(t, h1, h2)
	assert.NotEmpty(t, h1)
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []float32
		p      float64
		want   float32
	}{
		{"empty slice", nil, 0.99, 0},
		{"single element", []float32{7}, 0.99, 7},
		{"two elements p99", []float32{1, 9}, 0.99, 9},
		{"two elements p50", []float32{1, 9}, 0.50, 1},
		{"all equal", []float32{5, 5, 5, 5}, 0.99, 5},
		{"sorted input p99", []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.99, 10},
		{"unsorted input p99", []float32{10, 1, 5, 3, 8}, 0.99, 10},
		{"p0 picks minimum", []float32{4, 1, 3, 2}, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, percentile(tt.values, tt.p), 1e-6)
		})
	}
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := []float32{3, 1, 2}
	_ = percentile(in, 0.99)
	assert.Equal(t, []float32{3, 1, 2}, in)
}

func TestMakeBuckets(t *testing.T) {
	t.Parallel()

	t.Run("identical queries collapse into one bucket", func(t *testing.T) {
		t.Parallel()
		rows := []queryLogRow{
			finishRow("SELECT id FROM t WHERE id = 1", 100, 10, 1),
			finishRow("SELECT id FROM t WHERE id = 2", 300, 30, 3),
			finishRow("SELECT id FROM t WHERE id = 3", 200, 20, 2),
		}
		buckets := makeBuckets(rows, 0)
		require.Len(t, buckets, 1)

		b := buckets[0]
		assert.InDelta(t, float32(3), b.Common.NumQueries, 1e-6)
		assert.Equal(t, "SELECT id FROM t WHERE id = ?", b.Common.Fingerprint)
		assert.Equal(t, inventoryv1.AgentType_AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT, b.Common.AgentType)

		// query_duration_ms 100+300+200 = 600 ms -> 0.6 s.
		assert.InDelta(t, 0.6, b.Common.MQueryTimeSum, 1e-6)
		assert.InDelta(t, float32(3), b.Common.MQueryTimeCnt, 1e-6)
		assert.InDelta(t, 0.1, b.Common.MQueryTimeMin, 1e-6)
		assert.InDelta(t, 0.3, b.Common.MQueryTimeMax, 1e-6)

		require.NotNil(t, b.Clickhouse)
		assert.InDelta(t, float32(60), b.Clickhouse.MReadRowsSum, 1e-6)
		assert.InDelta(t, float32(10), b.Clickhouse.MReadRowsMin, 1e-6)
		assert.InDelta(t, float32(30), b.Clickhouse.MReadRowsMax, 1e-6)
		assert.InDelta(t, float32(3), b.Clickhouse.MReadRowsCnt, 1e-6)
	})

	t.Run("distinct shapes produce distinct buckets", func(t *testing.T) {
		t.Parallel()
		rows := []queryLogRow{
			finishRow("SELECT id FROM t WHERE id = 1", 100, 10, 1),
			insertRow("INSERT INTO t VALUES (1)", 50, 5),
		}
		buckets := makeBuckets(rows, 0)
		require.Len(t, buckets, 2)

		kinds := map[string]bool{}
		for _, b := range buckets {
			kinds[b.Clickhouse.QueryKind] = true
		}
		assert.True(t, kinds["Select"])
		assert.True(t, kinds["Insert"])
	})

	t.Run("error rows populate error counters", func(t *testing.T) {
		t.Parallel()
		rows := []queryLogRow{
			finishRow("SELECT 1", 10, 1, 1),
			errorRow("SELECT 1", 60),
		}
		buckets := makeBuckets(rows, 0)
		require.Len(t, buckets, 1)

		b := buckets[0]
		assert.InDelta(t, float32(2), b.Common.NumQueries, 1e-6)
		assert.InDelta(t, float32(1), b.Common.NumQueriesWithErrors, 1e-6)
		require.NotNil(t, b.Common.Errors)
		assert.Equal(t, uint64(1), b.Common.Errors[60])
	})

	t.Run("empty input yields no buckets", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, makeBuckets(nil, 0))
	})

	t.Run("written metrics for INSERT", func(t *testing.T) {
		t.Parallel()
		rows := []queryLogRow{insertRow("INSERT INTO t VALUES (1)", 50, 5)}
		buckets := makeBuckets(rows, 0)
		require.Len(t, buckets, 1)
		assert.InDelta(t, float32(5), buckets[0].Clickhouse.MWrittenRowsSum, 1e-6)
	})
}

// finishRow builds a successful QueryFinish Select row for tests.
func finishRow(query string, durationMs, readRows, resultRows uint64) queryLogRow {
	return queryLogRow{
		Type:            queryLogTypeQueryFinish,
		QueryID:         query,
		Query:           query,
		QueryKind:       "Select",
		QueryDurationMs: durationMs,
		ReadRows:        readRows,
		ResultRows:      resultRows,
		Databases:       []string{"default"},
		Tables:          []string{"default.t"},
		User:            "default",
	}
}

// insertRow builds a QueryFinish Insert row for tests.
func insertRow(query string, durationMs, writtenRows uint64) queryLogRow {
	return queryLogRow{
		Type:            queryLogTypeQueryFinish,
		QueryID:         query,
		Query:           query,
		QueryKind:       "Insert",
		QueryDurationMs: durationMs,
		WrittenRows:     writtenRows,
		Databases:       []string{"default"},
		Tables:          []string{"default.t"},
		User:            "default",
	}
}

// errorRow builds an ExceptionWhileProcessing row for tests.
func errorRow(query string, exceptionCode int32) queryLogRow {
	return queryLogRow{
		Type:          queryLogTypeExceptionWhileProcessing,
		QueryID:       query + "-err",
		Query:         query,
		QueryKind:     "Select",
		ExceptionCode: exceptionCode,
		Databases:     []string{"default"},
		Tables:        []string{"default.t"},
		User:          "default",
	}
}
