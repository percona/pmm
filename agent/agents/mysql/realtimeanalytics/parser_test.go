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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// fakeScanner mimics *sql.Rows.Scan by copying predefined values into the destinations.
type fakeScanner struct {
	values []any
}

func (f *fakeScanner) Scan(dest ...any) error {
	for i, d := range dest {
		switch d := d.(type) {
		case *sql.NullInt64:
			*d = f.values[i].(sql.NullInt64) //nolint:forcetypeassert
		case *sql.NullString:
			*d = f.values[i].(sql.NullString) //nolint:forcetypeassert
		}
	}
	return nil
}

func TestParseProcessRow(t *testing.T) {
	t.Run("full row", func(t *testing.T) {
		row := &fakeScanner{values: []any{
			sql.NullInt64{Int64: 42, Valid: true},                                        // conn_id
			sql.NullString{String: "app_user@10.0.0.5", Valid: true},                     // user
			sql.NullString{String: "sakila", Valid: true},                                // db
			sql.NullString{String: "Query", Valid: true},                                 // command
			sql.NullString{String: "Sending data", Valid: true},                          // state
			sql.NullString{String: "SELECT * FROM film WHERE length > 120", Valid: true}, // current_statement
			sql.NullInt64{Int64: 1_500_000_000_000, Valid: true},                         // statement_latency (1.5s in ps)
		}}

		qData, err := parseProcessRow(row)
		require.NoError(t, err)
		require.NotNil(t, qData)

		assert.Equal(t, "42", qData.QueryId)
		assert.Equal(t, "SELECT * FROM film WHERE length > 120", qData.QueryText)
		assert.Equal(t, "10.0.0.5", qData.ClientAddress)
		assert.InDelta(t, 1.5, qData.QueryExecutionDuration.AsDuration().Seconds(), 0.01)

		payload := qData.GetMySqlPayload()
		require.NotNil(t, payload)
		assert.Equal(t, "app_user", payload.Username)
		assert.Equal(t, "10.0.0.5", payload.ClientHost)
		assert.Equal(t, "sakila", payload.DatabaseName)
		assert.Equal(t, "Query", payload.Command)
		assert.Equal(t, "Sending data", payload.State)
		assert.WithinDuration(t, time.Now().Add(-1500*time.Millisecond), payload.OperationStartTime.AsTime(), 2*time.Second)
	})

	t.Run("null db and state, user without host", func(t *testing.T) {
		row := &fakeScanner{values: []any{
			sql.NullInt64{Int64: 7, Valid: true},
			sql.NullString{String: "event_scheduler", Valid: true},
			sql.NullString{Valid: false},
			sql.NullString{String: "Query", Valid: true},
			sql.NullString{Valid: false},
			sql.NullString{String: "DO SLEEP(1)", Valid: true},
			sql.NullInt64{Valid: false},
		}}

		qData, err := parseProcessRow(row)
		require.NoError(t, err)

		payload := qData.GetMySqlPayload()
		require.NotNil(t, payload)
		assert.Equal(t, "event_scheduler", payload.Username)
		assert.Empty(t, payload.ClientHost)
		assert.Empty(t, payload.DatabaseName)
		assert.Empty(t, payload.State)
		assert.Nil(t, qData.QueryExecutionDuration)
	})
}

func TestSplitUserHost(t *testing.T) {
	tests := []struct {
		in       string
		wantUser string
		wantHost string
	}{
		{"root@localhost", "root", "localhost"},
		{"app@10.0.0.5", "app", "10.0.0.5"},
		{"user@host@weird", "user@host", "weird"},
		{"event_scheduler", "event_scheduler", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		user, host := splitUserHost(tt.in)
		assert.Equal(t, tt.wantUser, user, "user for %q", tt.in)
		assert.Equal(t, tt.wantHost, host, "host for %q", tt.in)
	}
}

// ensure payload type assertion used by the agent stays valid.
func TestMySQLPayloadType(t *testing.T) {
	qData := &rtav1.QueryData{
		Payload: &rtav1.QueryData_MySqlPayload{MySqlPayload: &rtav1.QueryMySQLData{}},
	}
	_, ok := qData.Payload.(*rtav1.QueryData_MySqlPayload)
	assert.True(t, ok)
}
