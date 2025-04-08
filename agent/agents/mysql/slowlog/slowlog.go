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

// Package slowlog runs built-in QAN Agent for MySQL slow log.
package slowlog

import (
	"context"
	"crypto/md5" //nolint:gosec
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/go-mysql/event"
	"github.com/percona/go-mysql/log"
	"github.com/percona/go-mysql/query"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/mysql/shared"
	"github.com/percona/pmm/agent/agents/mysql/slowlog/parser"
	"github.com/percona/pmm/agent/queryparser"
	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/agent/utils/backoff"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

const (
	backoffMinDelay   = 1 * time.Second
	backoffMaxDelay   = 5 * time.Second
	recheckInterval   = 10 * time.Second
	aggregateInterval = time.Minute
)

// SlowLog extracts performance data from MySQL slow log.
type SlowLog struct {
	params  *Params
	l       *logrus.Entry
	changes chan agents.Change
}

// Params represent Agent parameters.
type Params struct {
	DSN                    string
	AgentID                string
	DisableCommentsParsing bool
	MaxQueryLength         int32
	DisableQueryExamples   bool
	MaxSlowlogFileSize     int64
	SlowLogFilePrefix      string // for development and testing
	TextFiles              *agentv1.TextFiles
	TLS                    bool
	TLSSkipVerify          bool
}

const queryTag = "agent='slowlog'"

type slowLogInfo struct {
	path        string
	outlierTime float64
}

// New creates new SlowLog QAN service.
func New(params *Params, l *logrus.Entry) (*SlowLog, error) {
	if params.TextFiles != nil {
		err := tlshelpers.RegisterMySQLCerts(params.TextFiles.Files, params.TLSSkipVerify)
		if err != nil {
			return nil, err
		}
	}

	return &SlowLog{
		params:  params,
		l:       l,
		changes: make(chan agents.Change, 10),
	}, nil
}

// Run extracts performance data and sends it to the channel until ctx is canceled.
func (s *SlowLog) Run(ctx context.Context) {
	defer func() {
		s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(s.changes)
	}()

	// send updates to fileInfos channel, close it when ctx is done
	fileInfos := make(chan *slowLogInfo, 1)
	go func() {
		recheck := time.NewTicker(recheckInterval)
		defer recheck.Stop()

		var oldInfo slowLogInfo
		for {
			newInfo := s.recheck(ctx)
			if newInfo != nil {
				if *newInfo != oldInfo {
					s.l.Debugf("Sloglow information changed: old = %+v, new = %+v.", oldInfo, *newInfo)
					fileInfos <- newInfo
					oldInfo = *newInfo
				} else {
					s.l.Tracef("Sloglow information not changed.")
				}
			}

			select {
			case <-ctx.Done():
				close(fileInfos)
				return
			case <-recheck.C:
				// nothing, continue loop
			}
		}
	}()

	b := backoff.New(backoffMinDelay, backoffMaxDelay)
	fileInfo := <-fileInfos
	for fileInfo != nil {
		s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

		// process file until fileCtx is done, or fatal processing error is encountered
		path, outlierTime := fileInfo.path, fileInfo.outlierTime
		fileCtx, fileCancel := context.WithCancel(ctx)
		fileDone := make(chan error)
		go func() {
			s.l.Infof("Processing file %s.", path)
			fileDone <- s.processFile(fileCtx, path, outlierTime)
		}()

		// cancel processing when new info is available, but always wait for it to finish
		var err error
		select {
		case fileInfo = <-fileInfos:
			fileCancel()
			err = <-fileDone
		case err = <-fileDone:
			fileCancel()
		}

		s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}

		if err == nil {
			b.Reset()
		} else {
			time.Sleep(b.Delay())
		}
	}
}

// recheck returns new slowlog information, and rotates slowlog file if needed.
func (s *SlowLog) recheck(ctx context.Context) *slowLogInfo {
	var newInfo *slowLogInfo

	db, err := sql.Open("mysql", s.params.DSN)
	if err != nil {
		s.l.Errorf("Cannot open database connection: %s", err)
		return nil
	}
	defer db.Close() //nolint:errcheck

	var grants string
	row := db.QueryRowContext(ctx, "SHOW GRANTS")
	if err := row.Scan(&grants); err != nil {
		s.l.Errorf("Cannot scan db user privileges: %s", err)
		return nil
	}

	if !strings.Contains(grants, "RELOAD") && !strings.Contains(grants, "ALL PRIVILEGES") {
		s.l.Error("RELOAD grant not enabled, cannot rotate slowlog")
		return nil
	}

	if newInfo, err = s.getSlowLogInfo(ctx); err != nil {
		s.l.Error(err)
		return nil
	}
	if s.params.SlowLogFilePrefix != "" {
		newInfo.path = filepath.Join(s.params.SlowLogFilePrefix, newInfo.path)
	}

	maxSize := s.params.MaxSlowlogFileSize
	if maxSize <= 0 {
		return newInfo
	}

	fi, err := os.Stat(newInfo.path)
	if err != nil {
		s.l.Errorf("Failed to stat file: %s", err)
		return newInfo
	}
	if size := fi.Size(); size > maxSize {
		s.l.Infof("Rotating slowlog file: %d > %d.", size, maxSize)
		if err = s.rotateSlowLog(ctx, newInfo.path); err != nil {
			s.l.Error(err)
		}
	}

	return newInfo
}

// getSlowLogInfo returns information about slowlog settings.
func (s *SlowLog) getSlowLogInfo(ctx context.Context) (*slowLogInfo, error) {
	db, err := sql.Open("mysql", s.params.DSN)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open database connection")
	}
	defer db.Close() //nolint:errcheck

	selectQuery := fmt.Sprintf("SELECT /* %s */ ", queryTag)
	var path string
	row := db.QueryRowContext(ctx, selectQuery+"@@slow_query_log_file")
	if err := row.Scan(&path); err != nil {
		return nil, errors.Wrap(err, "cannot select @@slow_query_log_file")
	}
	if path == "" {
		return nil, errors.New("cannot parse slowlog: @@slow_query_log_file is empty")
	}

	// Slow log file can be absolute or relative. If it's relative,
	// then prepend the datadir.
	if !filepath.IsAbs(path) {
		var dataDir string
		row = db.QueryRowContext(ctx, selectQuery+"@@datadir")
		if err := row.Scan(&dataDir); err != nil {
			return nil, errors.Wrap(err, "cannot select @@datadir")
		}
		path = filepath.Join(dataDir, path)
	}

	// Only @@slow_query_log_file is required, the rest global variables selected here
	// are optional and just help troubleshooting.

	// warn about disabled slowlog
	var enabled int
	row = db.QueryRowContext(ctx, selectQuery+"@@slow_query_log")
	if err := row.Scan(&enabled); err != nil {
		s.l.Warnf("Cannot SELECT @@slow_query_log: %s.", err)
	}
	if enabled != 1 {
		s.l.Warnf("@@slow_query_log is off: %v.", enabled)
	}

	// slow_query_log_always_write_time is Percona-specific, use debug level, not warning
	var outlierTime float64
	row = db.QueryRowContext(ctx, selectQuery+"@@slow_query_log_always_write_time")
	if err := row.Scan(&outlierTime); err != nil {
		s.l.Debugf("Cannot SELECT @@slow_query_log_always_write_time: %s.", err)
	}

	return &slowLogInfo{
		path:        path,
		outlierTime: outlierTime,
	}, nil
}

// rotateSlowLog removes slowlog file and calls FLUSH LOGS.
func (s *SlowLog) rotateSlowLog(ctx context.Context, slowLogPath string) error {
	db, err := sql.Open("mysql", s.params.DSN)
	if err != nil {
		return errors.Wrap(err, "cannot open database connection")
	}
	defer db.Close() //nolint:errcheck

	old := slowLogPath + ".old"
	if err = os.Remove(old); err != nil && !os.IsNotExist(err) {
		s.l.Warnf("Cannot remove previous old slowlog file: %s.", err)
	}

	// We have to rename slowlog file, not remove it, before flushing logs:
	// https://www.percona.com/blog/2007/12/09/be-careful-rotating-mysql-logs/
	// This problem is especially bad with MySQL in Docker - it locks completely even on small files.
	//
	// Reader will continue to read old file from open file descriptor until EOF.
	if err = os.Rename(slowLogPath, old); err != nil {
		return errors.Wrap(err, "cannot rename old slowlog file")
	}

	_, err = db.ExecContext(ctx, "FLUSH NO_WRITE_TO_BINLOG SLOW LOGS")
	if err != nil {
		return errors.Wrap(err, "cannot flush logs")
	}

	// keep one old file around, remove it on next iteration

	return nil
}

// processFile extracts performance data from given file and sends it to the channel until ctx is canceled.
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

	// send events to the channel, close it when parser is done
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

			if err := parser.Err(); !errors.Is(err, io.EOF) {
				s.l.Warnf("Parser error: %v.", err)
			}
			close(events)
			return
		}
	}()

	s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

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
			err = reader.Close() // that will let parser to stop
			s.l.Infof("Context done with %s. Reader closed with %v.", ctx.Err(), err)
			ctxDone = nil

		case e, ok := <-events:
			if !ok {
				// parser is done
				return nil
			}

			s.l.Tracef("Parsed slowlog event: %+v.", e)
			fingerprint := query.Fingerprint(e.Query)
			digest := hashIntoQueryID(fingerprint)
			aggregator.AddEvent(e, digest, e.User, e.Host, e.Db, e.Server, e.Query)

		case <-t.C:
			lengthS := uint32(math.Round(wait.Seconds())) // round 59.9s/60.1s to 60s
			res := aggregator.Finalize()
			buckets := makeBuckets(s.params.AgentID, res, start, lengthS, s.params.DisableCommentsParsing, s.params.DisableQueryExamples, s.params.MaxQueryLength, s.l)
			s.l.Debugf("Made %d buckets out of %d classes in %s+%d interval. Wait time: %s.",
				len(buckets), len(res.Class), start.Format("15:04:05"), lengthS, time.Since(start))

			aggregator = event.NewAggregator(true, 0, outlierTime)
			start = time.Now()
			wait = start.Truncate(aggregateInterval).Add(aggregateInterval).Sub(start)
			s.l.Debugf("Scheduling next aggregation in %s at %s.", wait, start.Add(wait).Format("15:04:05"))
			t.Reset(wait)

			s.changes <- agents.Change{MetricsBucket: buckets}
		}
	}
}

// hashIntoQueryID returns slowlog query ID hashed by MD5 from given fingerprint.
func hashIntoQueryID(fingerprint string) string {
	// MD5 is used only to hash fingerprint into query ID, so there is no risk.
	// It is ideal due to its length (32 chars) and it corresponds to Perfschema query ID length.
	id := md5.New() //nolint:gosec
	_, err := io.WriteString(id, fingerprint)
	if err != nil {
		logrus.Debugf("cannot hash fingerprint into query ID: %s", err.Error())
	}
	return strings.ToUpper(hex.EncodeToString(id.Sum(nil)))
}

// makeBuckets is a pure function for easier testing.
//
//nolint:cyclop,maintidx
func makeBuckets(
	agentID string,
	res event.Result,
	periodStart time.Time,
	periodLengthSecs uint32,
	disableCommentsParsing bool,
	disableQueryExamples bool,
	maxQueryLength int32,
	l *logrus.Entry,
) []*agentv1.MetricsBucket {
	buckets := make([]*agentv1.MetricsBucket, 0, len(res.Class))

	for _, v := range res.Class {
		if v.Metrics == nil {
			continue
		}

		// In fingerprint field there is no fingerprint yet.
		// It contains whole query without any changes.
		// This in workaround to keep original query until field "Query" will be
		// added here: https://github.com/percona/go-mysql/blob/v3/event/class.go#L56
		q := v.Fingerprint
		v.Fingerprint = query.Fingerprint(v.Fingerprint)
		fingerprint, isTruncated := truncate.Query(v.Fingerprint, maxQueryLength, truncate.GetDefaultMaxQueryLength())
		mb := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:              shared.QueryIDWithSchema(v.Db, v.Id),
				Fingerprint:          fingerprint,
				IsTruncated:          isTruncated,
				Database:             "",
				Schema:               v.Db,
				Username:             v.User,
				ClientHost:           v.Host,
				AgentId:              agentID,
				AgentType:            inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_SLOWLOG_AGENT,
				PeriodStartUnixSecs:  uint32(periodStart.Unix()),
				PeriodLengthSecs:     periodLengthSecs,
				NumQueries:           float32(v.TotalQueries),
				Errors:               errListsToMap(v.ErrorsCode, v.ErrorsCount),
				NumQueriesWithErrors: v.NumQueriesWithErrors,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{},
		}

		if q != "" {
			explainFingerprint, placeholdersCount := queryparser.GetMySQLFingerprintPlaceholders(q, fingerprint)
			explainFingerprint, truncated := truncate.Query(explainFingerprint, maxQueryLength, truncate.GetDefaultMaxQueryLength())
			if truncated {
				mb.Common.IsTruncated = truncated
			}
			mb.Common.ExplainFingerprint = explainFingerprint
			mb.Common.PlaceholdersCount = placeholdersCount
		}

		if !disableCommentsParsing {
			comments, err := queryparser.MySQLComments(q)
			if err != nil {
				l.Infof("cannot parse comments from query: %s", q)
			}
			mb.Common.Comments = comments
		}

		if v.Example != nil && !disableQueryExamples {
			example, truncated := truncate.Query(v.Example.Query, maxQueryLength, truncate.GetDefaultMaxQueryLength())
			if truncated {
				mb.Common.IsTruncated = truncated
			}
			mb.Common.Example = example
			mb.Common.ExampleType = agentv1.ExampleType_EXAMPLE_TYPE_RANDOM
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
			mb.Common.MQueryTimeCnt = float32(m.Cnt)
			mb.Common.MQueryTimeSum = float32(m.Sum)
			mb.Common.MQueryTimeMax = float32(*m.Max)
			mb.Common.MQueryTimeMin = float32(*m.Min)
			mb.Common.MQueryTimeP99 = float32(*m.P99)
		}
		// lock_time - Lock_time
		if m, ok := v.Metrics.TimeMetrics["Lock_time"]; ok {
			mb.Mysql.MLockTimeCnt = float32(m.Cnt)
			mb.Mysql.MLockTimeSum = float32(m.Sum)
			mb.Mysql.MLockTimeMax = float32(*m.Max)
			mb.Mysql.MLockTimeMin = float32(*m.Min)
			mb.Mysql.MLockTimeP99 = float32(*m.P99)
		}
		// rows_sent - Rows_sent
		if m, ok := v.Metrics.NumberMetrics["Rows_sent"]; ok {
			mb.Mysql.MRowsSentCnt = float32(m.Cnt)
			mb.Mysql.MRowsSentSum = float32(m.Sum)
			mb.Mysql.MRowsSentMax = float32(*m.Max)
			mb.Mysql.MRowsSentMin = float32(*m.Min)
			mb.Mysql.MRowsSentP99 = float32(*m.P99)
		}
		// rows_examined - Rows_examined
		if m, ok := v.Metrics.NumberMetrics["Rows_examined"]; ok {
			mb.Mysql.MRowsExaminedCnt = float32(m.Cnt)
			mb.Mysql.MRowsExaminedSum = float32(m.Sum)
			mb.Mysql.MRowsExaminedMax = float32(*m.Max)
			mb.Mysql.MRowsExaminedMin = float32(*m.Min)
			mb.Mysql.MRowsExaminedP99 = float32(*m.P99)
		}
		// rows_affected - Rows_affected
		if m, ok := v.Metrics.NumberMetrics["Rows_affected"]; ok {
			mb.Mysql.MRowsAffectedCnt = float32(m.Cnt)
			mb.Mysql.MRowsAffectedSum = float32(m.Sum)
			mb.Mysql.MRowsAffectedMax = float32(*m.Max)
			mb.Mysql.MRowsAffectedMin = float32(*m.Min)
			mb.Mysql.MRowsAffectedP99 = float32(*m.P99)
		}
		// rows_read - Rows_read
		if m, ok := v.Metrics.NumberMetrics["Rows_read"]; ok {
			mb.Mysql.MRowsReadCnt = float32(m.Cnt)
			mb.Mysql.MRowsReadSum = float32(m.Sum)
			mb.Mysql.MRowsReadMax = float32(*m.Max)
			mb.Mysql.MRowsReadMin = float32(*m.Min)
			mb.Mysql.MRowsReadP99 = float32(*m.P99)
		}
		// merge_passes - Merge_passes
		if m, ok := v.Metrics.NumberMetrics["Merge_passes"]; ok {
			mb.Mysql.MMergePassesCnt = float32(m.Cnt)
			mb.Mysql.MMergePassesSum = float32(m.Sum)
			mb.Mysql.MMergePassesMax = float32(*m.Max)
			mb.Mysql.MMergePassesMin = float32(*m.Min)
			mb.Mysql.MMergePassesP99 = float32(*m.P99)
		}
		// innodb_io_r_ops - InnoDB_IO_r_ops
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_ops"]; ok {
			mb.Mysql.MInnodbIoROpsCnt = float32(m.Cnt)
			mb.Mysql.MInnodbIoROpsSum = float32(m.Sum)
			mb.Mysql.MInnodbIoROpsMax = float32(*m.Max)
			mb.Mysql.MInnodbIoROpsMin = float32(*m.Min)
			mb.Mysql.MInnodbIoROpsP99 = float32(*m.P99)
		}
		// innodb_io_r_bytes - InnoDB_IO_r_bytes
		if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_bytes"]; ok {
			mb.Mysql.MInnodbIoRBytesCnt = float32(m.Cnt)
			mb.Mysql.MInnodbIoRBytesSum = float32(m.Sum)
			mb.Mysql.MInnodbIoRBytesMax = float32(*m.Max)
			mb.Mysql.MInnodbIoRBytesMin = float32(*m.Min)
			mb.Mysql.MInnodbIoRBytesP99 = float32(*m.P99)
		}
		// innodb_io_r_wait - InnoDB_IO_r_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_IO_r_wait"]; ok {
			mb.Mysql.MInnodbIoRWaitCnt = float32(m.Cnt)
			mb.Mysql.MInnodbIoRWaitSum = float32(m.Sum)
			mb.Mysql.MInnodbIoRWaitMax = float32(*m.Max)
			mb.Mysql.MInnodbIoRWaitMin = float32(*m.Min)
			mb.Mysql.MInnodbIoRWaitP99 = float32(*m.P99)
		}
		// innodb_rec_lock_wait - InnoDB_rec_lock_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_rec_lock_wait"]; ok {
			mb.Mysql.MInnodbRecLockWaitCnt = float32(m.Cnt)
			mb.Mysql.MInnodbRecLockWaitSum = float32(m.Sum)
			mb.Mysql.MInnodbRecLockWaitMax = float32(*m.Max)
			mb.Mysql.MInnodbRecLockWaitMin = float32(*m.Min)
			mb.Mysql.MInnodbRecLockWaitP99 = float32(*m.P99)
		}
		// innodb_queue_wait - InnoDB_queue_wait
		if m, ok := v.Metrics.TimeMetrics["InnoDB_queue_wait"]; ok {
			mb.Mysql.MInnodbQueueWaitCnt = float32(m.Cnt)
			mb.Mysql.MInnodbQueueWaitSum = float32(m.Sum)
			mb.Mysql.MInnodbQueueWaitMax = float32(*m.Max)
			mb.Mysql.MInnodbQueueWaitMin = float32(*m.Min)
			mb.Mysql.MInnodbQueueWaitP99 = float32(*m.P99)
		}
		// innodb_pages_distinct - InnoDB_pages_distinct
		if m, ok := v.Metrics.NumberMetrics["InnoDB_pages_distinct"]; ok {
			mb.Mysql.MInnodbPagesDistinctCnt = float32(m.Cnt)
			mb.Mysql.MInnodbPagesDistinctSum = float32(m.Sum)
			mb.Mysql.MInnodbPagesDistinctMax = float32(*m.Max)
			mb.Mysql.MInnodbPagesDistinctMin = float32(*m.Min)
			mb.Mysql.MInnodbPagesDistinctP99 = float32(*m.P99)
		}
		// query_length - Query_length
		if m, ok := v.Metrics.NumberMetrics["Query_length"]; ok {
			mb.Mysql.MQueryLengthCnt = float32(m.Cnt)
			mb.Mysql.MQueryLengthSum = float32(m.Sum)
			mb.Mysql.MQueryLengthMax = float32(*m.Max)
			mb.Mysql.MQueryLengthMin = float32(*m.Min)
			mb.Mysql.MQueryLengthP99 = float32(*m.P99)
		}
		// bytes_sent - Bytes_sent
		if m, ok := v.Metrics.NumberMetrics["Bytes_sent"]; ok {
			mb.Mysql.MBytesSentCnt = float32(m.Cnt)
			mb.Mysql.MBytesSentSum = float32(m.Sum)
			mb.Mysql.MBytesSentMax = float32(*m.Max)
			mb.Mysql.MBytesSentMin = float32(*m.Min)
			mb.Mysql.MBytesSentP99 = float32(*m.P99)
		}
		// tmp_tables - Tmp_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_tables"]; ok {
			mb.Mysql.MTmpTablesCnt = float32(m.Cnt)
			mb.Mysql.MTmpTablesSum = float32(m.Sum)
			mb.Mysql.MTmpTablesMax = float32(*m.Max)
			mb.Mysql.MTmpTablesMin = float32(*m.Min)
			mb.Mysql.MTmpTablesP99 = float32(*m.P99)
		}
		// tmp_disk_tables - Tmp_disk_tables
		if m, ok := v.Metrics.NumberMetrics["Tmp_disk_tables"]; ok {
			mb.Mysql.MTmpDiskTablesCnt = float32(m.Cnt)
			mb.Mysql.MTmpDiskTablesSum = float32(m.Sum)
			mb.Mysql.MTmpDiskTablesMax = float32(*m.Max)
			mb.Mysql.MTmpDiskTablesMin = float32(*m.Min)
			mb.Mysql.MTmpDiskTablesP99 = float32(*m.P99)
		}
		// tmp_table_sizes - Tmp_table_sizes
		if m, ok := v.Metrics.NumberMetrics["Tmp_table_sizes"]; ok {
			mb.Mysql.MTmpTableSizesCnt = float32(m.Cnt)
			mb.Mysql.MTmpTableSizesSum = float32(m.Sum)
			mb.Mysql.MTmpTableSizesMax = float32(*m.Max)
			mb.Mysql.MTmpTableSizesMin = float32(*m.Min)
			mb.Mysql.MTmpTableSizesP99 = float32(*m.P99)
		}
		// qc_hit - QC_Hit
		if m, ok := v.Metrics.BoolMetrics["QC_Hit"]; ok {
			mb.Mysql.MQcHitCnt = float32(m.Cnt)
			mb.Mysql.MQcHitSum = float32(m.Sum)
		}
		// full_scan - Full_scan
		if m, ok := v.Metrics.BoolMetrics["Full_scan"]; ok {
			mb.Mysql.MFullScanCnt = float32(m.Cnt)
			mb.Mysql.MFullScanSum = float32(m.Sum)
		}
		// full_join - Full_join
		if m, ok := v.Metrics.BoolMetrics["Full_join"]; ok {
			mb.Mysql.MFullJoinCnt = float32(m.Cnt)
			mb.Mysql.MFullJoinSum = float32(m.Sum)
		}
		// tmp_table - Tmp_table
		if m, ok := v.Metrics.BoolMetrics["Tmp_table"]; ok {
			mb.Mysql.MTmpTableCnt = float32(m.Cnt)
			mb.Mysql.MTmpTableSum = float32(m.Sum)
		}
		// tmp_table_on_disk - Tmp_table_on_disk
		if m, ok := v.Metrics.BoolMetrics["Tmp_table_on_disk"]; ok {
			mb.Mysql.MTmpTableOnDiskCnt = float32(m.Cnt)
			mb.Mysql.MTmpTableOnDiskSum = float32(m.Sum)
		}
		// filesort - Filesort
		if m, ok := v.Metrics.BoolMetrics["Filesort"]; ok {
			mb.Mysql.MFilesortCnt = float32(m.Cnt)
			mb.Mysql.MFilesortSum = float32(m.Sum)
		}
		// filesort_on_disk - Filesort_on_disk
		if m, ok := v.Metrics.BoolMetrics["Filesort_on_disk"]; ok {
			mb.Mysql.MFilesortOnDiskCnt = float32(m.Cnt)
			mb.Mysql.MFilesortOnDiskSum = float32(m.Sum)
		}
		// select_full_range_join - Select_full_range_join
		if m, ok := v.Metrics.BoolMetrics["Select_full_range_join"]; ok {
			mb.Mysql.MSelectFullRangeJoinCnt = float32(m.Cnt)
			mb.Mysql.MSelectFullRangeJoinSum = float32(m.Sum)
		}
		// select_range - Select_range
		if m, ok := v.Metrics.BoolMetrics["Select_range"]; ok {
			mb.Mysql.MSelectRangeCnt = float32(m.Cnt)
			mb.Mysql.MSelectRangeSum = float32(m.Sum)
		}
		// select_range_check - Select_range_check
		if m, ok := v.Metrics.BoolMetrics["Select_range_check"]; ok {
			mb.Mysql.MSelectRangeCheckCnt = float32(m.Cnt)
			mb.Mysql.MSelectRangeCheckSum = float32(m.Sum)
		}
		// sort_range - Sort_range
		if m, ok := v.Metrics.BoolMetrics["Sort_range"]; ok {
			mb.Mysql.MSortRangeCnt = float32(m.Cnt)
			mb.Mysql.MSortRangeSum = float32(m.Sum)
		}
		// sort_rows - Sort_rows
		if m, ok := v.Metrics.BoolMetrics["Sort_rows"]; ok {
			mb.Mysql.MSortRowsCnt = float32(m.Cnt)
			mb.Mysql.MSortRowsSum = float32(m.Sum)
		}
		// sort_scan - Sort_scan
		if m, ok := v.Metrics.BoolMetrics["Sort_scan"]; ok {
			mb.Mysql.MSortScanCnt = float32(m.Cnt)
			mb.Mysql.MSortScanSum = float32(m.Sum)
		}
		// no_index_used - No_index_used
		if m, ok := v.Metrics.BoolMetrics["No_index_used"]; ok {
			mb.Mysql.MNoIndexUsedCnt = float32(m.Cnt)
			mb.Mysql.MNoIndexUsedSum = float32(m.Sum)
		}
		// no_good_index_used - No_good_index_used
		if m, ok := v.Metrics.BoolMetrics["No_good_index_used"]; ok {
			mb.Mysql.MNoGoodIndexUsedCnt = float32(m.Cnt)
			mb.Mysql.MNoGoodIndexUsedSum = float32(m.Sum)
		}

		buckets = append(buckets, mb)
	}

	return buckets
}

// Changes returns channel that should be read until it is closed.
func (s *SlowLog) Changes() <-chan agents.Change {
	return s.changes
}

func errListsToMap(k, v []uint64) map[uint64]uint64 {
	m := make(map[uint64]uint64)
	for i, e := range k {
		m[e] = v[i]
	}
	return m
}

// Describe implements prometheus.Collector.
func (s *SlowLog) Describe(ch chan<- *prometheus.Desc) { //nolint:revive
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (s *SlowLog) Collect(ch chan<- prometheus.Metric) { //nolint:revive
	// This method is needed to satisfy interface.
}

// check interfaces.
var (
	_ prometheus.Collector = (*SlowLog)(nil)
)
