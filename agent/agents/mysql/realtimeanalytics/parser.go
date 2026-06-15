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
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// processRow represents a single row of the sys schema processlist view.
type processRow struct {
	ConnID           sql.NullInt64
	User             sql.NullString
	DB               sql.NullString
	Command          sql.NullString
	State            sql.NullString
	CurrentStatement sql.NullString
	StatementLatency sql.NullInt64 // picoseconds
}

// rowScanner is implemented by *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// parseProcessRow scans a single sys schema processlist row and converts it into *QueryData.
func parseProcessRow(rows rowScanner) (*rtav1.QueryData, error) {
	var row processRow
	err := rows.Scan(
		&row.ConnID,
		&row.User,
		&row.DB,
		&row.Command,
		&row.State,
		&row.CurrentStatement,
		&row.StatementLatency,
	)
	if err != nil {
		return nil, err
	}

	username, clientHost := splitUserHost(row.User.String)

	payload := &rtav1.QueryMySQLData{
		ClientHost:   clientHost,
		DatabaseName: row.DB.String,
		Command:      row.Command.String,
		State:        row.State.String,
		Username:     username,
	}

	qData := &rtav1.QueryData{
		QueryId:       strconv.FormatInt(row.ConnID.Int64, 10),
		QueryText:     row.CurrentStatement.String,
		ClientAddress: clientHost,
		Payload: &rtav1.QueryData_MySqlPayload{
			MySqlPayload: payload,
		},
	}

	if row.StatementLatency.Valid {
		// statement_latency is reported in picoseconds by the x$ processlist view.
		d := time.Duration(row.StatementLatency.Int64/1000) * time.Nanosecond
		// Round duration to 0.01s to avoid too much precision, mirroring the MongoDB RTA agent.
		qData.QueryExecutionDuration = durationpb.New(d.Round(10 * time.Millisecond)) //nolint:mnd
		payload.OperationStartTime = timestamppb.New(time.Now().Add(-d))
	}

	qData.QueryRawJson = rawJSON(row)

	return qData, nil
}

// splitUserHost splits the sys.processlist "user" column, formatted as "user@host",
// into the user name and the client host. If there is no "@", the whole value is treated as the user.
func splitUserHost(userHost string) (user, host string) { //nolint:nonamedreturns
	idx := strings.LastIndex(userHost, "@")
	if idx < 0 {
		return userHost, ""
	}

	return userHost[:idx], userHost[idx+1:]
}

// rawJSON returns an indented JSON representation of the processlist row for the details view.
func rawJSON(row processRow) string {
	m := map[string]any{
		"conn_id":           nullInt(row.ConnID),
		"user":              nullStr(row.User),
		"db":                nullStr(row.DB),
		"command":           nullStr(row.Command),
		"state":             nullStr(row.State),
		"current_statement": nullStr(row.CurrentStatement),
		"statement_latency": nullInt(row.StatementLatency),
	}

	b, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return ""
	}

	return string(b)
}

func nullStr(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func nullInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}
