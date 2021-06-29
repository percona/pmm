// pmm-agent
// Copyright 2019 Percona LLC
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
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

//go:generate reform

// pgStatDatabase represents a row in pg_stat_database view.
//reform:pg_catalog.pg_stat_database
type pgStatDatabase struct {
	DatID   int64   `reform:"datid"`
	DatName *string `reform:"datname"`
}

// pgUser represents a row in pg_user view.
//reform:pg_catalog.pg_user
type pgUser struct {
	UserID   int64   `reform:"usesysid"`
	UserName *string `reform:"usename"`
}

// pgStatMonitorDefault represents a row in pg_stat_monitor
// view in version lower than 0.8.
//reform:pg_stat_monitor
type pgStatMonitorDefault struct {
	Bucket            int64          `reform:"bucket"`
	BucketStartTime   time.Time      `reform:"bucket_start_time"`
	UserID            int64          `reform:"userid"`
	DBID              int64          `reform:"dbid"`
	QueryID           string         `reform:"queryid"` // we select only non-NULL rows
	Query             string         `reform:"query"`   // we select only non-NULL rows
	Calls             int64          `reform:"calls"`
	TotalTime         float64        `reform:"total_time"`
	Rows              int64          `reform:"rows"`
	SharedBlksHit     int64          `reform:"shared_blks_hit"`
	SharedBlksRead    int64          `reform:"shared_blks_read"`
	SharedBlksDirtied int64          `reform:"shared_blks_dirtied"`
	SharedBlksWritten int64          `reform:"shared_blks_written"`
	LocalBlksHit      int64          `reform:"local_blks_hit"`
	LocalBlksRead     int64          `reform:"local_blks_read"`
	LocalBlksDirtied  int64          `reform:"local_blks_dirtied"`
	LocalBlksWritten  int64          `reform:"local_blks_written"`
	TempBlksRead      int64          `reform:"temp_blks_read"`
	TempBlksWritten   int64          `reform:"temp_blks_written"`
	BlkReadTime       float64        `reform:"blk_read_time"`
	BlkWriteTime      float64        `reform:"blk_write_time"`
	ClientIP          string         `reform:"client_ip"`
	RespCalls         pq.StringArray `reform:"resp_calls"`
	CPUUserTime       float64        `reform:"cpu_user_time"`
	CPUSysTime        float64        `reform:"cpu_sys_time"`
	Relations         pq.StringArray `reform:"relations"`
}

func (m pgStatMonitorDefault) ToPgStatMonitor() pgStatMonitor {
	return pgStatMonitor{
		Bucket:            m.Bucket,
		BucketStartTime:   m.BucketStartTime,
		UserID:            m.UserID,
		DBID:              m.DBID,
		QueryID:           m.QueryID,
		Query:             m.Query,
		Calls:             m.Calls,
		TotalTime:         m.TotalTime,
		Rows:              m.Rows,
		SharedBlksHit:     m.SharedBlksHit,
		SharedBlksRead:    m.SharedBlksRead,
		SharedBlksDirtied: m.SharedBlksDirtied,
		SharedBlksWritten: m.SharedBlksWritten,
		LocalBlksHit:      m.LocalBlksHit,
		LocalBlksRead:     m.LocalBlksRead,
		LocalBlksDirtied:  m.LocalBlksDirtied,
		LocalBlksWritten:  m.LocalBlksWritten,
		TempBlksRead:      m.TempBlksRead,
		TempBlksWritten:   m.TempBlksWritten,
		BlkReadTime:       m.BlkReadTime,
		BlkWriteTime:      m.BlkWriteTime,
		ClientIP:          m.ClientIP,
		RespCalls:         m.RespCalls,
		CPUUserTime:       m.CPUUserTime,
		CPUSysTime:        m.CPUSysTime,
		Relations:         m.Relations,
	}
}

// pgStatMonitor08 represents a row in pg_stat_monitor
// view in version 0.8 and higher.
//reform:pg_stat_monitor
type pgStatMonitor08 struct {
	Bucket            int64          `reform:"bucket"`
	BucketStartTime   string         `reform:"bucket_start_time"`
	User              string         `reform:"userid"`
	DatName           string         `reform:"datname"`
	QueryID           string         `reform:"queryid"` // we select only non-NULL rows
	Query             string         `reform:"query"`   // we select only non-NULL rows
	Calls             int64          `reform:"calls"`
	TotalTime         float64        `reform:"total_time"`
	Rows              int64          `reform:"rows"`
	SharedBlksHit     int64          `reform:"shared_blks_hit"`
	SharedBlksRead    int64          `reform:"shared_blks_read"`
	SharedBlksDirtied int64          `reform:"shared_blks_dirtied"`
	SharedBlksWritten int64          `reform:"shared_blks_written"`
	LocalBlksHit      int64          `reform:"local_blks_hit"`
	LocalBlksRead     int64          `reform:"local_blks_read"`
	LocalBlksDirtied  int64          `reform:"local_blks_dirtied"`
	LocalBlksWritten  int64          `reform:"local_blks_written"`
	TempBlksRead      int64          `reform:"temp_blks_read"`
	TempBlksWritten   int64          `reform:"temp_blks_written"`
	BlkReadTime       float64        `reform:"blk_read_time"`
	BlkWriteTime      float64        `reform:"blk_write_time"`
	ClientIP          string         `reform:"client_ip"`
	RespCalls         pq.StringArray `reform:"resp_calls"`
	CPUUserTime       float64        `reform:"cpu_user_time"`
	CPUSysTime        float64        `reform:"cpu_sys_time"`
	Relations         pq.StringArray `reform:"relations"`
}

func (m pgStatMonitor08) ToPgStatMonitor() (pgStatMonitor, error) {
	bucketStartTime, err := time.Parse("2006-01-02 15:04:05", m.BucketStartTime)
	if err != nil {
		return pgStatMonitor{}, err
	}

	return pgStatMonitor{
		Bucket:            m.Bucket,
		BucketStartTime:   bucketStartTime,
		User:              m.User,
		DatName:           m.DatName,
		QueryID:           m.QueryID,
		Query:             m.Query,
		Calls:             m.Calls,
		TotalTime:         m.TotalTime,
		Rows:              m.Rows,
		SharedBlksHit:     m.SharedBlksHit,
		SharedBlksRead:    m.SharedBlksRead,
		SharedBlksDirtied: m.SharedBlksDirtied,
		SharedBlksWritten: m.SharedBlksWritten,
		LocalBlksHit:      m.LocalBlksHit,
		LocalBlksRead:     m.LocalBlksRead,
		LocalBlksDirtied:  m.LocalBlksDirtied,
		LocalBlksWritten:  m.LocalBlksWritten,
		TempBlksRead:      m.TempBlksRead,
		TempBlksWritten:   m.TempBlksWritten,
		BlkReadTime:       m.BlkReadTime,
		BlkWriteTime:      m.BlkWriteTime,
		ClientIP:          m.ClientIP,
		RespCalls:         m.RespCalls,
		CPUUserTime:       m.CPUUserTime,
		CPUSysTime:        m.CPUSysTime,
		Relations:         m.Relations,
	}, nil
}

// pgStatMonitor09 represents a row in pg_stat_monitor
// view in version 0.9 and higher.
//reform:pg_stat_monitor
type pgStatMonitor09 struct {
	Bucket            int64          `reform:"bucket"`
	BucketStartTime   string         `reform:"bucket_start_time"`
	User              string         `reform:"userid"`
	DatName           string         `reform:"datname"`
	ClientIP          string         `reform:"client_ip"`
	QueryID           string         `reform:"queryid"`
	TopQueryid        *string        `reform:"top_queryid"`
	Query             string         `reform:"query"`
	Planid            *string        `reform:"planid"`
	QueryPlan         *string        `reform:"query_plan"`
	TopQuery          *string        `reform:"top_query"`
	ApplicationName   *string        `reform:"application_name"`
	Relations         pq.StringArray `reform:"relations"`
	CmdType           int32          `reform:"cmd_type"`
	CmdTypeText       string         `reform:"cmd_type_text"`
	Elevel            int32          `reform:"elevel"`
	Sqlcode           *string        `reform:"sqlcode"`
	Message           *string        `reform:"message"`
	Calls             int64          `reform:"calls"`
	TotalTime         float64        `reform:"total_time"`
	MinTime           float64        `reform:"min_time"`
	MaxTime           float64        `reform:"max_time"`
	MeanTime          float64        `reform:"mean_time"`
	StddevTime        float64        `reform:"stddev_time"`
	RowsRetrieved     int64          `reform:"rows_retrieved"`
	PlansCalls        int64          `reform:"plans_calls"`
	PlanTotalTime     float64        `reform:"plan_total_time"`
	PlanMinTime       float64        `reform:"plan_min_time"`
	PlanMaxTime       float64        `reform:"plan_max_time"`
	PlanMeanTime      float64        `reform:"plan_mean_time"`
	SharedBlksHit     int64          `reform:"shared_blks_hit"`
	SharedBlksRead    int64          `reform:"shared_blks_read"`
	SharedBlksDirtied int64          `reform:"shared_blks_dirtied"`
	SharedBlksWritten int64          `reform:"shared_blks_written"`
	LocalBlksHit      int64          `reform:"local_blks_hit"`
	LocalBlksRead     int64          `reform:"local_blks_read"`
	LocalBlksDirtied  int64          `reform:"local_blks_dirtied"`
	LocalBlksWritten  int64          `reform:"local_blks_written"`
	TempBlksRead      int64          `reform:"temp_blks_read"`
	TempBlksWritten   int64          `reform:"temp_blks_written"`
	BlkReadTime       float64        `reform:"blk_read_time"`
	BlkWriteTime      float64        `reform:"blk_write_time"`
	RespCalls         pq.StringArray `reform:"resp_calls"`
	CPUUserTime       float64        `reform:"cpu_user_time"`
	CPUSysTime        float64        `reform:"cpu_sys_time"`
	WalRecords        int64          `reform:"wal_records"`
	WalFpi            int64          `reform:"wal_fpi"`
	WalBytes          int64          `reform:"wal_bytes"`
	StateCode         int64          `reform:"state_code"`
	State             string         `reform:"state"`
}

func (m pgStatMonitor09) ToPgStatMonitor() (pgStatMonitor, error) {
	bucketStartTime, err := time.Parse("2006-01-02 15:04:05", m.BucketStartTime)
	if err != nil {
		return pgStatMonitor{}, errors.Wrap(err, "cannot parse bucket start time")
	}

	return pgStatMonitor{
		Bucket:            m.Bucket,
		BucketStartTime:   bucketStartTime,
		User:              m.User,
		DatName:           m.DatName,
		QueryID:           m.QueryID,
		Query:             m.Query,
		Calls:             m.Calls,
		TotalTime:         m.TotalTime,
		Rows:              m.RowsRetrieved,
		SharedBlksHit:     m.SharedBlksHit,
		SharedBlksRead:    m.SharedBlksRead,
		SharedBlksDirtied: m.SharedBlksDirtied,
		SharedBlksWritten: m.SharedBlksWritten,
		LocalBlksHit:      m.LocalBlksHit,
		LocalBlksRead:     m.LocalBlksRead,
		LocalBlksDirtied:  m.LocalBlksDirtied,
		LocalBlksWritten:  m.LocalBlksWritten,
		TempBlksRead:      m.TempBlksRead,
		TempBlksWritten:   m.TempBlksWritten,
		BlkReadTime:       m.BlkReadTime,
		BlkWriteTime:      m.BlkWriteTime,
		ClientIP:          m.ClientIP,
		RespCalls:         m.RespCalls,
		CPUUserTime:       m.CPUUserTime,
		CPUSysTime:        m.CPUSysTime,
		Relations:         m.Relations,
		PlansCalls:        m.PlansCalls,
		WalFpi:            m.WalFpi,
		WalRecords:        m.WalRecords,
		WalBytes:          m.WalBytes,
		PlanTotalTime:     m.PlanTotalTime,
		PlanMinTime:       m.PlanMinTime,
		PlanMaxTime:       m.PlanMaxTime,
	}, nil
}

// pgStatMonitor represents a row in pg_stat_monitor view.
type pgStatMonitor struct {
	Bucket            int64
	BucketStartTime   time.Time
	UserID            int64
	User              string
	DBID              int64
	DatName           string
	QueryID           string
	Query             string
	Calls             int64
	TotalTime         float64
	Rows              int64
	SharedBlksHit     int64
	SharedBlksRead    int64
	SharedBlksDirtied int64
	SharedBlksWritten int64
	LocalBlksHit      int64
	LocalBlksRead     int64
	LocalBlksDirtied  int64
	LocalBlksWritten  int64
	TempBlksRead      int64
	TempBlksWritten   int64
	BlkReadTime       float64
	BlkWriteTime      float64
	ClientIP          string
	RespCalls         pq.StringArray
	CPUUserTime       float64
	CPUSysTime        float64
	Relations         pq.StringArray
	PlansCalls        int64
	WalFpi            int64
	WalRecords        int64
	WalBytes          int64
	PlanTotalTime     float64
	PlanMinTime       float64
	PlanMaxTime       float64
}

// pgStatMonitorSettings represents a row in pg_stat_monitor_settings view.
//reform:pg_stat_monitor_settings
type pgStatMonitorSettings struct {
	Name  string `reform:"name"`
	Value int64  `reform:"value"`
}

// pgStatMonitorExtended contains pgStatMonitor data and extends it with database, username and tables data.
// It's made for performance reason.
type pgStatMonitorExtended struct {
	pgStatMonitor

	Fingerprint      string
	Example          string
	Database         string
	Username         string
	IsQueryTruncated bool
}

func (e *pgStatMonitorExtended) String() string {
	return fmt.Sprintf("%q %q %v: %s: %s (truncated = %t)",
		e.Database, e.Username, e.Relations, e.QueryID, e.Query, e.IsQueryTruncated)
}
