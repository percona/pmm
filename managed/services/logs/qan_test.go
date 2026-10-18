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
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

func TestStatementOf(t *testing.T) {
	for body, want := range map[string]string{
		"statement: SELECT 1;":                                                                 "SELECT 1;",
		"duration: 1003.112 ms  statement: SELECT pg_sleep(1)":                                 "SELECT pg_sleep(1)",
		"execute <unnamed>: SELECT pg_database_size($1)\nDETAIL:  Parameters: $1 = 'postgres'": "SELECT pg_database_size($1)",
		"statement: SELECT a,\n  b\nFROM t":                                                    "SELECT a,\n  b\nFROM t",
		"connection authorized: user=pmm database=shop":                                        "",
		"statement: ;": "",
	} {
		assert.Equal(t, want, statementOf(body), body)
	}
}

func TestNormalizeSQL(t *testing.T) {
	for logged, fingerprint := range map[string]string{
		"UPDATE orders SET status = 'shipped' WHERE id = 1815961;": "UPDATE orders SET status = $1 WHERE id = $2",
		"SELECT /* agent='pgstatmonitor' */ version()":             "SELECT /* agent='pgstatmonitor' */ version()",
		"SELECT * FROM t WHERE id IN (1, 2, 3)":                    "SELECT * FROM t WHERE id IN ($1, $2)",
		"SELECT\n  a,b FROM t WHERE x = 'it''s'":                   "select a, b from t where x = $1",
		"SELECT pg_sleep(1.5)":                                     "SELECT pg_sleep($1)",
	} {
		assert.Equal(t, normalizeSQL(fingerprint), normalizeSQL(logged), logged)
	}
	assert.NotEqual(t,
		normalizeSQL("UPDATE accounts SET balance = balance - $1 WHERE id = $2"),
		normalizeSQL("UPDATE accounts SET balance = balance + 5 WHERE id = 1"))
	assert.NotEqual(t, normalizeSQL("SELECT a FROM t2"), normalizeSQL("SELECT a FROM t3"))
}

func TestFindLogQuery(t *testing.T) {
	s, mock := newTestService(t)
	at := testStart.Add(10 * time.Minute)
	q, _ := findQueryQuery("pg-node", at)
	cols := []string{"queryid", "service_name", "service_id", "fingerprint", "num"}
	mock.ExpectQuery(q).WillReturnRows(mock.NewRows(cols).
		AddRow("-21", "pg-tcp", "s1", "UPDATE accounts SET balance = balance - $1 WHERE id = $2", float64(90)).
		AddRow("42", "pg-tcp", "s1", "UPDATE orders SET status = $1 WHERE id = $2", float64(50)).
		AddRow("42", "pg-socket", "s2", "UPDATE orders SET status = $1 WHERE id = $2", float64(40)))
	res, err := s.FindLogQuery(t.Context(), &logsv1.FindLogQueryRequest{
		Time:     timestamppb.New(at),
		NodeName: "pg-node",
		Body:     "statement: UPDATE orders SET status = 'shipped' WHERE id = 7;",
	})
	require.NoError(t, err)
	assert.Equal(t, &logsv1.FindLogQueryResponse{
		QueryId:     "42",
		ServiceName: "pg-tcp",
		ServiceId:   "s1",
		Fingerprint: "UPDATE orders SET status = $1 WHERE id = $2",
	}, res)

	// A line without a statement has no query; QAN is not asked.
	res, err = s.FindLogQuery(t.Context(), &logsv1.FindLogQueryRequest{
		Time: timestamppb.New(at), NodeName: "pg-node", Body: "connection authorized: user=pmm",
	})
	require.NoError(t, err)
	assert.Empty(t, res.QueryId)

	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-proxy-filter", "x"))
	_, err = s.FindLogQuery(ctx, &logsv1.FindLogQueryRequest{Time: timestamppb.New(at), NodeName: "n", Body: "statement: SELECT 1"})
	requireCode(t, err, codes.PermissionDenied)
}

func TestFindQueryQuery(t *testing.T) {
	at := testStart
	q, args := findQueryQuery("pg-node", at)
	assert.Contains(t, q, "FROM pmm.metrics WHERE node_name = ? AND service_type = 'postgresql' AND period_start BETWEEN toDateTime(?) AND toDateTime(?)")
	assert.Equal(t, []any{"pg-node", at.Add(-30 * time.Minute).Unix(), at.Add(30 * time.Minute).Unix()}, args)
	assert.NotRegexp(t, "{.+:.+}", q)
	_ = sqlmock.AnyArg
}
