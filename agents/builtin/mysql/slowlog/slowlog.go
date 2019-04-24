// pmm-agent
// Copyright (C) 2018 Percona LLC
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

// Package slowlog runs built-in QAN Agent for MySQL slow log.
package slowlog

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/go-mysql/event"
	slowlog "github.com/percona/go-mysql/log"
	parser "github.com/percona/go-mysql/log/slow"
	"github.com/percona/go-mysql/query"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"
)

const (
	querySummaries = time.Minute
)

// SlowLog extracts performance data from MySQL slow log.
type SlowLog struct {
	db      *reform.DB
	agentID string
	l       *logrus.Entry
	changes chan Change
}

// Params represent Agent parameters.
type Params struct {
	DSN     string
	AgentID string
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

// New creates new MySQL QAN service.
func New(params *Params, l *logrus.Entry) (*SlowLog, error) {
	sqlDB, err := sql.Open("mysql", params.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(l.Tracef))

	return newMySQL(db, params.AgentID, l), nil
}

func newMySQL(db *reform.DB, agentID string, l *logrus.Entry) *SlowLog {
	return &SlowLog{
		db:      db,
		agentID: agentID,
		l:       l,
		changes: make(chan Change, 10),
	}
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (m *SlowLog) Run(ctx context.Context) {
	defer func() {
		m.db.DBInterface().(*sql.DB).Close() //nolint:errcheck
		m.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(m.changes)
	}()

	m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}

	slowLogFilePath, outlierTime, err := m.getSlowLogFilePath()
	if err != nil {
		m.l.Errorf("cannot get getSlowLogFilePath: %s", err)
		return
	}

	stat, err := os.Stat(slowLogFilePath)
	if err != nil {
		m.l.Errorf("cannot get stat of slowlog (%s): %s", slowLogFilePath, err)
		return
	}
	fileSize := uint64(stat.Size())

	opts := slowlog.Options{
		StartOffset: fileSize,
		Debug:       false,
		FilterAdminCommand: map[string]bool{
			"Binlog Dump":      true,
			"Binlog Dump GTID": true,
		},
	}

	running := true

	ticker := time.NewTicker(1 * querySummaries)

	slowLogParser, slowLogFileDescriptor := parseSlowLog(slowLogFilePath, opts, m.l)
	if slowLogParser == nil {
		return
	}
	defer func() {
		slowLogParser.Stop()
		if err = slowLogFileDescriptor.Close(); err != nil {
			m.l.Warn(err)
		}
	}()
	logEvent := slowLogParser.EventChan()
	aggregator := event.NewAggregator(true, 0, outlierTime)

	for {
		select {
		case <-ctx.Done():
			m.changes <- Change{Status: inventorypb.AgentStatus_STOPPING}
			m.l.Infof("Context canceled.")
			return

		case e := <-logEvent:
			if e == nil {
				continue
			}
			if !running {
				m.changes <- Change{Status: inventorypb.AgentStatus_STARTING}
			}
			m.l.Debugf("Parsed %v events in slowlog.", e)
			fingerprint := query.Fingerprint(e.Query)
			digest := query.Id(fingerprint)
			opts.StartOffset = e.OffsetEnd
			aggregator.AddEvent(e, digest, e.User, e.Host, e.Db, e.Server, fingerprint)

		case <-ticker.C:
			if !running {
				running = true
				m.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}
			}
			m.l.Debugln("Aggregating slowlog events.")
			res := aggregator.Finalize()

			// Check if MySQL SlowLog config is changed and slowlog rotated.
			curStat, err := os.Stat(slowLogFilePath)
			if err != nil {
				m.l.Errorf("cannot get stat of slowlog (%s): %s", slowLogFilePath, err)
				return
			}
			if !os.SameFile(stat, curStat) {
				opts.StartOffset = uint64(curStat.Size())
			}
			// Prepare fresh parser and aggregator for next iteration.
			slowLogParser.Stop()
			if err = slowLogFileDescriptor.Close(); err != nil {
				m.l.Warn(err)
			}
			slowLogParser, slowLogFileDescriptor = parseSlowLog(slowLogFilePath, opts, m.l)
			if slowLogParser == nil {
				return
			}
			logEvent = slowLogParser.EventChan()
			aggregator = event.NewAggregator(true, 0, outlierTime)

			buckets := makeBuckets(m.agentID, res, time.Now())
			lenBuckets := len(buckets)
			if lenBuckets == 0 {
				continue
			}
			m.l.Debugf("Collected %d buckets.", lenBuckets)
			m.changes <- Change{Request: &qanpb.CollectRequest{MetricsBucket: buckets}}

		default:
			running = false
			m.changes <- Change{Status: inventorypb.AgentStatus_WAITING}
		}
	}
}

// getSlowLogFilePath get path to MySQL slow log and check correct config.
func (m *SlowLog) getSlowLogFilePath() (string, float64, error) {
	var slowLogFilePath string
	row := m.db.QueryRow("SELECT @@slow_query_log_file")
	if err := row.Scan(&slowLogFilePath); err != nil {
		return "", 0, errors.Wrap(err, "cannot select @@slow_query_log_file")
	}
	if slowLogFilePath == "" {
		return "", 0, errors.New("cannot parse slowlog: @@slow_query_log_file is empty")
	}

	// Only @@slow_query_log is required, the rest global variables selected here
	// are optional and just help troubleshooting.

	var isSlowQueryLogON int
	row = m.db.QueryRow("SELECT @@slow_query_log")
	if err := row.Scan(&isSlowQueryLogON); err != nil {
		m.l.Warnf("cannot select @@slow_query_log: %s", err)
	}
	if isSlowQueryLogON != 1 {
		m.l.Warnf("@@slow_query_log is off: %v", isSlowQueryLogON)
	}

	// Select @@slow_query_log_always_write_time if this version of MySQL has it.
	var outlierTime float64
	row = m.db.QueryRow("SELECT @@slow_query_log_always_write_time")
	if err := row.Scan(&outlierTime); err != nil {
		m.l.Warnf("cannot select @@slow_query_log_always_write_time: %s", err)
	}

	slowLogFilePath = filepath.Clean(slowLogFilePath)
	m.l.Debugf("slowLogFilePath: %q, isSlowQueryLogON: %v, outlierTime: %v", slowLogFilePath, isSlowQueryLogON, outlierTime)
	return slowLogFilePath, outlierTime, nil
}

// parseSlowLog create new slow log parser.
func parseSlowLog(filename string, o slowlog.Options, l *logrus.Entry) (*parser.SlowLogParser, *os.File) {
	file, err := os.Open(filename) //nolint:gosec
	if err != nil {
		l.Errorf("Failed to open slowlog file %q: %s.", filename, err)
		return nil, nil
	}

	p := parser.NewSlowLogParser(file, o)
	go func() {
		if err = p.Start(); err != nil {
			l.Errorf("Failed to start slowlog parser: %s.", err)
		}
	}()
	return p, file
}

// makeBuckets is a pure function for easier testing.
func makeBuckets(agentID string, res event.Result, ts time.Time) []*qanpb.MetricsBucket {
	buckets := []*qanpb.MetricsBucket{}
	for _, v := range res.Class {
		mb := &qanpb.MetricsBucket{
			Queryid:             v.Id,
			Fingerprint:         v.Fingerprint,
			DDatabase:           "",
			DSchema:             v.Db,
			DUsername:           v.User,
			DClientHost:         v.Host,
			ServiceName:         v.Server,
			AgentId:             agentID,
			MetricsSource:       qanpb.MetricsSource_MYSQL_SLOWLOG,
			PeriodStartUnixSecs: uint32(ts.Truncate(1 * time.Minute).Unix()),
			PeriodLengthSecs:    uint32(60),
			Example:             v.Example.Query,
			ExampleFormat:       1,
			ExampleType:         1,
			NumQueries:          float32(v.TotalQueries),
		}

		// If key has suffix _time or _wait than field is TimeMetrics.
		// For Boolean metrics exists only Sum.
		// https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html
		// TimeMetrics: query_time, lock_time, rows_sent, innodb_io_r_wait, innodb_rec_lock_wait, innodb_queue_wait.
		// NumberMetrics: rows_examined, rows_affected, rows_read, merge_passes, innodb_io_r_ops, innodb_io_r_bytes,
		// innodb_pages_distinct, query_length, bytes_sent, tmp_tables, tmp_disk_tables, tmp_table_sizes.
		// BooleanMetrics: qc_hit, full_scan, full_join, tmp_table, tmp_table_on_disk, filesort, filesort_on_disk,
		// select_full_range_join, select_range, select_range_check, sort_range, sort_rows, sort_scan,
		// no_index_used, no_good_index_used.

		// query_time - Query_time
		if m, ok := v.Metrics.TimeMetrics["Query_time"]; ok {
			mb.MQueryTimeSum = float32(m.Sum)
			mb.MQueryTimeMax = float32(*m.Max)
			mb.MQueryTimeMin = float32(*m.Min)
			mb.MQueryTimeP99 = float32(*m.P99)
		}
		// lock_time - Lock_time
		if m, ok := v.Metrics.TimeMetrics["Lock_time"]; ok {
			mb.MLockTimeSum = float32(m.Sum)
			mb.MLockTimeMax = float32(*m.Max)
			mb.MLockTimeMin = float32(*m.Min)
			mb.MLockTimeP99 = float32(*m.P99)
		}
		// rows_sent - Rows_sent
		if m, ok := v.Metrics.NumberMetrics["Rows_sent"]; ok {
			mb.MRowsSentSum = float32(m.Sum)
			mb.MRowsSentMax = float32(*m.Max)
			mb.MRowsSentMin = float32(*m.Min)
			mb.MRowsSentP99 = float32(*m.P99)
		}
		// rows_examined - Rows_examined
		if m, ok := v.Metrics.NumberMetrics["Rows_examined"]; ok {
			mb.MRowsExaminedSum = float32(m.Sum)
			mb.MRowsExaminedMax = float32(*m.Max)
			mb.MRowsExaminedMin = float32(*m.Min)
			mb.MRowsExaminedP99 = float32(*m.P99)
		}
		// rows_affected - Rows_affected
		if m, ok := v.Metrics.NumberMetrics["Rows_affected"]; ok {
			mb.MRowsAffectedSum = float32(m.Sum)
			mb.MRowsAffectedMax = float32(*m.Max)
			mb.MRowsAffectedMin = float32(*m.Min)
			mb.MRowsAffectedP99 = float32(*m.P99)
		}
		// rows_read - Rows_read
		if m, ok := v.Metrics.NumberMetrics["Rows_read"]; ok {
			mb.MRowsReadSum = float32(m.Sum)
			mb.MRowsReadMax = float32(*m.Max)
			mb.MRowsReadMin = float32(*m.Min)
			mb.MRowsReadP99 = float32(*m.P99)
		}
		// merge_passes - Merge_passes
		if m, ok := v.Metrics.NumberMetrics["Merge_passes"]; ok {
			mb.MMergePassesSum = float32(m.Sum)
			mb.MMergePassesMax = float32(*m.Max)
			mb.MMergePassesMin = float32(*m.Min)
			mb.MMergePassesP99 = float32(*m.P99)
		}
		// innodb_io_r_ops - InnoDB_IO_r_ops
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_ops"]; ok {
			mb.MInnodbIoROpsSum = float32(m.Sum)
			mb.MInnodbIoROpsMax = float32(*m.Max)
			mb.MInnodbIoROpsMin = float32(*m.Min)
			mb.MInnodbIoROpsP99 = float32(*m.P99)
		}
		// innodb_io_r_bytes - InnoDB_IO_r_bytes
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_bytes"]; ok {
			mb.MInnodbIoRBytesSum = float32(m.Sum)
			mb.MInnodbIoRBytesMax = float32(*m.Max)
			mb.MInnodbIoRBytesMin = float32(*m.Min)
			mb.MInnodbIoRBytesP99 = float32(*m.P99)
		}
		// innodb_io_r_wait - InnoDB_IO_r_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_IO_r_wait"]; ok {
			mb.MInnodbIoRWaitSum = float32(m.Sum)
			mb.MInnodbIoRWaitMax = float32(*m.Max)
			mb.MInnodbIoRWaitMin = float32(*m.Min)
			mb.MInnodbIoRWaitP99 = float32(*m.P99)
		}
		// innodb_rec_lock_wait - InnoDB_rec_lock_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_rec_lock_wait"]; ok {
			mb.MInnodbRecLockWaitSum = float32(m.Sum)
			mb.MInnodbRecLockWaitMax = float32(*m.Max)
			mb.MInnodbRecLockWaitMin = float32(*m.Min)
			mb.MInnodbRecLockWaitP99 = float32(*m.P99)
		}
		// innodb_queue_wait - InnoDB_queue_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_queue_wait"]; ok {
			mb.MInnodbQueueWaitSum = float32(m.Sum)
			mb.MInnodbQueueWaitMax = float32(*m.Max)
			mb.MInnodbQueueWaitMin = float32(*m.Min)
			mb.MInnodbQueueWaitP99 = float32(*m.P99)
		}
		// innodb_pages_distinct - InnoDB_pages_distinct
		if m, ok := v.Metrics.NumberMetrics["InnoDB_pages_distinct"]; ok {
			mb.MInnodbPagesDistinctSum = float32(m.Sum)
			mb.MInnodbPagesDistinctMax = float32(*m.Max)
			mb.MInnodbPagesDistinctMin = float32(*m.Min)
			mb.MInnodbPagesDistinctP99 = float32(*m.P99)
		}
		// query_length - Query_length
		if m, ok := v.Metrics.NumberMetrics["Query_length"]; ok {
			mb.MQueryLengthSum = float32(m.Sum)
			mb.MQueryLengthMax = float32(*m.Max)
			mb.MQueryLengthMin = float32(*m.Min)
			mb.MQueryLengthP99 = float32(*m.P99)
		}
		// bytes_sent - Bytes_sent
		if m, ok := v.Metrics.NumberMetrics["Bytes_sent"]; ok {
			mb.MBytesSentSum = float32(m.Sum)
			mb.MBytesSentMax = float32(*m.Max)
			mb.MBytesSentMin = float32(*m.Min)
			mb.MBytesSentP99 = float32(*m.P99)
		}
		// tmp_tables - Tmp_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_tables"]; ok {
			mb.MTmpTablesSum = float32(m.Sum)
			mb.MTmpTablesMax = float32(*m.Max)
			mb.MTmpTablesMin = float32(*m.Min)
			mb.MTmpTablesP99 = float32(*m.P99)
		}
		// tmp_disk_tables - Tmp_disk_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_disk_tables"]; ok {
			mb.MTmpDiskTablesSum = float32(m.Sum)
			mb.MTmpDiskTablesMax = float32(*m.Max)
			mb.MTmpDiskTablesMin = float32(*m.Min)
			mb.MTmpDiskTablesP99 = float32(*m.P99)
		}
		// tmp_table_sizes - Tmp_table_sizes
		if m, ok := v.Metrics.NumberMetrics["Tmp_table_sizes"]; ok {
			mb.MTmpTableSizesSum = float32(m.Sum)
			mb.MTmpTableSizesMax = float32(*m.Max)
			mb.MTmpTableSizesMin = float32(*m.Min)
			mb.MTmpTableSizesP99 = float32(*m.P99)
		}
		// qc_hit - QC_Hit
		if m, ok := v.Metrics.BoolMetrics["QC_Hit"]; ok {
			mb.MQcHitCnt = float32(m.Cnt)
			mb.MQcHitSum = float32(m.Sum)
		}
		// full_scan - Full_scan
		if m, ok := v.Metrics.BoolMetrics["Full_scan"]; ok {
			mb.MFullScanCnt = float32(m.Cnt)
			mb.MFullScanSum = float32(m.Sum)
		}
		// full_join - Full_join
		if m, ok := v.Metrics.BoolMetrics["Full_join"]; ok {
			mb.MFullJoinCnt = float32(m.Cnt)
			mb.MFullJoinSum = float32(m.Sum)
		}
		// tmp_table - Tmp_table
		if m, ok := v.Metrics.BoolMetrics["Tmp_table"]; ok {
			mb.MTmpTableCnt = float32(m.Cnt)
			mb.MTmpTableSum = float32(m.Sum)
		}
		// tmp_table_on_disk - Tmp_table_on_disk
		if m, ok := v.Metrics.BoolMetrics["Tmp_table_on_disk"]; ok {
			mb.MTmpTableOnDiskCnt = float32(m.Cnt)
			mb.MTmpTableOnDiskSum = float32(m.Sum)
		}
		// filesort - Filesort
		if m, ok := v.Metrics.BoolMetrics["Filesort"]; ok {
			mb.MFilesortCnt = float32(m.Cnt)
			mb.MFilesortSum = float32(m.Sum)
		}
		// filesort_on_disk - Filesort_on_disk
		if m, ok := v.Metrics.BoolMetrics["Filesort_on_disk"]; ok {
			mb.MFilesortOnDiskCnt = float32(m.Cnt)
			mb.MFilesortOnDiskSum = float32(m.Sum)
		}
		// select_full_range_join - Select_full_range_join
		if m, ok := v.Metrics.BoolMetrics["Select_full_range_join"]; ok {
			mb.MSelectFullRangeJoinCnt = float32(m.Cnt)
			mb.MSelectFullRangeJoinSum = float32(m.Sum)
		}
		// select_range - Select_range
		if m, ok := v.Metrics.BoolMetrics["Select_range"]; ok {
			mb.MSelectRangeCnt = float32(m.Cnt)
			mb.MSelectRangeSum = float32(m.Sum)
		}
		// select_range_check - Select_range_check
		if m, ok := v.Metrics.BoolMetrics["Select_range_check"]; ok {
			mb.MSelectRangeCheckCnt = float32(m.Cnt)
			mb.MSelectRangeCheckSum = float32(m.Sum)
		}
		// sort_range - Sort_range
		if m, ok := v.Metrics.BoolMetrics["Sort_range"]; ok {
			mb.MSortRangeCnt = float32(m.Cnt)
			mb.MSortRangeSum = float32(m.Sum)
		}
		// sort_rows - Sort_rows
		if m, ok := v.Metrics.BoolMetrics["Sort_rows"]; ok {
			mb.MSortRowsCnt = float32(m.Cnt)
			mb.MSortRowsSum = float32(m.Sum)
		}
		// sort_scan - Sort_scan
		if m, ok := v.Metrics.BoolMetrics["Sort_scan"]; ok {
			mb.MSortScanCnt = float32(m.Cnt)
			mb.MSortScanSum = float32(m.Sum)
		}
		// no_index_used - No_index_used
		if m, ok := v.Metrics.BoolMetrics["No_index_used"]; ok {
			mb.MNoIndexUsedCnt = float32(m.Cnt)
			mb.MNoIndexUsedSum = float32(m.Sum)
		}
		// no_good_index_used - No_good_index_used
		if m, ok := v.Metrics.BoolMetrics["No_good_index_used"]; ok {
			mb.MNoGoodIndexUsedCnt = float32(m.Cnt)
			mb.MNoGoodIndexUsedSum = float32(m.Sum)
		}

		buckets = append(buckets, mb)
	}
	return buckets
}

// Changes returns channel that should be read until it is closed.
func (m *SlowLog) Changes() <-chan Change {
	return m.changes
}
