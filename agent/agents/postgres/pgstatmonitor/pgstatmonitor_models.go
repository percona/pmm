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

package pgstatmonitor

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/lib/pq"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

var (
	v21 = version.Must(version.NewVersion("2.1.0"))
	v20 = version.Must(version.NewVersion("2.0.0"))
	v11 = version.Must(version.NewVersion("1.1.0"))
	v10 = version.Must(version.NewVersion("1.0.0"))
	v09 = version.Must(version.NewVersion("0.9"))
	v08 = version.Must(version.NewVersion("0.8"))
)

// pgStatMonitor represents a row in pg_stat_monitor view.
type pgStatMonitor struct {
	// PGSM < 0.6.0
	DBID   int64
	UserID int64

	// PGSM >= 0.8.0
	DatName               string
	UserName              string
	BucketStartTimeString string

	// rest
	Bucket             int64
	BucketStartTime    time.Time
	ClientIP           string
	QueryID            string // we select only non-NULL rows
	Query              string // we select only non-NULL rows
	Comments           *string
	Relations          pq.StringArray
	Calls              int64
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
	RespCalls          pq.StringArray
	CPUUserTime        float64
	CPUSysTime         float64
	Rows               int64

	TopQueryID      *string
	PlanID          *string
	QueryPlan       *string
	TopQuery        *string
	ApplicationName *string
	CmdType         int32
	CmdTypeText     string
	Elevel          int32
	Sqlcode         *string
	Message         *string
	TotalExecTime   float64
	MinExecTime     float64
	MaxExecTime     float64
	MeanExecTime    float64
	StddevExecTime  float64
	PlansCalls      int64
	TotalPlanTime   float64
	MinPlanTime     float64
	MaxPlanTime     float64
	MeanPlanTime    float64
	WalRecords      int64
	WalFpi          int64
	WalBytes        int64

	// reform related fields
	pointers []interface{}
	view     reform.View
}

type field struct {
	info    parse.FieldInfo
	pointer interface{}
}

func newPgStatMonitorStructs(vPGSM pgStatMonitorVersion, vPG pgVersion) (*pgStatMonitor, reform.View) { //nolint:ireturn
	s := &pgStatMonitor{}
	fields := []field{
		{info: parse.FieldInfo{Name: "Bucket", Type: "int64", Column: "bucket"}, pointer: &s.Bucket},
		{info: parse.FieldInfo{Name: "ClientIP", Type: "string", Column: "client_ip"}, pointer: &s.ClientIP},
		{info: parse.FieldInfo{Name: "Query", Type: "string", Column: "query"}, pointer: &s.Query},
		{info: parse.FieldInfo{Name: "Calls", Type: "int64", Column: "calls"}, pointer: &s.Calls},
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
		{info: parse.FieldInfo{Name: "RespCalls", Type: "pq.StringArray", Column: "resp_calls"}, pointer: &s.RespCalls},
		{info: parse.FieldInfo{Name: "CPUUserTime", Type: "float64", Column: "cpu_user_time"}, pointer: &s.CPUUserTime},
		{info: parse.FieldInfo{Name: "CPUSysTime", Type: "float64", Column: "cpu_sys_time"}, pointer: &s.CPUSysTime},
	}

	if vPGSM == pgStatMonitorVersion06 {
		// versions older than 0.8
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "Relations", Type: "pq.StringArray", Column: "tables_names"}, pointer: &s.Relations},
			field{info: parse.FieldInfo{Name: "DBID", Type: "int64", Column: "dbid"}, pointer: &s.DBID})
	}
	if vPGSM <= pgStatMonitorVersion08 || vPGSM >= pgStatMonitorVersion20PG12 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "Rows", Type: "int64", Column: "rows"}, pointer: &s.Rows})
	} else {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "Rows", Type: "int64", Column: "rows_retrieved"}, pointer: &s.Rows})
	}
	if vPGSM >= pgStatMonitorVersion08 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "Relations", Type: "pq.StringArray", Column: "relations"}, pointer: &s.Relations},
			field{info: parse.FieldInfo{Name: "DatName", Type: "string", Column: "datname"}, pointer: &s.DatName},
			field{info: parse.FieldInfo{Name: "UserName", Type: "string", Column: "userid"}, pointer: &s.UserName})
	}
	if vPGSM == pgStatMonitorVersion09 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TotalPlanTime", Type: "float64", Column: "plan_total_time"}, pointer: &s.TotalPlanTime},
			field{info: parse.FieldInfo{Name: "MinPlanTime", Type: "float64", Column: "plan_min_time"}, pointer: &s.MinPlanTime},
			field{info: parse.FieldInfo{Name: "MaxPlanTime", Type: "float64", Column: "plan_max_time"}, pointer: &s.MaxPlanTime},
			field{info: parse.FieldInfo{Name: "MeanPlanTime", Type: "float64", Column: "plan_mean_time"}, pointer: &s.MeanPlanTime},
			field{info: parse.FieldInfo{Name: "PlansCalls", Type: "int64", Column: "plans_calls"}, pointer: &s.PlansCalls})
	}
	if vPGSM >= pgStatMonitorVersion09 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TopQueryID", Type: "*string", Column: "top_queryid"}, pointer: &s.TopQueryID},
			field{info: parse.FieldInfo{Name: "PlanID", Type: "*string", Column: "planid"}, pointer: &s.PlanID},
			field{info: parse.FieldInfo{Name: "QueryPlan", Type: "*string", Column: "query_plan"}, pointer: &s.QueryPlan},
			field{info: parse.FieldInfo{Name: "TopQuery", Type: "*string", Column: "top_query"}, pointer: &s.TopQuery},
			field{info: parse.FieldInfo{Name: "ApplicationName", Type: "*string", Column: "application_name"}, pointer: &s.ApplicationName},
			field{info: parse.FieldInfo{Name: "CmdType", Type: "int32", Column: "cmd_type"}, pointer: &s.CmdType},
			field{info: parse.FieldInfo{Name: "CmdTypeText", Type: "string", Column: "cmd_type_text"}, pointer: &s.CmdTypeText},
			field{info: parse.FieldInfo{Name: "Elevel", Type: "int32", Column: "elevel"}, pointer: &s.Elevel},
			field{info: parse.FieldInfo{Name: "Sqlcode", Type: "*string", Column: "sqlcode"}, pointer: &s.Sqlcode},
			field{info: parse.FieldInfo{Name: "Message", Type: "*string", Column: "message"}, pointer: &s.Message})
	}
	if vPGSM >= pgStatMonitorVersion20PG12 {
		fields = append(fields, field{info: parse.FieldInfo{Name: "QueryID", Type: "string", Column: "pgsm_query_id"}, pointer: &s.QueryID},
			field{info: parse.FieldInfo{Name: "DBID", Type: "int64", Column: "dbid"}, pointer: &s.DBID})
	} else {
		fields = append(fields, field{info: parse.FieldInfo{Name: "QueryID", Type: "string", Column: "queryid"}, pointer: &s.QueryID})
	}

	if vPGSM >= pgStatMonitorVersion21PG17 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "SharedBlkReadTime", Type: "float64", Column: "shared_blk_read_time"}, pointer: &s.SharedBlkReadTime},
			field{info: parse.FieldInfo{Name: "SharedBlkWriteTime", Type: "float64", Column: "shared_blk_write_time"}, pointer: &s.SharedBlkWriteTime},
			field{info: parse.FieldInfo{Name: "LocalBlkReadTime", Type: "float64", Column: "local_blk_read_time"}, pointer: &s.LocalBlkReadTime},
			field{info: parse.FieldInfo{Name: "LocalBlkWriteTime", Type: "float64", Column: "local_blk_write_time"}, pointer: &s.LocalBlkWriteTime})
	} else {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "SharedBlkReadTime", Type: "float64", Column: "blk_read_time"}, pointer: &s.SharedBlkReadTime},
			field{info: parse.FieldInfo{Name: "SharedBlkWriteTime", Type: "float64", Column: "blk_write_time"}, pointer: &s.SharedBlkWriteTime})
	}

	if vPG <= 12 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TotalExecTime", Type: "float64", Column: "total_time"}, pointer: &s.TotalExecTime},
			field{info: parse.FieldInfo{Name: "MinExecTime", Type: "float64", Column: "min_time"}, pointer: &s.MinExecTime},
			field{info: parse.FieldInfo{Name: "MaxExecTime", Type: "float64", Column: "max_time"}, pointer: &s.MaxExecTime},
			field{info: parse.FieldInfo{Name: "MeanExecTime", Type: "float64", Column: "mean_time"}, pointer: &s.MeanExecTime},
			field{info: parse.FieldInfo{Name: "StddevExecTime", Type: "float64", Column: "stddev_time"}, pointer: &s.StddevExecTime})
	}
	if vPG >= 13 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "TotalExecTime", Type: "float64", Column: "total_exec_time"}, pointer: &s.TotalExecTime},
			field{info: parse.FieldInfo{Name: "MinExecTime", Type: "float64", Column: "min_exec_time"}, pointer: &s.MinExecTime},
			field{info: parse.FieldInfo{Name: "MaxExecTime", Type: "float64", Column: "max_exec_time"}, pointer: &s.MaxExecTime},
			field{info: parse.FieldInfo{Name: "MeanExecTime", Type: "float64", Column: "mean_exec_time"}, pointer: &s.MeanExecTime},
			field{info: parse.FieldInfo{Name: "StddevExecTime", Type: "float64", Column: "stddev_exec_time"}, pointer: &s.StddevExecTime},
			field{info: parse.FieldInfo{Name: "TotalPlanTime", Type: "float64", Column: "total_plan_time"}, pointer: &s.TotalPlanTime},
			field{info: parse.FieldInfo{Name: "MinPlanTime", Type: "float64", Column: "min_plan_time"}, pointer: &s.MinPlanTime},
			field{info: parse.FieldInfo{Name: "MaxPlanTime", Type: "float64", Column: "max_plan_time"}, pointer: &s.MaxPlanTime},
			field{info: parse.FieldInfo{Name: "MeanPlanTime", Type: "float64", Column: "mean_plan_time"}, pointer: &s.MeanPlanTime})

		if vPGSM >= pgStatMonitorVersion09 {
			fields = append(fields,
				field{info: parse.FieldInfo{Name: "WalRecords", Type: "int64", Column: "wal_records"}, pointer: &s.WalRecords},
				field{info: parse.FieldInfo{Name: "WalFpi", Type: "int64", Column: "wal_fpi"}, pointer: &s.WalFpi},
				field{info: parse.FieldInfo{Name: "WalBytes", Type: "int64", Column: "wal_bytes"}, pointer: &s.WalBytes})
		}

		if vPGSM >= pgStatMonitorVersion20PG12 {
			fields = append(fields, field{info: parse.FieldInfo{Name: "PlansCalls", Type: "int64", Column: "plans"}, pointer: &s.PlansCalls})
			fields = append(fields, field{info: parse.FieldInfo{Name: "Comments", Type: "string", Column: "comments"}, pointer: &s.Comments})
		} else {
			fields = append(fields, field{info: parse.FieldInfo{Name: "PlansCalls", Type: "int64", Column: "plans_calls"}, pointer: &s.PlansCalls})
		}
	}

	if vPGSM >= pgStatMonitorVersion08 && vPGSM < pgStatMonitorVersion20PG12 {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "BucketStartTimeString", Type: "string", Column: "bucket_start_time"}, pointer: &s.BucketStartTimeString},
			field{info: parse.FieldInfo{Name: "UserName", Type: "string", Column: "userid"}, pointer: &s.UserName})
	} else {
		fields = append(fields,
			field{info: parse.FieldInfo{Name: "BucketStartTime", Type: "time.Time", Column: "bucket_start_time"}, pointer: &s.BucketStartTime},
			field{info: parse.FieldInfo{Name: "UserName", Type: "string", Column: "username"}, pointer: &s.UserName})
	}

	s.pointers = make([]interface{}, len(fields))
	pgStatMonitorDefaultView := &pgStatMonitorAllViewType{
		s: parse.StructInfo{
			Type:         "pgStatMonitor",
			SQLName:      "pg_stat_monitor",
			Fields:       make([]parse.FieldInfo, len(fields)),
			PKFieldIndex: -1,
		},
		c:     make([]string, len(fields)),
		vPGSM: vPGSM,
		vPG:   vPG,
	}
	for i, field := range fields {
		pgStatMonitorDefaultView.s.Fields[i] = field.info
		pgStatMonitorDefaultView.c[i] = field.info.Column
		s.pointers[i] = field.pointer
	}
	s.view = pgStatMonitorDefaultView
	pgStatMonitorDefaultView.z = s.Values()
	return s, pgStatMonitorDefaultView
}

type pgStatMonitorAllViewType struct {
	s     parse.StructInfo
	z     []interface{}
	c     []string
	vPGSM pgStatMonitorVersion
	vPG   pgVersion
}

// Schema returns a schema name in SQL database ("").
func (v *pgStatMonitorAllViewType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("pg_stat_monitor").
func (v *pgStatMonitorAllViewType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *pgStatMonitorAllViewType) Columns() []string {
	return v.c
}

// NewStruct makes a new struct for that view or table.
func (v *pgStatMonitorAllViewType) NewStruct() reform.Struct { //nolint:ireturn
	str, _ := newPgStatMonitorStructs(v.vPGSM, v.vPG)
	return str
}

// String returns a string representation of this struct or record.
func (s pgStatMonitor) String() string {
	res := make([]string, 51)
	res[0] = "Bucket: " + reform.Inspect(s.Bucket, true)
	res[1] = "BucketStartTime: " + reform.Inspect(s.BucketStartTime, true)
	res[2] = "UserID: " + reform.Inspect(s.UserID, true)
	res[3] = "ClientIP: " + reform.Inspect(s.ClientIP, true)
	res[4] = "QueryID: " + reform.Inspect(s.QueryID, true)
	res[5] = "Query: " + reform.Inspect(s.Query, true)
	res[6] = "Relations: " + reform.Inspect(s.Relations, true)
	res[7] = "Calls: " + reform.Inspect(s.Calls, true)
	res[8] = "TotalExecTime: " + reform.Inspect(s.TotalExecTime, true)
	res[9] = "SharedBlksHit: " + reform.Inspect(s.SharedBlksHit, true)
	res[10] = "SharedBlksRead: " + reform.Inspect(s.SharedBlksRead, true)
	res[11] = "SharedBlksDirtied: " + reform.Inspect(s.SharedBlksDirtied, true)
	res[12] = "SharedBlksWritten: " + reform.Inspect(s.SharedBlksWritten, true)
	res[13] = "LocalBlksHit: " + reform.Inspect(s.LocalBlksHit, true)
	res[14] = "LocalBlksRead: " + reform.Inspect(s.LocalBlksRead, true)
	res[15] = "LocalBlksDirtied: " + reform.Inspect(s.LocalBlksDirtied, true)
	res[16] = "LocalBlksWritten: " + reform.Inspect(s.LocalBlksWritten, true)
	res[17] = "TempBlksRead: " + reform.Inspect(s.TempBlksRead, true)
	res[18] = "TempBlksWritten: " + reform.Inspect(s.TempBlksWritten, true)
	res[19] = "SharedBlkReadTime: " + reform.Inspect(s.SharedBlkReadTime, true)
	res[20] = "SharedBlkWriteTime: " + reform.Inspect(s.SharedBlkWriteTime, true)
	res[21] = "LocalBlkReadTime: " + reform.Inspect(s.LocalBlkReadTime, true)
	res[22] = "LocalBlkWriteTime: " + reform.Inspect(s.LocalBlkWriteTime, true)
	res[23] = "RespCalls: " + reform.Inspect(s.RespCalls, true)
	res[24] = "CPUUserTime: " + reform.Inspect(s.CPUUserTime, true)
	res[25] = "CPUSysTime: " + reform.Inspect(s.CPUSysTime, true)
	res[26] = "DBID: " + reform.Inspect(s.DBID, true)
	res[27] = "DatName: " + reform.Inspect(s.DatName, true)
	res[28] = "Rows: " + reform.Inspect(s.Rows, true)
	res[29] = "TopQueryID: " + reform.Inspect(s.TopQueryID, true)
	res[30] = "PlanID: " + reform.Inspect(s.PlanID, true)
	res[31] = "QueryPlan: " + reform.Inspect(s.QueryPlan, true)
	res[32] = "TopQuery: " + reform.Inspect(s.TopQuery, true)
	res[33] = "ApplicationName: " + reform.Inspect(s.ApplicationName, true)
	res[34] = "CmdType: " + reform.Inspect(s.CmdType, true)
	res[35] = "CmdTypeText: " + reform.Inspect(s.CmdTypeText, true)
	res[36] = "Elevel: " + reform.Inspect(s.Elevel, true)
	res[37] = "Sqlcode: " + reform.Inspect(s.Sqlcode, true)
	res[38] = "Message: " + reform.Inspect(s.Message, true)
	res[39] = "MinExecTime: " + reform.Inspect(s.MinExecTime, true)
	res[40] = "MaxExecTime: " + reform.Inspect(s.MaxExecTime, true)
	res[41] = "MeanExecTime: " + reform.Inspect(s.MeanExecTime, true)
	res[42] = "StddevExecTime: " + reform.Inspect(s.StddevExecTime, true)
	res[43] = "PlansCalls: " + reform.Inspect(s.PlansCalls, true)
	res[44] = "TotalPlanTime: " + reform.Inspect(s.TotalPlanTime, true)
	res[45] = "MinPlanTime: " + reform.Inspect(s.MinPlanTime, true)
	res[46] = "MaxPlanTime: " + reform.Inspect(s.MaxPlanTime, true)
	res[47] = "MeanPlanTime: " + reform.Inspect(s.MeanPlanTime, true)
	res[48] = "WalRecords: " + reform.Inspect(s.WalRecords, true)
	res[49] = "WalFpi: " + reform.Inspect(s.WalFpi, true)
	res[50] = "WalBytes: " + reform.Inspect(s.WalBytes, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *pgStatMonitor) Values() []interface{} {
	values := make([]interface{}, len(s.pointers))
	for i, pointer := range s.pointers {
		values[i] = reflect.ValueOf(pointer).Interface()
	}
	return values
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *pgStatMonitor) Pointers() []interface{} {
	return s.pointers
}

// View returns View object for that struct.
func (s *pgStatMonitor) View() reform.View { //nolint:ireturn
	return s.view
}

var (
	// Check interfaces.
	_ reform.Struct = (*pgStatMonitor)(nil)
	_ fmt.Stringer  = (*pgStatMonitor)(nil)
)
