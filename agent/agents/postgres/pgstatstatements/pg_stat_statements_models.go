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

package pgstatstatements

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/blang/semver"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

// pgStatStatements represents a row in pg_stat_statements view.
type pgStatStatements struct {
	UserID        int64
	DBID          int64
	QueryID       int64  // we select only non-NULL rows
	Query         string // we select only non-NULL rows
	Calls         int64
	TotalExecTime float64
	// MinTime
	// MaxTime
	// MeanTime
	// StddevTime
	Rows               int64
	SharedBlksHit      int64
	SharedBlksRead     int64
	SharedBlksDirtied  int64
	SharedBlksWritten  int64
	LocalBlksHit       int64
	LocalBlksRead      int64
	LocalBlksDirtied   int64
	LocalBlksWritten   int64
	TempBlksRead       int64
	TempBlksWritten    int64
	SharedBlkReadTime  float64
	SharedBlkWriteTime float64
	LocalBlkReadTime   float64
	LocalBlkWriteTime  float64

	// reform related fields
	pointers []interface{}
	view     reform.View
}

type field struct {
	info    parse.FieldInfo
	pointer interface{}
}

func newPgStatMonitorStructs(vPGSS semver.Version) (*pgStatStatements, reform.View) { //nolint:ireturn
	s := &pgStatStatements{}
	fields := []field{
		{info: parse.FieldInfo{Name: "UserID", Type: "int64", Column: "userid"}, pointer: &s.UserID},
		{info: parse.FieldInfo{Name: "DBID", Type: "int64", Column: "dbid"}, pointer: &s.DBID},
		{info: parse.FieldInfo{Name: "QueryID", Type: "int64", Column: "queryid"}, pointer: &s.QueryID},
		{info: parse.FieldInfo{Name: "Query", Type: "string", Column: "query"}, pointer: &s.Query},
		{info: parse.FieldInfo{Name: "Calls", Type: "int64", Column: "calls"}, pointer: &s.Calls},
		{info: parse.FieldInfo{Name: "Rows", Type: "int64", Column: "rows"}, pointer: &s.Rows},
		{info: parse.FieldInfo{Name: "SharedBlksHit", Type: "int64", Column: "shared_blks_hit"}, pointer: &s.SharedBlksHit},
		{info: parse.FieldInfo{Name: "SharedBlksRead", Type: "int64", Column: "shared_blks_read"}, pointer: &s.SharedBlksRead},
		{info: parse.FieldInfo{Name: "SharedBlksDirtied", Type: "int64", Column: "shared_blks_dirtied"}, pointer: &s.SharedBlksDirtied},
		{info: parse.FieldInfo{Name: "SharedBlksWritten", Type: "int64", Column: "shared_blks_written"}, pointer: &s.SharedBlksWritten},
		{info: parse.FieldInfo{Name: "LocalBlksHit", Type: "int64", Column: "local_blks_hit"}, pointer: &s.LocalBlksHit},
		{info: parse.FieldInfo{Name: "LocalBlksRead", Type: "int64", Column: "local_blks_read"}, pointer: &s.LocalBlksRead},
		{info: parse.FieldInfo{Name: "LocalBlksDirtied", Type: "int64", Column: "local_blks_dirtied"}, pointer: &s.LocalBlksDirtied},
		{info: parse.FieldInfo{Name: "LocalBlksWritten", Type: "int64", Column: "local_blks_written"}, pointer: &s.LocalBlksWritten},
		{info: parse.FieldInfo{Name: "TempBlksRead", Type: "int64", Column: "temp_blks_read"}, pointer: &s.TempBlksRead},
		{info: parse.FieldInfo{Name: "TempBlksWritten", Type: "int64", Column: "temp_blks_written"}, pointer: &s.TempBlksWritten},
	}

	if vPGSS.LT(pgStatVer1_8) {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TotalExecTime", Type: "float64", Column: "total_time"}, pointer: &s.TotalExecTime})
	} else {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TotalExecTime", Type: "float64", Column: "total_exec_time"}, pointer: &s.TotalExecTime})
	}

	if vPGSS.LT(pgStatVer1_11) {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "SharedBlkReadTime", Type: "float64", Column: "blk_read_time"}, pointer: &s.SharedBlkReadTime},
			field{info: parse.FieldInfo{Name: "SharedBlkWriteTime", Type: "float64", Column: "blk_write_time"}, pointer: &s.SharedBlkWriteTime})
	} else {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "SharedBlkReadTime", Type: "float64", Column: "shared_blk_read_time"}, pointer: &s.SharedBlkReadTime},
			field{info: parse.FieldInfo{Name: "SharedBlkWriteTime", Type: "float64", Column: "shared_blk_write_time"}, pointer: &s.SharedBlkWriteTime},
			field{info: parse.FieldInfo{Name: "LocalBlkReadTime", Type: "float64", Column: "local_blk_read_time"}, pointer: &s.LocalBlkReadTime},
			field{info: parse.FieldInfo{Name: "LocalBlkWriteTime", Type: "float64", Column: "local_blk_write_time"}, pointer: &s.LocalBlkWriteTime})
	}

	s.pointers = make([]interface{}, len(fields))
	pgStatStatementsDefaultView := &pgStatStatementsAllViewType{
		s: parse.StructInfo{
			Type:         "pgStatStatements",
			SQLName:      "pg_stat_statements",
			Fields:       make([]parse.FieldInfo, len(fields)),
			PKFieldIndex: -1,
		},
		c:     make([]string, len(fields)),
		vPGSS: vPGSS,
	}
	for i, field := range fields {
		pgStatStatementsDefaultView.s.Fields[i] = field.info
		pgStatStatementsDefaultView.c[i] = field.info.Column
		s.pointers[i] = field.pointer
	}
	s.view = pgStatStatementsDefaultView
	pgStatStatementsDefaultView.z = s.Values()

	return s, pgStatStatementsDefaultView
}

type pgStatStatementsAllViewType struct {
	s     parse.StructInfo
	z     []interface{}
	c     []string
	vPGSS semver.Version
}

// Schema returns a schema name in SQL database ("").
func (v *pgStatStatementsAllViewType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("pg_stat_monitor").
func (v *pgStatStatementsAllViewType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *pgStatStatementsAllViewType) Columns() []string {
	return v.c
}

// NewStruct makes a new struct for that view or table.
func (v *pgStatStatementsAllViewType) NewStruct() reform.Struct { //nolint:ireturn
	str, _ := newPgStatMonitorStructs(v.vPGSS)
	return str
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *pgStatStatements) Values() []interface{} {
	values := make([]interface{}, len(s.pointers))
	for i, pointer := range s.pointers {
		values[i] = reflect.ValueOf(pointer).Interface()
	}
	return values
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *pgStatStatements) Pointers() []interface{} {
	return s.pointers
}

// View returns View object for that struct.
func (s *pgStatStatements) View() reform.View { //nolint:ireturn
	return s.view
}

func (s *pgStatStatements) String() string {
	res := make([]string, 19)
	res[0] = "UserID: " + reform.Inspect(s.UserID, true)
	res[1] = "DBID: " + reform.Inspect(s.DBID, true)
	res[2] = "QueryID: " + reform.Inspect(s.QueryID, true)
	res[3] = "Query: " + reform.Inspect(s.Query, true)
	res[4] = "Calls: " + reform.Inspect(s.Calls, true)
	res[5] = "TotalExecTime: " + reform.Inspect(s.TotalExecTime, true)
	res[6] = "Rows: " + reform.Inspect(s.Rows, true)
	res[7] = "SharedBlksHit: " + reform.Inspect(s.SharedBlksHit, true)
	res[8] = "SharedBlksRead: " + reform.Inspect(s.SharedBlksRead, true)
	res[9] = "SharedBlksDirtied: " + reform.Inspect(s.SharedBlksDirtied, true)
	res[10] = "SharedBlksWritten: " + reform.Inspect(s.SharedBlksWritten, true)
	res[11] = "LocalBlksHit: " + reform.Inspect(s.LocalBlksHit, true)
	res[12] = "LocalBlksRead: " + reform.Inspect(s.LocalBlksRead, true)
	res[13] = "LocalBlksDirtied: " + reform.Inspect(s.LocalBlksDirtied, true)
	res[14] = "LocalBlksWritten: " + reform.Inspect(s.LocalBlksWritten, true)
	res[15] = "TempBlksRead: " + reform.Inspect(s.TempBlksRead, true)
	res[16] = "TempBlksWritten: " + reform.Inspect(s.TempBlksWritten, true)
	res[17] = "SharedBlkReadTime: " + reform.Inspect(s.SharedBlkReadTime, true)
	res[18] = "SharedBlkWriteTime: " + reform.Inspect(s.SharedBlkWriteTime, true)
	return strings.Join(res, ", ")
}

var (
	// Check interfaces.
	_ reform.Struct = (*pgStatStatements)(nil)
	_ fmt.Stringer  = (*pgStatStatements)(nil)
)

// pgStatStatementsExtended contains pgStatStatements data and extends it with database, username and tables data.
// It's made for performance reason.
type pgStatStatementsExtended struct {
	pgStatStatements

	Database         string
	Username         string
	Tables           []string
	IsQueryTruncated bool
	Comments         map[string]string
	RealQuery        string // RealQuery is a query which is not truncated, it's not sent to API and stored only locally in memory.
}

func (e *pgStatStatementsExtended) String() string {
	return fmt.Sprintf("%q %q %v: %d: %s (truncated = %t) %v",
		e.Database, e.Username, e.Tables, e.QueryID, e.Query, e.IsQueryTruncated, e.Comments)
}
