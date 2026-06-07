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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
)

func TestInsertColsMatchArgs(t *testing.T) {
	t.Parallel()
	// The column list and the values returned by bucketArgs must stay in lockstep.
	require.Len(t, bucketArgs(&qanv1.MetricsBucket{}), len(insertCols))
}

func TestLongTailRouting(t *testing.T) {
	t.Parallel()
	mb := &qanv1.MetricsBucket{
		MQueryTimeSum: 0.5, MQueryTimeCnt: 5, // core -> typed columns, not maps
		MRowsReadSum: 100, MRowsReadCnt: 5, // non-core -> long-tail
		MFullScanSum: 2, MFullScanCnt: 3, // non-core boolean -> long-tail
	}
	sums, cnts := longTail(mb)

	assert.Equal(t, float64(100), sums["rows_read"])
	assert.Equal(t, float64(2), sums["full_scan"])
	assert.Equal(t, uint64(5), cnts["rows_read"])
	assert.Equal(t, uint64(3), cnts["full_scan"])

	// core metrics are stored as columns, never in the long-tail maps
	_, ok := sums["query_time"]
	assert.False(t, ok)
	// zero-valued metrics are omitted (sparse maps)
	_, ok = sums["rows_affected"]
	assert.False(t, ok)
}

func TestIdempotencyKey(t *testing.T) {
	t.Parallel()
	base := &qanv1.MetricsBucket{
		AgentId: "a1", Queryid: "q1", Database: "db", Schema: "s",
		Username: "u", ClientHost: "h", PeriodStartUnixSecs: 1000,
	}
	same := &qanv1.MetricsBucket{
		AgentId: "a1", Queryid: "q1", Database: "db", Schema: "s",
		Username: "u", ClientHost: "h", PeriodStartUnixSecs: 1000,
	}
	require.Equal(t, idempotencyKey(base), idempotencyKey(same))

	other := &qanv1.MetricsBucket{
		AgentId: "a1", Queryid: "q1", Database: "db", Schema: "s",
		Username: "u", ClientHost: "h2", PeriodStartUnixSecs: 1000,
	}
	require.NotEqual(t, idempotencyKey(base), idempotencyKey(other))
}

func TestDedupCache(t *testing.T) {
	t.Parallel()
	c := newDedupCache(dedupTTL)
	require.False(t, c.seenBefore(42))
	require.True(t, c.seenBefore(42))
	require.False(t, c.seenBefore(43))
}
