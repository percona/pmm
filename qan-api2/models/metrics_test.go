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
	"os"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register database/sql driver
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
)

func setupTestClickHouse(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://default:clickhouse@127.0.0.1:19000/pmm_test"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	return db
}

func TestMetrics_Get(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	sqlxDB := setupTestClickHouse(t)
	m := NewMetrics(sqlxDB)

	t1, err := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	require.NoError(t, err)
	periodFrom := t1.Unix()
	t2, err := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	require.NoError(t, err)
	periodTo := t2.Unix()

	t.Run("Get metrics with filter", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, periodFrom, periodTo, "B305F6354FA21F2A", "queryid", nil, nil, false)
		require.NoError(t, err)
		require.Len(t, res, 2) // 1 metric + total
		assert.InDelta(t, float64(340703), res[0]["num_queries"], 0.1)
		assert.InDelta(t, 0.02430000001913868, res[0]["m_query_time_sum"], 1e-9)
	})

	t.Run("Get metrics with dimensions", func(t *testing.T) {
		t.Parallel()
		dimensions := map[string][]string{"service_type": {"service_type1"}}
		res, err := m.Get(ctx, periodFrom, periodTo, "B305F6354FA21F2A", "queryid", dimensions, nil, false)
		require.NoError(t, err)
		require.Len(t, res, 2) // 2 metric
		assert.InDelta(t, float64(340703), res[0]["num_queries"], 0.1)
		assert.InDelta(t, float64(340703), res[1]["num_queries"], 0.1)
	})

	t.Run("Get totals only", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, periodFrom, periodTo, "", "", nil, nil, true)
		require.NoError(t, err)
		require.Len(t, res, 2) // 1 result + total
		assert.InDelta(t, float64(89060995), res[1]["num_queries"], 0.0001)
	})

	t.Run("No metrics with filter", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, periodFrom, periodTo, "absent", "queryid", nil, nil, false)
		require.NoError(t, err)
		require.Len(t, res, 1) // total
		assert.InDelta(t, float64(0), res[0]["num_queries"], 0.1)
	})

	t.Run("Get metrics no period from", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, 0, periodTo, "B305F6354FA21F2A", "queryid", nil, nil, false)
		require.NoError(t, err)
		require.Len(t, res, 2) // 1 metric + total
		assert.InDelta(t, float64(340703), res[0]["num_queries"], 0.1)
		assert.InDelta(t, float64(340703), res[1]["num_queries"], 0.1)
	})

	t.Run("Get metrics no period to", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, periodFrom, 0, "B305F6354FA21F2A", "queryid", nil, nil, false)
		require.NoError(t, err)
		require.Len(t, res, 1) // 1 metric
		assert.InDelta(t, float64(0), res[0]["num_queries"], 0.1)
	})

	t.Run("Invalid group", func(t *testing.T) {
		t.Parallel()
		res, err := m.Get(ctx, periodFrom, periodTo, "B305F6354FA21F2A", "absent_group_name", nil, nil, false)
		require.ErrorContains(t, err, "Unknown expression or function identifier `absent_group_name`")
		require.Nil(t, res)
	})
}

func TestMetrics_SelectQueryExamples(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	sqlxDB := setupTestClickHouse(t)
	m := NewMetrics(sqlxDB)

	periodFrom, err := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	require.NoError(t, err)
	periodTo, err := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	require.NoError(t, err)

	t.Run("Select all query examples", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "B305F6354FA21F2A", "queryid", 1, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.QueryExamples, 1)

		qe := res.QueryExamples[0]
		assert.Equal(t, "schema29", qe.Schema)
		assert.Equal(t, "service_id1", qe.ServiceId)
		assert.Equal(t, "service_type1", qe.ServiceType)
		assert.Empty(t, qe.ExplainFingerprint)
		assert.Zero(t, qe.PlaceholdersCount)
		assert.Equal(t, "SELECT @@GLOBAL.slow_query_log_file", qe.Example)
		assert.Zero(t, qe.IsTruncated)
		assert.Equal(t, qanv1.ExampleType_EXAMPLE_TYPE_RANDOM, qe.ExampleType)
		assert.Empty(t, qe.ExampleMetrics)
	})

	t.Run("Select query examples with filter", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "service_id1", "service_id", 10, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.QueryExamples, 10)
		for _, qe := range res.QueryExamples {
			assert.Equal(t, "service_id1", qe.ServiceId)
			assert.Equal(t, "service_type1", qe.ServiceType)
		}
	})

	t.Run("Select query examples no period from", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, time.Time{}, periodTo, "service_id1", "service_id", 10, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.QueryExamples, 10)
		for _, qe := range res.QueryExamples {
			assert.Equal(t, "service_id1", qe.ServiceId)
			assert.Equal(t, "service_type1", qe.ServiceType)
		}
	})

	t.Run("Select query examples no period to", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, time.Time{}, "service_id1", "service_id", 10, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Nil(t, res.QueryExamples)
	})

	t.Run("Select query examples no group", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "B305F6354FA21F2A", "", 5, nil, nil)
		require.ErrorContains(t, err, "Syntax error: failed at position")
		require.Nil(t, res)
	})

	t.Run("Select query examples no filter", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "", "queryid", 5, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.QueryExamples, 5)
	})

	t.Run("Select query examples absent group", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "", "absent_group", 10, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.QueryExamples, 10)
	})

	t.Run("Select query examples not found", func(t *testing.T) {
		t.Parallel()
		res, err := m.SelectQueryExamples(ctx, periodFrom, periodTo, "absent_id", "queryid", 10, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Nil(t, res.QueryExamples)
	})
}

func TestMetrics_SchemaByQueryID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	sqlxDB := setupTestClickHouse(t)
	m := NewMetrics(sqlxDB)

	t.Run("Get schema for existing query ID and service ID", func(t *testing.T) {
		t.Parallel()
		res, err := m.SchemaByQueryID(ctx, "service_id1", "B305F6354FA21F2A")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "schema12", res.Schema)
	})

	t.Run("Get schema for absent query ID", func(t *testing.T) {
		t.Parallel()
		res, err := m.SchemaByQueryID(ctx, "service1", "non-existent-queryid")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Schema)
	})

	t.Run("Get schema for absent service ID", func(t *testing.T) {
		t.Parallel()
		res, err := m.SchemaByQueryID(ctx, "non-existent-service", "queryid1")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Schema)
	})
}

func TestMetrics_ExplainFingerprintByQueryID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	sqlxDB := setupTestClickHouse(t)
	m := NewMetrics(sqlxDB)

	t.Run("Get fingerprint", func(t *testing.T) {
		t.Parallel()
		res, err := m.ExplainFingerprintByQueryID(ctx, "service_id1", "B305F6354FA21F2A")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "SELECT @@GLOBAL.slow_query_log_file", res.ExplainFingerprint)
		assert.Zero(t, res.PlaceholdersCount)
	})

	t.Run("Get fingerprint with empty explain_fingerprint", func(t *testing.T) {
		t.Parallel()
		res, err := m.ExplainFingerprintByQueryID(ctx, "service_id1", "1D410B4BE5060972")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "Ping", res.ExplainFingerprint)
		assert.Zero(t, res.PlaceholdersCount)
	})

	t.Run("Error for non-existent query ID", func(t *testing.T) {
		t.Parallel()
		res, err := m.ExplainFingerprintByQueryID(ctx, "service_id1", "non-existent-queryid")
		require.ErrorContains(t, err, "query_id")
		require.NotNil(t, res)
		assert.Empty(t, res.ExplainFingerprint)
		assert.Zero(t, res.PlaceholdersCount)
	})
}
