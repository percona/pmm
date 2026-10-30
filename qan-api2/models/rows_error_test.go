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
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/percona/pmm/qan-api2/utils/logger"
)

// errMidStream stands in for ClickHouse dropping out part-way through streaming a result
// set -- the case database/sql reports through Rows.Err() rather than through the error
// returned by the query call.
var errMidStream = errors.New("clickhouse went away mid-stream")

type failingConnector struct{}

func (failingConnector) Connect(context.Context) (driver.Conn, error) { return failingConn{}, nil }
func (failingConnector) Driver() driver.Driver                        { return nil }

type failingConn struct{}

func (failingConn) Prepare(string) (driver.Stmt, error) { return failingStmt{}, nil }
func (failingConn) Close() error                        { return nil }
func (failingConn) Begin() (driver.Tx, error)           { return nil, errMidStream }

func (failingConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return failingRows{}, nil
}

type failingStmt struct{}

func (failingStmt) Close() error                               { return nil }
func (failingStmt) NumInput() int                              { return -1 }
func (failingStmt) Exec([]driver.Value) (driver.Result, error) { return nil, errMidStream }
func (failingStmt) Query([]driver.Value) (driver.Rows, error)  { return failingRows{}, nil }

// failingRows opens successfully and then fails while being iterated, which is exactly
// how a mid-stream server failure surfaces: the query call returns no error at all.
type failingRows struct{}

func (failingRows) Columns() []string         { return nil }
func (failingRows) Close() error              { return nil }
func (failingRows) Next([]driver.Value) error { return errMidStream }

// rowsErrCtx carries both the incoming gRPC metadata headersToLbacFilter requires and the
// logrus entry Reporter.Select fetches with logger.Get.
func rowsErrCtx(t *testing.T) context.Context {
	t.Helper()
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-request-id", t.Name()))
	return logger.SetEntry(ctx, logrus.WithField("test", t.Name()))
}

func failingDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := sqlx.NewDb(sql.OpenDB(failingConnector{}), "clickhouse")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestRowsErrorIsReported pins the checks added for the mid-stream failure case. Without
// them each of these methods returns the error it got from the query call -- nil -- so a
// truncated result set is reported as success. For the two sparkline methods that is the
// worst of the set: the gap-fill loop then invents an empty point for every row that never
// arrived, producing a flat-zero sparkline indistinguishable from real data.
func TestRowsErrorIsReported(t *testing.T) {
	t.Parallel()

	t.Run("Reporter.SelectSparklines", func(t *testing.T) {
		t.Parallel()
		r := NewReporter(failingDB(t))
		points, err := r.SelectSparklines(rowsErrCtx(t), "queryid",
			1750322640, 1750322640+3600, nil, nil, "queryid", "load", false)
		require.ErrorIs(t, err, errMidStream)
		require.Nil(t, points, "a failed stream must not be gap-filled into a flat series")
	})

	t.Run("Metrics.SelectSparklines", func(t *testing.T) {
		t.Parallel()
		m := NewMetrics(failingDB(t))
		points, err := m.SelectSparklines(rowsErrCtx(t),
			1750322640, 1750322640+3600, "", "queryid", nil, nil)
		require.ErrorIs(t, err, errMidStream)
		require.Nil(t, points, "a failed stream must not be gap-filled into a flat series")
	})

	t.Run("Reporter.Select", func(t *testing.T) {
		t.Parallel()
		r := NewReporter(failingDB(t))
		rows, err := r.Select(rowsErrCtx(t), 1750322640, 1750322640+3600,
			nil, nil, "queryid", "load", "", 0, 10,
			[]string{"load"}, []string{"query_time"}, nil)
		require.ErrorIs(t, err, errMidStream)
		require.Nil(t, rows)
	})

	t.Run("Metrics.Get", func(t *testing.T) {
		t.Parallel()
		m := NewMetrics(failingDB(t))
		rows, err := m.Get(rowsErrCtx(t), 1750322640, 1750322640+3600,
			"", "queryid", nil, nil, false)
		require.ErrorIs(t, err, errMidStream)
		require.Nil(t, rows)
	})

	t.Run("Metrics.SelectHistogram", func(t *testing.T) {
		t.Parallel()
		m := NewMetrics(failingDB(t))
		_, err := m.SelectHistogram(rowsErrCtx(t), 1750322640, 1750322640+3600,
			nil, nil, "queryid")
		require.ErrorIs(t, err, errMidStream)
	})
}
