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
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestMetricsLBAC pins that every query-details read path applies the caller's LBAC
// filter, so a queryid recorded only on hidden services reads as absent; see PMM-15697.
func TestMetricsLBAC(t *testing.T) {
	t.Parallel()
	db := setupTestClickHouse(t)
	// The same mapper as qan-api2's own connection, which GetQueryPlanResponse needs.
	db.Mapper = reflectx.NewMapperTagFunc("json", strings.ToUpper, func(value string) string {
		return strings.Split(value, ",")[0]
	})
	m := NewMetrics(db)

	from, err := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	require.NoError(t, err)
	to, err := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	require.NoError(t, err)
	const queryID = "B305F6354FA21F2A"

	// read runs every query-details read path and returns what each one saw.
	read := func(t *testing.T, ctx context.Context) map[string]any {
		t.Helper()
		got := make(map[string]any)

		rows, err := m.Get(ctx, from.Unix(), to.Unix(), queryID, "queryid", nil, nil, false)
		require.NoError(t, err)
		// WITH TOTALS always adds a row, so only the data rows are compared.
		got["Get"] = len(rows) - 1

		sparklines, err := m.SelectSparklines(ctx, from.Unix(), to.Unix(), queryID, "queryid", nil, nil)
		require.NoError(t, err)
		var load float32
		for _, p := range sparklines {
			load += p.Load
		}
		got["SelectSparklines"] = load

		examples, err := m.SelectQueryExamples(ctx, from, to, queryID, "queryid", 10, nil, nil)
		require.NoError(t, err)
		got["SelectQueryExamples"] = len(examples.QueryExamples)

		labels, err := m.SelectObjectDetailsLabels(ctx, from, to, queryID, "queryid")
		require.NoError(t, err)
		got["SelectObjectDetailsLabels"] = len(labels.Labels["service_name"].GetValues()) + len(labels.Labels["label7"].GetValues())

		histogram, err := m.SelectHistogram(ctx, from.Unix(), to.Unix(), nil, nil, queryID)
		require.NoError(t, err)
		got["SelectHistogram"] = len(histogram.HistogramItems)

		metadata, err := m.GetSelectedQueryMetadata(ctx, from.Unix(), to.Unix(), queryID, "queryid", nil, nil, false)
		require.NoError(t, err)
		got["GetSelectedQueryMetadata"] = metadata.ServiceName

		fingerprint, err := m.GetFingerprintByQueryID(ctx, queryID)
		require.NoError(t, err)
		got["GetFingerprintByQueryID"] = fingerprint

		plan, err := m.SelectQueryPlan(ctx, queryID, "service_id1")
		require.NoError(t, err)
		got["SelectQueryPlan"] = plan.Planid + plan.QueryPlan

		schema, err := m.SchemaByQueryID(ctx, "service_id1", queryID)
		require.NoError(t, err)
		got["SchemaByQueryID"] = schema.Schema

		explain, err := m.ExplainFingerprintByQueryID(ctx, "service_id1", queryID)
		if err != nil {
			require.ErrorContains(t, err, "query_id")
		}
		got["ExplainFingerprintByQueryID"] = explain.ExplainFingerprint
		return got
	}

	unfiltered := read(t, metadata.NewIncomingContext(t.Context(), metadata.MD{}))
	for path, v := range unfiltered {
		// The fixture holds no plans or histograms: those paths only prove their SQL runs.
		if path != "SelectQueryPlan" && path != "SelectHistogram" {
			require.NotZero(t, v, "the fixture must exercise %s", path)
		}
	}

	// Each filter matches every row of the queryid, some of them, or none.
	const (
		all = iota
		some
		none
	)
	for name, tc := range map[string]struct {
		filter  string
		matches int
	}{
		"Allowed":           {filter: `["{service_id=\"service_id1\"}"]`, matches: all},
		"AnyOfTwoSelectors": {filter: `["{service_id=\"other\"}", "{service_id=\"service_id1\"}"]`, matches: all},
		"AllowedByLabel":    {filter: `["{label7=~\"value.*\"}"]`, matches: some},
		"Denied":            {filter: `["{service_id=\"other\"}"]`, matches: none},
		"DeniedByLabel":     {filter: `["{label7=\"no-such-value\"}"]`, matches: none},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := read(t, setup(t, tc.filter))
			for path, v := range got {
				switch {
				case tc.matches == none:
					assert.Zero(t, v, path)
				case tc.matches == all && path != "SchemaByQueryID":
					assert.Equal(t, unfiltered[path], v, path)
				case !reflect.ValueOf(unfiltered[path]).IsZero():
					// SchemaByQueryID reads one row of many, so it is only checked for presence.
					assert.NotZero(t, v, path)
				}
			}
		})
	}

	t.Run("PlanIsScopedToTheService", func(t *testing.T) {
		t.Parallel()
		plan, err := m.SelectQueryPlan(metadata.NewIncomingContext(t.Context(), metadata.MD{}), queryID, "other")
		require.NoError(t, err)
		assert.Empty(t, plan.Planid+plan.QueryPlan)
	})
}
