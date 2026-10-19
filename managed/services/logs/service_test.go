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

package logs

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

// passthrough hands query args to sqlmock unchanged; the ClickHouse driver accepts slices and large uint64 values.
type passthrough struct{}

func (passthrough) ConvertValue(v any) (driver.Value, error) {
	return v, nil
}

var lineCols = []string{
	"Timestamp", "node_name", "ServiceName", "preset", "file_name",
	"SeverityNumber", "SeverityText", "body", "body_truncated", "LogAttributes", "PatternHash",
}

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	s, mock, _ := newTestServiceWithInventory(t)
	return s, mock
}

// newTestServiceWithInventory returns the service with mocks of ClickHouse and of the inventory database.
func newTestServiceWithInventory(t *testing.T) (*Service, sqlmock.Sqlmock, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual), sqlmock.ValueConverterOption(passthrough{}))
	require.NoError(t, err)
	invDB, inv, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		assert.NoError(t, inv.ExpectationsWereMet())
		_ = db.Close()
		_ = invDB.Close()
	})
	return New(db, reform.NewDB(invDB, postgresql.Dialect, nil)), mock, inv
}

var labelCols = []string{"cluster", "environment", "replication_set", "node_name"}

func testLogFilter() *logsv1.LogFilter {
	return &logsv1.LogFilter{Start: timestamppb.New(testStart), End: timestamppb.New(testEnd)}
}

func requireCode(t *testing.T, err error, code codes.Code, msgAndArgs ...any) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
	assert.Equal(t, code, status.Code(err), err.Error())
}

func TestSearchLogs(t *testing.T) {
	s, mock := newTestService(t)
	f := Filter{Start: testStart, End: testEnd}
	t1 := testStart.Add(2 * time.Minute)
	t2 := testStart.Add(time.Minute)
	rows := func() *sqlmock.Rows {
		return mock.NewRows(append(lineCols, "h")).
			AddRow(t1, "pmm-server", "pmm-managed", "pmm_managed", "pmm-managed.log", int32(17), "error", "boom", true,
				map[string]string{"component": "grpc"}, uint64(31), uint64(18446744073709551615)).
			AddRow(t2, "pmm-server", "pmm-managed", "pmm_managed", "pmm-managed.log", int32(9), "info", "ok", false,
				map[string]string{}, uint64(32), uint64(7))
	}

	q, _ := searchQuery(f, nil, 2)
	mock.ExpectQuery(q).WillReturnRows(rows())
	res, err := s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter(), Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Lines, 2)
	assert.Equal(t, &logsv1.LogLine{
		Time:           timestamppb.New(t1),
		NodeName:       "pmm-server",
		Source:         "pmm-managed",
		Preset:         "pmm_managed",
		FileName:       "pmm-managed.log",
		SeverityNumber: 17,
		SeverityText:   "error",
		Body:           "boom",
		BodyTruncated:  true,
		Attributes:     map[string]string{"component": "grpc"},
		PatternHash:    31,
	}, res.Lines[0])
	assert.Equal(t, Cursor{Time: t2, Hash: 7, Skip: 1}.Encode(), res.NextCursor)

	after := Cursor{Time: t2, Hash: 7, Skip: 1}
	q, _ = searchQuery(f, &after, 3)
	mock.ExpectQuery(q).WillReturnRows(rows())
	res, err = s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter(), Cursor: after.Encode(), Limit: 3})
	require.NoError(t, err)
	assert.Len(t, res.Lines, 2)
	assert.Empty(t, res.NextCursor)

	_, err = s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter(), Cursor: "bad"})
	requireCode(t, err, codes.InvalidArgument)
}

func TestSearchLogs_IdenticalLines(t *testing.T) {
	s, mock := newTestService(t)
	f := Filter{Start: testStart, End: testEnd}
	t1, t2 := testStart.Add(2*time.Minute), testStart.Add(time.Minute)
	row := func(rows *sqlmock.Rows, ts time.Time, h uint64) *sqlmock.Rows {
		return rows.AddRow(ts, "n", "s", "postgres", "f", int32(9), "LOG", "statement: COMMIT", false, map[string]string{}, uint64(1), h)
	}

	// The page ends with two identical lines: the cursor skips both.
	q, _ := searchQuery(f, nil, 3)
	mock.ExpectQuery(q).WillReturnRows(row(row(row(mock.NewRows(append(lineCols, "h")), t1, 5), t2, 7), t2, 7))
	res, err := s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter(), Limit: 3})
	require.NoError(t, err)
	assert.Equal(t, Cursor{Time: t2, Hash: 7, Skip: 2}.Encode(), res.NextCursor)

	// A page made only of more such lines adds to the skip.
	after := Cursor{Time: t2, Hash: 7, Skip: 2}
	q, _ = searchQuery(f, &after, 3)
	mock.ExpectQuery(q).WillReturnRows(row(row(row(mock.NewRows(append(lineCols, "h")), t2, 7), t2, 7), t2, 7))
	res, err = s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter(), Cursor: after.Encode(), Limit: 3})
	require.NoError(t, err)
	assert.Equal(t, Cursor{Time: t2, Hash: 7, Skip: 5}.Encode(), res.NextCursor)
}

func TestSearchLogs_Timeout(t *testing.T) {
	s, mock := newTestService(t)
	q, _ := searchQuery(Filter{Start: testStart, End: testEnd}, nil, 0)
	mock.ExpectQuery(q).WillReturnError(errors.New("code: 159, message: Timeout exceeded: elapsed 30.001 seconds, maximum: 30. (TIMEOUT_EXCEEDED)"))
	_, err := s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: testLogFilter()})
	requireCode(t, err, codes.DeadlineExceeded)
	assert.Equal(t, "The query took longer than 30 s; narrow the time range or add filters.", status.Convert(err).Message())
}

func TestGetLogVolume(t *testing.T) {
	s, mock := newTestService(t)
	q, _, _ := volumeQuery(Filter{Start: testStart, End: testEnd})
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows([]string{"t", "error", "warning", "info", "other"}).
		AddRow(testStart, uint64(1), uint64(2), uint64(3), uint64(4)).
		AddRow(testStart.Add(time.Minute), uint64(0), uint64(0), uint64(5), uint64(0)))
	res, err := s.GetLogVolume(t.Context(), &logsv1.GetLogVolumeRequest{Filter: testLogFilter()})
	require.NoError(t, err)
	assert.Equal(t, durationpb.New(time.Minute), res.Step)
	assert.Equal(t, []*logsv1.VolumeBucket{
		{Start: timestamppb.New(testStart), Error: 1, Warning: 2, Info: 3, Other: 4},
		{Start: timestamppb.New(testStart.Add(time.Minute)), Info: 5},
	}, res.Buckets)
}

func TestGetLogPatterns(t *testing.T) {
	s, mock := newTestService(t)
	q, _ := patternsQuery(Filter{Start: testStart, End: testEnd, PatternHash: 18446744073709551615, ExcludePatternHashes: []uint64{5}})
	first, last := testStart.Add(time.Minute), testStart.Add(2*time.Minute)
	cols := []string{"p", "PatternHash", "current", "previous", "is_new", "max_severity", "first_seen", "last_seen", "sample", "top_sources"}
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows(cols).
		AddRow("deadlock detected", uint64(18446744073709551615), uint64(12), uint64(0), true, int32(17), first, last, "deadlock detected",
			[]string{"pg1", "pg2"}).
		AddRow("checkpoint starting: ?", uint64(8), uint64(3), uint64(4), false, int32(9), first, last, "checkpoint starting: time",
			[]string{}))
	f := testLogFilter()
	f.PatternHash = 18446744073709551615
	f.ExcludePatternHashes = []uint64{5}
	res, err := s.GetLogPatterns(t.Context(), &logsv1.GetLogPatternsRequest{Filter: f})
	require.NoError(t, err)
	require.Len(t, res.Patterns, 2)
	assert.Equal(t, &logsv1.LogPattern{
		Pattern:           "deadlock detected",
		Count:             12,
		MaxSeverityNumber: 17,
		FirstSeen:         timestamppb.New(first),
		LastSeen:          timestamppb.New(last),
		Sample:            "deadlock detected",
		IsNew:             true,
		PatternHash:       18446744073709551615,
		TopSources:        []string{"pg1", "pg2"},
	}, res.Patterns[0])
	assert.False(t, res.Patterns[1].IsNew)
	assert.Equal(t, int64(4), res.Patterns[1].PreviousCount)
	assert.Empty(t, res.Patterns[1].TopSources)
}

func TestGetLogContext(t *testing.T) {
	s, mock := newTestService(t)
	at := testStart.Add(30 * time.Minute)
	q, _ := contextQuery(at, "node1", "postgresql.log", "node1", 50)
	rows := mock.NewRows(lineCols)
	for i, ts := range []time.Time{at.Add(-time.Second), at, at, at.Add(time.Second)} {
		rows.AddRow(ts, "node1", "node1", "postgres", "postgresql.log", int32(9), "LOG", string(rune('a'+i)), false, map[string]string{}, uint64(1))
	}
	mock.ExpectQuery(q).WillReturnRows(rows)
	res, err := s.GetLogContext(t.Context(), &logsv1.GetLogContextRequest{
		Time:     timestamppb.New(at),
		NodeName: "node1",
		FileName: "postgresql.log",
		Source:   "node1",
	})
	require.NoError(t, err)
	require.Len(t, res.Lines, 4)
	assert.Equal(t, int32(1), res.AnchorIndex, "first line at the requested time")

	// No line at the requested time: the last one before it.
	q, _ = contextQuery(at, "node1", "postgresql.log", "", 50)
	rows = mock.NewRows(lineCols)
	for _, ts := range []time.Time{at.Add(-2 * time.Second), at.Add(-time.Second), at.Add(time.Second)} {
		rows.AddRow(ts, "node1", "node1", "postgres", "postgresql.log", int32(9), "LOG", "x", false, map[string]string{}, uint64(1))
	}
	mock.ExpectQuery(q).WillReturnRows(rows)
	res, err = s.GetLogContext(t.Context(), &logsv1.GetLogContextRequest{Time: timestamppb.New(at), NodeName: "node1", FileName: "postgresql.log"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.AnchorIndex)
}

func TestGetLogFacets(t *testing.T) {
	s, mock, inv := newTestServiceWithInventory(t)
	f := Filter{Start: testStart, End: testEnd}
	nodes, sources, presets, _ := facetsQueries(f)
	cols := []string{"v", "count()"}
	mock.ExpectQuery(nodes).WillReturnRows(mock.NewRows(cols).AddRow("pmm-server", uint64(10)).AddRow("node1", uint64(5)))
	mock.ExpectQuery(sources).WillReturnRows(mock.NewRows(cols).AddRow("pmm-managed", uint64(10)))
	mock.ExpectQuery(presets).WillReturnRows(mock.NewRows(cols).AddRow("postgres", uint64(5)))
	inv.ExpectQuery(labelsQuery).WillReturnRows(inv.NewRows(labelCols).
		AddRow("prod-pg", "prod", "", "node1").
		AddRow("prod-pg", "prod", "", "node2"))
	counts, _ := nodeCountsQuery(f, []string{"node1", "node2"})
	mock.ExpectQuery(counts).WillReturnRows(mock.NewRows(cols).AddRow("node1", uint64(5)))
	res, err := s.GetLogFacets(t.Context(), &logsv1.GetLogFacetsRequest{Filter: testLogFilter()})
	require.NoError(t, err)
	assert.Equal(t, []*logsv1.Facet{{Value: "pmm-server", Count: 10}, {Value: "node1", Count: 5}}, res.NodeNames)
	assert.Equal(t, []*logsv1.Facet{{Value: "pmm-managed", Count: 10}}, res.Sources)
	assert.Equal(t, []*logsv1.Facet{{Value: "postgres", Count: 5}}, res.Presets)
	assert.Equal(t, []*logsv1.Facet{{Value: "prod-pg", Count: 5}}, res.Clusters)
	assert.Equal(t, []*logsv1.Facet{{Value: "prod", Count: 5}}, res.Environments)
	assert.Empty(t, res.ReplicationSets)
}

func TestGetLogFacets_NoLabels(t *testing.T) {
	s, mock, inv := newTestServiceWithInventory(t)
	nodes, sources, presets, _ := facetsQueries(Filter{Start: testStart, End: testEnd})
	cols := []string{"v", "count()"}
	for _, q := range []string{nodes, sources, presets} {
		mock.ExpectQuery(q).WillReturnRows(mock.NewRows(cols))
	}
	// Without labeled services ClickHouse is not asked for node counts.
	inv.ExpectQuery(labelsQuery).WillReturnRows(inv.NewRows(labelCols))
	res, err := s.GetLogFacets(t.Context(), &logsv1.GetLogFacetsRequest{Filter: testLogFilter()})
	require.NoError(t, err)
	assert.Empty(t, res.Clusters)
}

func TestSearchLogs_Labels(t *testing.T) {
	s, mock, inv := newTestServiceWithInventory(t)
	inv.ExpectQuery(labelsQuery).WillReturnRows(inv.NewRows(labelCols).
		AddRow("sharded", "", "rs1", "rs101").
		AddRow("ps", "", "", "ps1"))
	q, _ := searchQuery(Filter{Start: testStart, End: testEnd, ByLabels: true, LabelNodes: []string{"rs101"}}, nil, 0)
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows(append(lineCols, "h")))
	f := testLogFilter()
	f.Clusters = []string{"sharded"}
	_, err := s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: f})
	require.NoError(t, err)

	inv.ExpectQuery(labelsQuery).WillReturnError(errors.New("connection refused"))
	_, err = s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: f})
	requireCode(t, err, codes.Internal)
}

func TestLBACDenied(t *testing.T) {
	s, _ := newTestService(t)
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-proxy-filter", "abc"))
	_, err := s.SearchLogs(ctx, &logsv1.SearchLogsRequest{Filter: testLogFilter()})
	requireCode(t, err, codes.PermissionDenied)
	assert.Equal(t, "Logs are not available to users with label-based access filters.", status.Convert(err).Message())
	_, err = s.GetLogVolume(ctx, &logsv1.GetLogVolumeRequest{Filter: testLogFilter()})
	requireCode(t, err, codes.PermissionDenied)
	_, err = s.GetLogPatterns(ctx, &logsv1.GetLogPatternsRequest{Filter: testLogFilter()})
	requireCode(t, err, codes.PermissionDenied)
	_, err = s.GetLogContext(ctx, &logsv1.GetLogContextRequest{Time: timestamppb.New(testStart), FileName: "f"})
	requireCode(t, err, codes.PermissionDenied)
	_, err = s.GetLogFacets(ctx, &logsv1.GetLogFacetsRequest{Filter: testLogFilter()})
	requireCode(t, err, codes.PermissionDenied)
}

func TestBadRange(t *testing.T) {
	s, _ := newTestService(t)
	for name, f := range map[string]*logsv1.LogFilter{
		"reversed": {Start: timestamppb.New(testEnd), End: timestamppb.New(testStart)},
		// Longer than 100 daily volume buckets.
		"too long": {Start: timestamppb.New(testEnd.Add(-100 * 24 * time.Hour)), End: timestamppb.New(testEnd)},
		"severity bounds reversed": {
			Start: timestamppb.New(testStart), End: timestamppb.New(testEnd), MinSeverity: 17, MaxSeverity: 16,
		},
	} {
		_, err := s.SearchLogs(t.Context(), &logsv1.SearchLogsRequest{Filter: f})
		requireCode(t, err, codes.InvalidArgument)
		_, err = s.GetLogVolume(t.Context(), &logsv1.GetLogVolumeRequest{Filter: f})
		requireCode(t, err, codes.InvalidArgument)
		_, err = s.GetLogPatterns(t.Context(), &logsv1.GetLogPatternsRequest{Filter: f})
		requireCode(t, err, codes.InvalidArgument)
		_, err = s.GetLogFacets(t.Context(), &logsv1.GetLogFacetsRequest{Filter: f})
		requireCode(t, err, codes.InvalidArgument, name)
	}
}
