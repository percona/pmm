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
	"io"
	"math"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/go-mysql/event"
	"github.com/percona/go-mysql/log"
	"github.com/percona/go-mysql/query"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/agents/mysql/slowlog/parser"
	"github.com/percona/pmm-agent/utils/backoff"
)

const (
	backoffMinDelay   = 1 * time.Second
	backoffMaxDelay   = 5 * time.Second
	recheckInterval   = 5 * time.Second
	aggregateInterval = time.Minute
)

// SlowLog extracts performance data from MySQL slow log.
type SlowLog struct {
	dsn               string
	agentID           string
	slowLogFilePrefix string
	l                 *logrus.Entry
	changes           chan Change
}

// Params represent Agent parameters.
type Params struct {
	DSN               string
	AgentID           string
	SlowLogFilePrefix string // for development and testing
}

// Change represents Agent status change _or_ QAN collect request.
type Change struct {
	Status  inventorypb.AgentStatus
	Request *qanpb.CollectRequest
}

type slowLogInfo struct {
	path        string
	outlierTime float64
}

// New creates new SlowLog QAN service.
func New(params *Params, l *logrus.Entry) (*SlowLog, error) {
	return &SlowLog{
		dsn:               params.DSN,
		agentID:           params.AgentID,
		slowLogFilePrefix: params.SlowLogFilePrefix,
		l:                 l,
		changes:           make(chan Change, 10),
	}, nil
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (s *SlowLog) Run(ctx context.Context) {
	defer func() {
		s.changes <- Change{Status: inventorypb.AgentStatus_DONE}
		close(s.changes)
	}()

	// send updates to fileInfos channel, close it when ctx is done
	fileInfos := make(chan *slowLogInfo, 1)
	go func() {
		var oldInfo slowLogInfo
		for {
			newInfo, err := s.getSlowLogInfo(ctx)
			if err == nil {
				if s.slowLogFilePrefix != "" {
					newInfo.path = filepath.Join(s.slowLogFilePrefix, newInfo.path)
				}
				if oldInfo == *newInfo {
					s.l.Tracef("Sloglow information not changed.")
				} else {
					s.l.Debugf("Sloglow information changed: old = %+v, new = %+v.", oldInfo, *newInfo)
					fileInfos <- newInfo
					oldInfo = *newInfo
				}
			} else {
				s.l.Error(err)
			}

			select {
			case <-ctx.Done():
				close(fileInfos)
				return
			case <-time.Tick(recheckInterval):
				// nothing, continue loop
			}
		}
	}()

	b := backoff.New(backoffMinDelay, backoffMaxDelay)
	fileInfo := <-fileInfos
	for fileInfo != nil {
		s.changes <- Change{Status: inventorypb.AgentStatus_STARTING}

		// process file until fileCtx is done, or fatal processing error is encountered
		fileCtx, fileCancel := context.WithCancel(ctx)
		fileDone := make(chan error)
		go func() {
			s.l.Infof("Processing file %s.", fileInfo.path)
			fileDone <- s.processFile(fileCtx, fileInfo.path, fileInfo.outlierTime)
		}()

		// cancel processing when new info is available, but always wait for it to finish
		var err error
		select {
		case fileInfo = <-fileInfos:
			fileCancel()
			err = <-fileDone
		case err = <-fileDone:
		}

		s.changes <- Change{Status: inventorypb.AgentStatus_WAITING}

		if err == nil {
			b.Reset()
		} else {
			time.Sleep(b.Delay())
		}
	}
}

// getSlowLogInfo returns information about slowlog settings.
func (s *SlowLog) getSlowLogInfo(ctx context.Context) (*slowLogInfo, error) {
	db, err := sql.Open("mysql", s.dsn)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open database connection")
	}
	defer db.Close() //nolint:errcheck

	var path string
	row := db.QueryRowContext(ctx, "SELECT @@slow_query_log_file")
	if err := row.Scan(&path); err != nil {
		return nil, errors.Wrap(err, "cannot select @@slow_query_log_file")
	}
	if path == "" {
		return nil, errors.New("cannot parse slowlog: @@slow_query_log_file is empty")
	}

	// Only @@slow_query_log_file is required, the rest global variables selected here
	// are optional and just help troubleshooting.

	// warn about disabled slowlog
	var enabled int
	row = db.QueryRowContext(ctx, "SELECT @@slow_query_log")
	if err := row.Scan(&enabled); err != nil {
		s.l.Warnf("Cannot SELECT @@slow_query_log: %s.", err)
	}
	if enabled != 1 {
		s.l.Warnf("@@slow_query_log is off: %v.", enabled)
	}

	// slow_query_log_always_write_time is Percona-specific, use debug level, not warning
	var outlierTime float64
	row = db.QueryRowContext(ctx, "SELECT @@slow_query_log_always_write_time")
	if err := row.Scan(&outlierTime); err != nil {
		s.l.Debugf("Cannot SELECT @@slow_query_log_always_write_time: %s.", err)
	}

	return &slowLogInfo{
		path:        path,
		outlierTime: outlierTime,
	}, nil
}

// processFile extracts performance data from given file and sends it to the channel until ctx is canceled,
// or fatal error is encountered.
func (s *SlowLog) processFile(ctx context.Context, file string, outlierTime float64) error {
	rl := s.l.WithField("component", "slowlog/reader").WithField("file", file)
	reader, err := parser.NewContinuousFileReader(file, rl)
	if err != nil {
		s.l.Errorf("Failed to start reader for file %s: %s.", file, err)
		return err
	}

	opts := log.Options{
		FilterAdminCommand: map[string]bool{
			"Binlog Dump":      true,
			"Binlog Dump GTID": true,
		},
	}
	if s.l.Logger.GetLevel() == logrus.TraceLevel {
		opts.Debug = true
		opts.Debugf = s.l.WithField("component", "slowlog/parser").WithField("file", file).Tracef
	}

	parser := parser.NewSlowLogParser(reader, opts)
	go parser.Run()
	events := make(chan *log.Event, 1000)
	go func() {
		for {
			event := parser.Parse()
			if event != nil {
				events <- event
				continue
			}

			if err := parser.Err(); err != io.EOF {
				s.l.Warnf("Parser error: %v.", err)
			}
			close(events)
			return
		}
	}()

	s.changes <- Change{Status: inventorypb.AgentStatus_RUNNING}

	aggregator := event.NewAggregator(true, 0, outlierTime)
	ctxDone := ctx.Done()

	// aggregate every minute at 00 seconds
	start := time.Now()
	wait := start.Truncate(aggregateInterval).Add(aggregateInterval).Sub(start)
	s.l.Debugf("Scheduling next aggregation in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
	t := time.NewTimer(wait)
	defer t.Stop()

	for {
		select {
		case <-ctxDone:
			err = reader.Close()
			s.l.Infof("Context done with %s. Reader closed with %v.", ctx.Err(), err)
			ctxDone = nil

		case e, ok := <-events:
			if !ok {
				return nil
			}

			s.l.Tracef("Parsed slowlog event: %+v.", e)
			fingerprint := query.Fingerprint(e.Query)
			digest := query.Id(fingerprint)
			aggregator.AddEvent(e, digest, e.User, e.Host, e.Db, e.Server, fingerprint)

		case <-t.C:
			lengthS := uint32(math.Round(wait.Seconds())) // round 59.9s/60.1s to 60s
			res := aggregator.Finalize()
			buckets := makeBuckets(s.agentID, res, start, lengthS)
			s.l.Debugf("Made %d buckets out of %d classes in %s+%d interval. Wait time: %s.",
				len(buckets), len(res.Class), start.Format("15:04:05"), lengthS, time.Since(start))

			aggregator = event.NewAggregator(true, 0, outlierTime)
			start = time.Now()
			wait = start.Truncate(aggregateInterval).Add(aggregateInterval).Sub(start)
			s.l.Debugf("Scheduling next aggregation in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
			t.Reset(wait)

			s.changes <- Change{Request: &qanpb.CollectRequest{MetricsBucket: buckets}}
		}
	}
}

// makeBuckets is a pure function for easier testing.
func makeBuckets(agentID string, res event.Result, periodStart time.Time, periodLengthSecs uint32) []*qanpb.MetricsBucket {
	buckets := make([]*qanpb.MetricsBucket, 0, len(res.Class))

	for _, v := range res.Class {
		mb := &qanpb.MetricsBucket{
			Queryid:              v.Id,
			Fingerprint:          v.Fingerprint,
			Database:             "",
			Schema:               v.Db,
			Username:             v.User,
			ClientHost:           v.Host,
			AgentId:              agentID,
			AgentType:            inventorypb.AgentType_QAN_MYSQL_SLOWLOG_AGENT,
			PeriodStartUnixSecs:  uint32(periodStart.Unix()),
			PeriodLengthSecs:     periodLengthSecs,
			Example:              v.Example.Query,
			ExampleFormat:        qanpb.ExampleFormat_EXAMPLE,
			ExampleType:          qanpb.ExampleType_RANDOM,
			NumQueries:           float32(v.TotalQueries),
			Errors:               errListsToMap(v.ErrorsCode, v.ErrorsCount),
			NumQueriesWithErrors: v.NumQueriesWithErrors,
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
			mb.MQueryTimeCnt = float32(m.Cnt)
			mb.MQueryTimeSum = float32(m.Sum)
			mb.MQueryTimeMax = float32(*m.Max)
			mb.MQueryTimeMin = float32(*m.Min)
			mb.MQueryTimeP99 = float32(*m.P99)
		}
		// lock_time - Lock_time
		if m, ok := v.Metrics.TimeMetrics["Lock_time"]; ok {
			mb.MLockTimeCnt = float32(m.Cnt)
			mb.MLockTimeSum = float32(m.Sum)
			mb.MLockTimeMax = float32(*m.Max)
			mb.MLockTimeMin = float32(*m.Min)
			mb.MLockTimeP99 = float32(*m.P99)
		}
		// rows_sent - Rows_sent
		if m, ok := v.Metrics.NumberMetrics["Rows_sent"]; ok {
			mb.MRowsSentCnt = float32(m.Cnt)
			mb.MRowsSentSum = float32(m.Sum)
			mb.MRowsSentMax = float32(*m.Max)
			mb.MRowsSentMin = float32(*m.Min)
			mb.MRowsSentP99 = float32(*m.P99)
		}
		// rows_examined - Rows_examined
		if m, ok := v.Metrics.NumberMetrics["Rows_examined"]; ok {
			mb.MRowsExaminedCnt = float32(m.Cnt)
			mb.MRowsExaminedSum = float32(m.Sum)
			mb.MRowsExaminedMax = float32(*m.Max)
			mb.MRowsExaminedMin = float32(*m.Min)
			mb.MRowsExaminedP99 = float32(*m.P99)
		}
		// rows_affected - Rows_affected
		if m, ok := v.Metrics.NumberMetrics["Rows_affected"]; ok {
			mb.MRowsAffectedCnt = float32(m.Cnt)
			mb.MRowsAffectedSum = float32(m.Sum)
			mb.MRowsAffectedMax = float32(*m.Max)
			mb.MRowsAffectedMin = float32(*m.Min)
			mb.MRowsAffectedP99 = float32(*m.P99)
		}
		// rows_read - Rows_read
		if m, ok := v.Metrics.NumberMetrics["Rows_read"]; ok {
			mb.MRowsReadCnt = float32(m.Cnt)
			mb.MRowsReadSum = float32(m.Sum)
			mb.MRowsReadMax = float32(*m.Max)
			mb.MRowsReadMin = float32(*m.Min)
			mb.MRowsReadP99 = float32(*m.P99)
		}
		// merge_passes - Merge_passes
		if m, ok := v.Metrics.NumberMetrics["Merge_passes"]; ok {
			mb.MMergePassesCnt = float32(m.Cnt)
			mb.MMergePassesSum = float32(m.Sum)
			mb.MMergePassesMax = float32(*m.Max)
			mb.MMergePassesMin = float32(*m.Min)
			mb.MMergePassesP99 = float32(*m.P99)
		}
		// innodb_io_r_ops - InnoDB_IO_r_ops
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_ops"]; ok {
			mb.MInnodbIoROpsCnt = float32(m.Cnt)
			mb.MInnodbIoROpsSum = float32(m.Sum)
			mb.MInnodbIoROpsMax = float32(*m.Max)
			mb.MInnodbIoROpsMin = float32(*m.Min)
			mb.MInnodbIoROpsP99 = float32(*m.P99)
		}
		// innodb_io_r_bytes - InnoDB_IO_r_bytes
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_bytes"]; ok {
			mb.MInnodbIoRBytesCnt = float32(m.Cnt)
			mb.MInnodbIoRBytesSum = float32(m.Sum)
			mb.MInnodbIoRBytesMax = float32(*m.Max)
			mb.MInnodbIoRBytesMin = float32(*m.Min)
			mb.MInnodbIoRBytesP99 = float32(*m.P99)
		}
		// innodb_io_r_wait - InnoDB_IO_r_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_IO_r_wait"]; ok {
			mb.MInnodbIoRWaitCnt = float32(m.Cnt)
			mb.MInnodbIoRWaitSum = float32(m.Sum)
			mb.MInnodbIoRWaitMax = float32(*m.Max)
			mb.MInnodbIoRWaitMin = float32(*m.Min)
			mb.MInnodbIoRWaitP99 = float32(*m.P99)
		}
		// innodb_rec_lock_wait - InnoDB_rec_lock_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_rec_lock_wait"]; ok {
			mb.MInnodbRecLockWaitCnt = float32(m.Cnt)
			mb.MInnodbRecLockWaitSum = float32(m.Sum)
			mb.MInnodbRecLockWaitMax = float32(*m.Max)
			mb.MInnodbRecLockWaitMin = float32(*m.Min)
			mb.MInnodbRecLockWaitP99 = float32(*m.P99)
		}
		// innodb_queue_wait - InnoDB_queue_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_queue_wait"]; ok {
			mb.MInnodbQueueWaitCnt = float32(m.Cnt)
			mb.MInnodbQueueWaitSum = float32(m.Sum)
			mb.MInnodbQueueWaitMax = float32(*m.Max)
			mb.MInnodbQueueWaitMin = float32(*m.Min)
			mb.MInnodbQueueWaitP99 = float32(*m.P99)
		}
		// innodb_pages_distinct - InnoDB_pages_distinct
		if m, ok := v.Metrics.NumberMetrics["InnoDB_pages_distinct"]; ok {
			mb.MInnodbPagesDistinctCnt = float32(m.Cnt)
			mb.MInnodbPagesDistinctSum = float32(m.Sum)
			mb.MInnodbPagesDistinctMax = float32(*m.Max)
			mb.MInnodbPagesDistinctMin = float32(*m.Min)
			mb.MInnodbPagesDistinctP99 = float32(*m.P99)
		}
		// query_length - Query_length
		if m, ok := v.Metrics.NumberMetrics["Query_length"]; ok {
			mb.MQueryLengthCnt = float32(m.Cnt)
			mb.MQueryLengthSum = float32(m.Sum)
			mb.MQueryLengthMax = float32(*m.Max)
			mb.MQueryLengthMin = float32(*m.Min)
			mb.MQueryLengthP99 = float32(*m.P99)
		}
		// bytes_sent - Bytes_sent
		if m, ok := v.Metrics.NumberMetrics["Bytes_sent"]; ok {
			mb.MBytesSentCnt = float32(m.Cnt)
			mb.MBytesSentSum = float32(m.Sum)
			mb.MBytesSentMax = float32(*m.Max)
			mb.MBytesSentMin = float32(*m.Min)
			mb.MBytesSentP99 = float32(*m.P99)
		}
		// tmp_tables - Tmp_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_tables"]; ok {
			mb.MTmpTablesCnt = float32(m.Cnt)
			mb.MTmpTablesSum = float32(m.Sum)
			mb.MTmpTablesMax = float32(*m.Max)
			mb.MTmpTablesMin = float32(*m.Min)
			mb.MTmpTablesP99 = float32(*m.P99)
		}
		// tmp_disk_tables - Tmp_disk_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_disk_tables"]; ok {
			mb.MTmpDiskTablesCnt = float32(m.Cnt)
			mb.MTmpDiskTablesSum = float32(m.Sum)
			mb.MTmpDiskTablesMax = float32(*m.Max)
			mb.MTmpDiskTablesMin = float32(*m.Min)
			mb.MTmpDiskTablesP99 = float32(*m.P99)
		}
		// tmp_table_sizes - Tmp_table_sizes
		if m, ok := v.Metrics.NumberMetrics["Tmp_table_sizes"]; ok {
			mb.MTmpTableSizesCnt = float32(m.Cnt)
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
func (s *SlowLog) Changes() <-chan Change {
	return s.changes
}

func errListsToMap(k, v []uint64) map[uint64]uint64 {
	m := map[uint64]uint64{}
	for i, e := range k {
		m[e] = v[i]
	}
	return m
}
