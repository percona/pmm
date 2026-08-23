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
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// errNoConnection is returned by unreachableDB instead of ever opening a connection.
var errNoConnection = errors.New("no database in this test")

// unreachableConnector hands back a *sqlx.DB whose queries always fail. The sparkline
// point arithmetic runs before any query is issued, so this lets us drive
// SelectSparklines all the way through that arithmetic without a live ClickHouse:
// a panic in it surfaces as a panic, while correct arithmetic surfaces as a query error.
type unreachableConnector struct{}

func (unreachableConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errNoConnection
}

func (unreachableConnector) Driver() driver.Driver { return nil }

func unreachableDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := sqlx.NewDb(sql.OpenDB(unreachableConnector{}), "clickhouse")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// sparklineCtx carries incoming gRPC metadata, which headersToLbacFilter requires.
func sparklineCtx(t *testing.T) context.Context {
	t.Helper()
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("test", t.Name()))
}

// sparklineRanges covers every shape of period the point arithmetic has to survive,
// including the zero-width range from PMM-15160 that made amountOfPoints 0.
func sparklineRanges() []struct {
	name     string
	from, to int64
} {
	const (
		base   = int64(1750322640) // 2026-06-19T08:44:00Z
		minute = int64(60)
		hour   = 60 * minute
	)
	return []struct {
		name     string
		from, to int64
	}{
		// PMM-15160: both endpoints align down to the same minute, so timePeriod == 0.
		{"same minute", base + 10, base + 50},
		{"identical timestamps", base, base},
		{"one minute", base, base + minute},
		{"two minutes", base, base + 2*minute},
		// Just inside and just outside the < 2h branch that reduces the point count.
		{"just under two hours", base, base + 2*hour - minute},
		{"exactly two hours", base, base + 2*hour},
		{"twelve hours", base, base + 12*hour},
		{"thirty days", base, base + 30*24*hour},
		// Reversed range: must not panic either, however the period is treated.
		{"reversed", base + hour, base},
		{"reversed within a minute", base + 50, base + 10},
	}
}

func TestReporterSelectSparklinesNeverPanics(t *testing.T) {
	t.Parallel()

	r := NewReporter(unreachableDB(t))
	for _, tc := range sparklineRanges() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotPanics(t, func() {
				_, _ = r.SelectSparklines(sparklineCtx(t), "queryid", tc.from, tc.to,
					nil, nil, "queryid", "load", false)
			})
		})
	}
}

func TestMetricsSelectSparklinesNeverPanics(t *testing.T) {
	t.Parallel()

	m := NewMetrics(unreachableDB(t))
	for _, tc := range sparklineRanges() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotPanics(t, func() {
				_, _ = m.SelectSparklines(sparklineCtx(t), tc.from, tc.to,
					"", "queryid", nil, nil)
			})
		})
	}
}

// TestSparklinePoints pins the point arithmetic itself. The panic tests above exercise
// the real methods but cannot see their return values, because the query fails first.
func TestSparklinePoints(t *testing.T) {
	t.Parallel()

	const (
		base   = int64(1750322640) // 2026-06-19T08:44:00Z, already minute-aligned
		minute = int64(60)
		hour   = 60 * minute
	)
	for _, tc := range []struct {
		name          string
		from, to      int64
		wantPoints    int64
		wantTimeFrame int64
	}{
		// PMM-15160: a period of less than a minute must still yield one usable point.
		{"identical bounds", base, base, 1, 60},
		{"reversed by a minute", base + minute, base, 1, 60},
		{"reversed by an hour", base + hour, base, 1, 60},
		// Below two hours, one point per minute.
		{"one minute", base, base + minute, 1, 60},
		{"two minutes", base, base + 2*minute, 2, 60},
		{"one hour", base, base + hour, 60, 60},
		{"one minute under two hours", base, base + 2*hour - minute, 119, 60},
		// At and above two hours, capped at the optimal point count.
		{"exactly two hours", base, base + 2*hour, 120, 60},
		{"two hours and a minute", base, base + 2*hour + minute, 121, 60},
		// minutesInPoint > 1 with a non-zero remainder, so the remainder really is divided.
		{"five hours", base, base + 5*hour, 150, 120},
		{"seven hours", base, base + 7*hour, 140, 180},
		{"twelve hours", base, base + 12*hour, 120, 360},
		{"thirty days", base, base + 30*24*hour, 120, 21600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			points, timeFrame := sparklinePoints(tc.from, tc.to)
			require.Equal(t, tc.wantPoints, points, "amountOfPoints")
			require.Equal(t, tc.wantTimeFrame, timeFrame, "timeFrame")
		})
	}
}

// TestSparklinePointsAlwaysUsable is the invariant behind PMM-15160: whatever bounds a
// client sends, the arithmetic must not divide by zero and must describe at least one
// point of at least one minute.
func TestSparklinePointsAlwaysUsable(t *testing.T) {
	t.Parallel()

	for _, from := range []int64{math.MinInt64, math.MinInt64 + 1, -1e9, -3600, -60, -1, 0, 1, 59, 60, 1750322640, math.MaxInt64 - 1, math.MaxInt64} {
		for _, offset := range []int64{math.MinInt64, -1e9, -7200, -60, -1, 0, 1, 59, 60, 61, 7199, 7200, 1e9, math.MaxInt64} {
			to := from + offset // deliberately allowed to overflow
			points, timeFrame := sparklinePoints(from, to)
			require.GreaterOrEqual(t, points, int64(1), "from=%d to=%d", from, to)
			require.GreaterOrEqual(t, timeFrame, int64(60), "from=%d to=%d", from, to)
		}
	}
}

// emptyRowsConnector answers every query with zero rows, so SelectSparklines runs to
// completion and its gap-fill loop emits exactly the points the arithmetic asked for.
// The unreachable connector above cannot do this: it stops the function at the query.
type emptyRowsConnector struct{}

func (emptyRowsConnector) Connect(context.Context) (driver.Conn, error) { return emptyConn{}, nil }
func (emptyRowsConnector) Driver() driver.Driver                        { return nil }

type emptyConn struct{}

func (emptyConn) Prepare(string) (driver.Stmt, error) { return emptyStmt{}, nil }
func (emptyConn) Close() error                        { return nil }
func (emptyConn) Begin() (driver.Tx, error)           { return nil, errNoConnection }

func (emptyConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return emptyRows{}, nil
}

type emptyStmt struct{}

func (emptyStmt) Close() error                               { return nil }
func (emptyStmt) NumInput() int                              { return -1 }
func (emptyStmt) Exec([]driver.Value) (driver.Result, error) { return nil, errNoConnection }
func (emptyStmt) Query([]driver.Value) (driver.Rows, error)  { return emptyRows{}, nil }

type emptyRows struct{}

func (emptyRows) Columns() []string         { return nil }
func (emptyRows) Close() error              { return nil }
func (emptyRows) Next([]driver.Value) error { return io.EOF }

func emptyDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := sqlx.NewDb(sql.OpenDB(emptyRowsConnector{}), "clickhouse")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSparklineWiring checks that each SelectSparklines actually passes its bounds to
// sparklinePoints in the right order and applies the results the right way round: the
// returned series must carry exactly the number of points, and the seconds per point,
// that the helper computed for the same bounds.
func TestSparklineWiring(t *testing.T) {
	t.Parallel()

	const (
		base   = int64(1750322640) // 2026-06-19T08:44:00Z
		minute = int64(60)
		hour   = 60 * minute
	)
	cases := []struct {
		name     string
		from, to int64
	}{
		{"same minute", base + 10, base + 50},
		{"one hour", base, base + hour},
		{"five hours", base, base + 5*hour},
		{"twelve hours", base, base + 12*hour},
	}

	t.Run("reporter", func(t *testing.T) {
		t.Parallel()
		r := NewReporter(emptyDB(t))
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				wantPoints, wantTimeFrame := sparklinePoints(tc.from/60*60, tc.to/60*60)
				points, err := r.SelectSparklines(sparklineCtx(t), "queryid", tc.from, tc.to,
					nil, nil, "queryid", "load", false)
				require.NoError(t, err)
				require.Len(t, points, int(wantPoints))
				require.Equal(t, uint32(wantTimeFrame), points[0].TimeFrame)
			})
		}
	})

	t.Run("metrics", func(t *testing.T) {
		t.Parallel()
		m := NewMetrics(emptyDB(t))
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				wantPoints, wantTimeFrame := sparklinePoints(tc.from/60*60, tc.to/60*60)
				points, err := m.SelectSparklines(sparklineCtx(t), tc.from, tc.to,
					"", "queryid", nil, nil)
				require.NoError(t, err)
				require.Len(t, points, int(wantPoints))
				require.Equal(t, uint32(wantTimeFrame), points[0].TimeFrame)
			})
		}
	})
}
