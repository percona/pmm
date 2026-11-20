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

// errNoConnection is returned instead of ever opening a real connection.
var errNoConnection = errors.New("no database in this test")

// sparklineCtx carries incoming gRPC metadata, which headersToLbacFilter requires.
func sparklineCtx(t *testing.T) context.Context {
	t.Helper()
	return metadata.NewIncomingContext(t.Context(), metadata.Pairs("test", t.Name()))
}

// TestSparklinePoints pins the point arithmetic itself. TestSparklineWiring below pins
// the two real methods to it.
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
			layout := newSparklineLayout(tc.from, tc.to)
			require.Equal(t, tc.wantPoints, layout.amountOfPoints, "amountOfPoints")
			require.Equal(t, tc.wantTimeFrame, layout.timeFrame, "timeFrame")
		})
	}
}

// TestSparklineLayoutAlignsItsBounds pins the alignment that both callers used to do for
// themselves. The layout always describes a whole-minute window, and an unaligned request
// is treated exactly like the aligned one it falls inside -- so a caller that forgets to
// align cannot reach a different answer, and the two that used to align get the same
// window they got before.
func TestSparklineLayoutAlignsItsBounds(t *testing.T) {
	t.Parallel()

	const base = int64(1750322640) // 2026-06-19T08:44:00Z, already minute-aligned

	t.Run("bounds land on a whole minute", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name             string
			from, to         int64
			wantFrom, wantTo int64
		}{
			{"already aligned", base, base + 3600, base, base + 3600},
			{"both inside one minute", base + 10, base + 50, base, base},
			{"to spans into the next minute", base, base + 119, base, base + 60},
			{"reversed", base + 50, base + 10, base, base},
			// Integer division truncates towards zero, so a pre-1970 bound lands on the
			// minute above rather than below. Preserved from the arithmetic the callers
			// used to do inline; no real QAN request has a negative period.
			{"negative bounds", -119, -61, -60, -60},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				layout := newSparklineLayout(tc.from, tc.to)
				require.Equal(t, tc.wantFrom, layout.periodStartFromSec, "periodStartFromSec")
				require.Equal(t, tc.wantTo, layout.periodStartToSec, "periodStartToSec")
				require.Zero(t, layout.periodStartFromSec%60, "from not on a minute")
				require.Zero(t, layout.periodStartToSec%60, "to not on a minute")
			})
		}
	})

	// Aligning before the call must make no difference, which is what lets the callers
	// drop their own alignment without changing the window they query.
	t.Run("aligning first changes nothing", func(t *testing.T) {
		t.Parallel()
		for _, from := range []int64{-3601, -60, -1, 0, 1, 59, base, base + 10, base + 59} {
			for _, offset := range []int64{-3600, -61, -1, 0, 1, 59, 60, 61, 7199, 7200, 43200} {
				to := from + offset
				require.Equal(t,
					newSparklineLayout(from/60*60, to/60*60),
					newSparklineLayout(from, to),
					"from=%d to=%d", from, to)
			}
		}
	})
}

// TestSparklinePointsAlwaysUsable is the invariant behind PMM-15160: whatever bounds a
// client sends -- aligned to a minute or not -- the arithmetic must not divide by zero and
// must describe at least one point of at least one minute.
func TestSparklinePointsAlwaysUsable(t *testing.T) {
	t.Parallel()

	for _, from := range []int64{math.MinInt64, math.MinInt64 + 1, -1e9, -3600, -60, -1, 0, 1, 59, 60, 1750322640, math.MaxInt64 - 1, math.MaxInt64} {
		for _, offset := range []int64{math.MinInt64, -1e9, -7200, -60, -1, 0, 1, 59, 60, 61, 7199, 7200, 1e9, math.MaxInt64} {
			to := from + offset // deliberately allowed to overflow
			layout := newSparklineLayout(from, to)
			require.GreaterOrEqual(t, layout.amountOfPoints, int64(1), "from=%d to=%d", from, to)
			require.GreaterOrEqual(t, layout.timeFrame, int64(60), "from=%d to=%d", from, to)
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
		{"reversed within a minute", base + 50, base + 10},
		{"reversed by an hour", base + hour, base},
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
				want := newSparklineLayout(tc.from, tc.to)
				points, err := r.SelectSparklines(sparklineCtx(t), "queryid", tc.from, tc.to,
					nil, nil, "queryid", "load", false)
				require.NoError(t, err)
				require.Len(t, points, int(want.amountOfPoints))
				require.Equal(t, uint32(want.timeFrame), points[0].TimeFrame)
			})
		}
	})

	t.Run("metrics", func(t *testing.T) {
		t.Parallel()
		m := NewMetrics(emptyDB(t))
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				want := newSparklineLayout(tc.from, tc.to)
				points, err := m.SelectSparklines(sparklineCtx(t), tc.from, tc.to,
					"", "queryid", nil, nil)
				require.NoError(t, err)
				require.Len(t, points, int(want.amountOfPoints))
				require.Equal(t, uint32(want.timeFrame), points[0].TimeFrame)
			})
		}
	})
}
