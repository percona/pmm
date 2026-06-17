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

package realtimeanalytics

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoerceValue(t *testing.T) {
	t.Parallel()

	assert.Nil(t, coerceValue(nil), "NULL must become nil")
	assert.Equal(t, int64(123), coerceValue(sql.RawBytes("123")))
	assert.Equal(t, int64(-5), coerceValue(sql.RawBytes("-5")))
	assert.Equal(t, int64(2648724198000), coerceValue(sql.RawBytes("2648724198000")))
	assert.Equal(t, 1.5, coerceValue(sql.RawBytes("1.5")))
	assert.Equal(t, "COMMIT", coerceValue(sql.RawBytes("COMMIT")))
	assert.Equal(t, "ACTIVE", coerceValue(sql.RawBytes("ACTIVE")))
	// non-nil empty value stays an empty string (not nil)
	assert.Equal(t, "", coerceValue(sql.RawBytes("")))
}

func TestMapHelpers(t *testing.T) {
	t.Parallel()

	row := map[string]any{
		"i":        int64(7),
		"f":        2.5,
		"s":        "text",
		"numStr":   "9",
		"floatStr": "3.5",
		"null":     nil,
	}

	assert.Equal(t, "7", mapString(row, "i"))
	assert.Equal(t, "text", mapString(row, "s"))
	assert.Equal(t, "", mapString(row, "missing"))
	assert.Equal(t, "", mapString(row, "null"))

	assert.Equal(t, int64(7), mapInt(row, "i"))
	assert.Equal(t, int64(2), mapInt(row, "f")) // truncates
	assert.Equal(t, int64(9), mapInt(row, "numStr"))
	assert.Equal(t, int64(0), mapInt(row, "missing"))

	assert.InDelta(t, 2.5, mapFloat(row, "f"), 0)
	assert.InDelta(t, float64(7), mapFloat(row, "i"), 0)
	assert.InDelta(t, 3.5, mapFloat(row, "floatStr"), 0)
	assert.InDelta(t, float64(0), mapFloat(row, "missing"), 0)
}

func TestBuildQueryData(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{
		serviceID:         "svc-1",
		serviceName:       "rta-mysql",
		dbInstanceAddress: "127.0.0.1:3306",
	}

	row := map[string]any{
		"conn_id":           int64(42),
		"user":              "sbtest@localhost",
		"db":                "sbtest",
		"command":           "Query",
		"state":             "executing",
		"statement_latency": int64(2_000_000_000), // 2ms expressed in picoseconds
		"current_statement": "SELECT 1",
		"rows_examined":     int64(200),
		"rows_sent":         int64(100),
		"full_scan":         "YES",
		"program_name":      "mysql",
		"trx_state":         "ACTIVE",
		"pid":               nil,
	}

	qd := m.buildQueryData(row)
	require.NotNil(t, qd)

	assert.Equal(t, "svc-1", qd.ServiceId)
	assert.Equal(t, "rta-mysql", qd.ServiceName)
	assert.Equal(t, "42", qd.QueryId)
	assert.Equal(t, "SELECT 1", qd.QueryText)
	// 2_000_000_000 ps / 1000 = 2_000_000 ns = 2ms
	assert.Equal(t, 2*time.Millisecond, qd.QueryExecutionDuration.AsDuration())

	p := qd.GetMySqlPayload()
	require.NotNil(t, p)
	assert.Equal(t, "127.0.0.1:3306", p.DbInstanceAddress)
	assert.Equal(t, "sbtest", p.DatabaseName)
	assert.Equal(t, "Query", p.Command)
	assert.Equal(t, "executing", p.State)
	assert.Equal(t, "sbtest@localhost", p.Username)
	assert.Equal(t, int64(200), p.RowsExamined)
	assert.Equal(t, int64(100), p.RowsSent)
	assert.True(t, p.FullScan)
	assert.Equal(t, "mysql", p.ProgramName)

	// Raw payload is pretty-printed (multi-line) and preserves the whole row, NULLs included.
	assert.Contains(t, qd.QueryRawJson, "\n")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(qd.QueryRawJson), &parsed))
	assert.Contains(t, parsed, "current_statement")
	assert.Contains(t, parsed, "statement_latency")
	assert.Contains(t, parsed, "trx_state")
	assert.Nil(t, parsed["pid"], "NULL columns are preserved as JSON null")
}

func TestBuildQueryDataFullScanAndMissing(t *testing.T) {
	t.Parallel()

	m := &MySQLRTA{serviceID: "svc", serviceName: "svc"}

	// full_scan "NO" -> false, and a missing statement_latency -> zero duration.
	qd := m.buildQueryData(map[string]any{
		"conn_id":           int64(1),
		"current_statement": "SELECT 2",
		"full_scan":         "NO",
	})
	require.NotNil(t, qd)
	assert.False(t, qd.GetMySqlPayload().FullScan)
	assert.Equal(t, time.Duration(0), qd.QueryExecutionDuration.AsDuration())
}
