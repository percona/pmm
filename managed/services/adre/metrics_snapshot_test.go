// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

type mockVMAPI struct {
	matrix model.Matrix
}

func (m *mockVMAPI) QueryRange(_ context.Context, _ string, _ v1.Range, _ ...v1.Option) (model.Value, v1.Warnings, error) {
	return m.matrix, nil, nil
}

func (m *mockVMAPI) Alerts(context.Context) (v1.AlertsResult, error) { panic("unexpected Alerts") }
func (m *mockVMAPI) AlertManagers(context.Context) (v1.AlertManagersResult, error) {
	panic("unexpected AlertManagers")
}

func (m *mockVMAPI) CleanTombstones(context.Context) error           { panic("unexpected CleanTombstones") }
func (m *mockVMAPI) Config(context.Context) (v1.ConfigResult, error) { panic("unexpected Config") }
func (m *mockVMAPI) DeleteSeries(context.Context, []string, time.Time, time.Time) error {
	panic("unexpected DeleteSeries")
}
func (m *mockVMAPI) Flags(context.Context) (v1.FlagsResult, error) { panic("unexpected Flags") }
func (m *mockVMAPI) LabelNames(context.Context, []string, time.Time, time.Time, ...v1.Option) ([]string, v1.Warnings, error) {
	panic("unexpected LabelNames")
}

func (m *mockVMAPI) LabelValues(context.Context, string, []string, time.Time, time.Time, ...v1.Option) (model.LabelValues, v1.Warnings, error) {
	panic("unexpected LabelValues")
}

func (m *mockVMAPI) Query(context.Context, string, time.Time, ...v1.Option) (model.Value, v1.Warnings, error) {
	panic("unexpected Query")
}

func (m *mockVMAPI) QueryExemplars(context.Context, string, time.Time, time.Time) ([]v1.ExemplarQueryResult, error) {
	panic("unexpected QueryExemplars")
}

func (m *mockVMAPI) Buildinfo(context.Context) (v1.BuildinfoResult, error) {
	panic("unexpected Buildinfo")
}

func (m *mockVMAPI) Runtimeinfo(context.Context) (v1.RuntimeinfoResult, error) {
	panic("unexpected Runtimeinfo")
}

func (m *mockVMAPI) Series(context.Context, []string, time.Time, time.Time, ...v1.Option) ([]model.LabelSet, v1.Warnings, error) {
	panic("unexpected Series")
}

func (m *mockVMAPI) Snapshot(context.Context, bool) (v1.SnapshotResult, error) {
	panic("unexpected Snapshot")
}
func (m *mockVMAPI) Rules(context.Context) (v1.RulesResult, error)     { panic("unexpected Rules") }
func (m *mockVMAPI) Targets(context.Context) (v1.TargetsResult, error) { panic("unexpected Targets") }

func (m *mockVMAPI) TargetsMetadata(context.Context, string, string, string) ([]v1.MetricMetadata, error) {
	panic("unexpected TargetsMetadata")
}

func (m *mockVMAPI) Metadata(context.Context, string, string) (map[string][]v1.Metadata, error) {
	panic("unexpected Metadata")
}

func (m *mockVMAPI) TSDB(context.Context, ...v1.Option) (v1.TSDBResult, error) {
	panic("unexpected TSDB")
}

func (m *mockVMAPI) WalReplay(context.Context) (v1.WalReplayStatus, error) {
	panic("unexpected WalReplay")
}

func TestParseSnapshotBounds(t *testing.T) {
	t.Parallel()

	t.Run("RFC3339", func(t *testing.T) {
		t.Parallel()
		start, end, err := parseSnapshotBounds("2026-05-24T00:00:00Z", "2026-05-24T06:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC), start)
		assert.Equal(t, time.Date(2026, 5, 24, 6, 0, 0, 0, time.UTC), end)
	})

	t.Run("NegativeStartRelativeToEnd", func(t *testing.T) {
		t.Parallel()
		start, end, err := parseSnapshotBounds("-3600", "2026-05-24T06:00:00Z")
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 5, 24, 5, 0, 0, 0, time.UTC), start)
		assert.Equal(t, time.Date(2026, 5, 24, 6, 0, 0, 0, time.UTC), end)
	})

	t.Run("MissingStart", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseSnapshotBounds("", "2026-05-24T06:00:00Z")
		require.Error(t, err)
		assert.Contains(t, err.Error(), snapshotTimeFormatHint)
	})

	t.Run("InvalidStart", func(t *testing.T) {
		t.Parallel()
		_, _, err := parseSnapshotBounds("not-a-time", "2026-05-24T06:00:00Z")
		require.Error(t, err)
		assert.Contains(t, err.Error(), snapshotTimeFormatHint)
	})
}

func TestSampleStreamPointsTruncated(t *testing.T) {
	t.Parallel()

	values := make([]model.SamplePair, 600)
	for i := range values {
		values[i] = model.SamplePair{
			Timestamp: model.TimeFromUnix(int64(i)),
			Value:     model.SampleValue(i),
		}
	}
	_, floats, truncated := sampleStreamPoints(&model.SampleStream{Values: values}, 500)
	assert.True(t, truncated)
	assert.Len(t, floats, 500)
	assert.InDelta(t, 100.0, floats[0], 1e-9) // last 500 of 0..599
}

func TestFindAnomaliesCapped(t *testing.T) {
	t.Parallel()

	ts := make([]time.Time, 30)
	vals := make([]float64, 30)
	for i := range vals {
		ts[i] = time.Date(2026, 5, 24, 10, i, 0, 0, time.UTC)
		vals[i] = 10
	}
	for i := 20; i < 30; i++ {
		vals[i] = float64(i * 100)
	}
	anomalies := findAnomaliesTopN(ts, vals, 1.0, defaultMaxAnomalies)
	assert.LessOrEqual(t, len(anomalies), defaultMaxAnomalies)
	assert.NotEmpty(t, anomalies)
}

func TestPostMetricsSnapshot_HandlerErrors(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("MethodNotAllowed", func(t *testing.T) {
		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
		req := httptest.NewRequest(http.MethodGet, "/v1/adre/metrics/snapshot", nil)
		rec := httptest.NewRecorder()
		h.PostMetricsSnapshot(rec, req)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		var body map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, "method not allowed", body["error"])
	})

	t.Run("AdreDisabled", func(t *testing.T) {
		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, &mockVMAPI{}, ClickHousePools{})
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/metrics/snapshot", bytes.NewReader([]byte(`{"query":"up","start":"2026-05-24T00:00:00Z"}`)))
		rec := httptest.NewRecorder()
		h.PostMetricsSnapshot(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("VMNotConfigured", func(t *testing.T) {
		_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
			EnableAdre: new(true),
			AdreURL:    new("http://holmes:8080"),
		})
		require.NoError(t, err)

		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/metrics/snapshot", bytes.NewReader([]byte(`{"query":"up","start":"2026-05-24T00:00:00Z"}`)))
		rec := httptest.NewRecorder()
		h.PostMetricsSnapshot(rec, req)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}

func TestPostMetricsSnapshot_Success(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EnableAdre: new(true),
		AdreURL:    new("http://holmes:8080"),
	})
	require.NoError(t, err)

	start := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	matrix := model.Matrix{
		&model.SampleStream{
			Metric: model.Metric{"__name__": "up"},
			Values: []model.SamplePair{
				{Timestamp: model.TimeFromUnix(start.Unix()), Value: 1},
				{Timestamp: model.TimeFromUnix(start.Add(time.Hour).Unix()), Value: 1},
			},
		},
	}
	h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, &mockVMAPI{matrix: matrix}, ClickHousePools{})

	body := `{"query":"up","start":"2026-05-24T00:00:00Z","end":"2026-05-24T06:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/adre/metrics/snapshot", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	h.PostMetricsSnapshot(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp metricsSnapshotResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Series, 1)
	assert.Equal(t, 1.0, resp.Series[0].Stats.Min) //nolint:testifylint
	assert.False(t, resp.Truncated)
}
