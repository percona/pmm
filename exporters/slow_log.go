// qan-api
// Copyright (C) 2019 Percona LLC
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

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	pb "github.com/Percona-Lab/qan-api/api/collector"
	"github.com/percona/go-mysql/event"
	slowlog "github.com/percona/go-mysql/log"
	parser "github.com/percona/go-mysql/log/slow"
	"github.com/percona/go-mysql/query"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const agentUUID = "dc889ca7be92a66f0a00f616f69ffa7b"
const dbServer = "fb_db"

type closedChannelError struct {
	error
}

type QueryClassDimentions struct {
	DbUsername  string
	ClientHost  string
	PeriodStart int64
	PeriodEnd   int64
}

func main() {
	dbs := []string{"db0", "db1", "db2", "db3", "db4", "db5", "db6", "db7", "db8", "db9"}
	rand.Seed(time.Now().UnixNano())

	logOpt := slowlog.Options{}
	slowLogPath := flag.String("slow-log", "logs/mysql-slow.log", "Path to MySQL slow log file")
	serverURL := flag.String("server-url", "127.0.0.1:80", "ULR of QAN-API Server")
	offset := flag.Uint64("offset", 0, "Start Offset of slowlog")

	maxQCtoSent := flag.Int("max-qc-to-sent", 100000, "Maximum query classes  to sent to QAN-API.")
	maxTimeForSent := flag.Duration("max-time-for-tx", 5*time.Second, "Maximum time to send .")
	newEventWait := flag.Duration("new-event-wait", 10*time.Second, "Time to wait for a new event in slow log.")
	flag.Parse()
	logOpt.StartOffset = *offset

	log.SetOutput(os.Stderr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(*serverURL, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	client := pb.NewAgentClient(conn)

	events := parseSlowLog(*slowLogPath, logOpt)

	ctx := context.TODO()
	stream, err := client.DataInterchange(ctx)
	if err != nil {
		log.Fatalf("%v.DataInterchange(), %v", client, err)
	}

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				continue
			}
			log.Printf("Got message from Api. Saved %d query class(es)", in.SavedAmount)
		}
	}()

	events = parseSlowLog(*slowLogPath, logOpt)
	fmt.Println("Parsing slowlog: ", *slowLogPath, "...")

	for {
		start := time.Now()
		err = bulkSend(stream, func(am *pb.AgentMessage) error {
			i := 0
			aggregator := event.NewAggregator(true, 0, 1) // add right params
			qcDimentions := map[string]*QueryClassDimentions{}
			for e := range events {
				fingerprint := query.Fingerprint(e.Query)
				digest := query.Id(fingerprint)
				aggregator.AddEvent(e, digest, fingerprint)
				// Pass last offset to restart reader when reached out end of slowlog.
				logOpt.StartOffset = e.OffsetEnd

				qcd := &QueryClassDimentions{
					DbUsername: e.User,
					ClientHost: e.Host,
					PeriodEnd:  e.Ts.UnixNano(),
				}

				qcDimentions[digest] = qcd
				if qcDimentions[digest].PeriodStart != 0 {
					qcDimentions[digest].PeriodStart = e.Ts.UnixNano()
				}

				i++
				if i >= *maxQCtoSent || time.Since(start) > *maxTimeForSent {

					fmt.Printf("offset: %v\n", logOpt.StartOffset)
					break
				}
			}

			r := aggregator.Finalize()

			for k, v := range r.Class {
				n := rand.Intn(9)
				labels := map[string]string{}
				for i := 1; i <= n; i++ {
					labels[fmt.Sprintf("key%v", rand.Intn(9))] = fmt.Sprintf("label%v", rand.Intn(9))
				}

				qc := &pb.QueryClass{
					Digest:     k,
					DigestText: v.Fingerprint,
					DbSchema:   dbs[rand.Intn(9)], // fake data
					DbUsername: qcDimentions[k].DbUsername,
					ClientHost: fmt.Sprintf("192.168.1.%v", rand.Intn(99)), // fake data
					// ClientHost:   qcDimentions[k].ClientHost,
					DbServer:     fmt.Sprintf("hostname_%v", rand.Intn(99)), // fake data
					Labels:       labels,
					AgentUuid:    agentUUID,
					PeriodStart:  qcDimentions[k].PeriodStart,
					PeriodLength: uint32(qcDimentions[k].PeriodStart - qcDimentions[k].PeriodStart),
					Example:      v.Example.Query,
					NumQueries:   uint64(v.TotalQueries),
				}

				t, _ := time.Parse("2006-01-02 15:04:05", v.Example.Ts)
				qc.PeriodStart = t.Unix()

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
					qc.MQueryTimeSum = float32(m.Sum)
					qc.MQueryTimeMax = float32(*m.Max)
					qc.MQueryTimeMin = float32(*m.Min)
					qc.MQueryTimeP99 = float32(*m.P95)
				}
				// lock_time - Lock_time
				if m, ok := v.Metrics.TimeMetrics["Lock_time"]; ok {
					qc.MLockTimeSum = float32(m.Sum)
					qc.MLockTimeMax = float32(*m.Max)
					qc.MLockTimeMin = float32(*m.Min)
					qc.MLockTimeP99 = float32(*m.P95)
				}
				// rows_sent - Rows_sent
				if m, ok := v.Metrics.NumberMetrics["Rows_sent"]; ok {
					qc.MRowsSentSum = m.Sum
					qc.MRowsSentMax = *m.Max
					qc.MRowsSentMin = *m.Min
					qc.MRowsSentP99 = *m.P95
				}
				// rows_examined - Rows_examined
				if m, ok := v.Metrics.NumberMetrics["Rows_examined"]; ok {
					qc.MRowsExaminedSum = m.Sum
					qc.MRowsExaminedMax = *m.Max
					qc.MRowsExaminedMin = *m.Min
					qc.MRowsExaminedP99 = *m.P95
				}
				// rows_affected - Rows_affected
				if m, ok := v.Metrics.NumberMetrics["Rows_affected"]; ok {
					qc.MRowsAffectedSum = m.Sum
					qc.MRowsAffectedMax = *m.Max
					qc.MRowsAffectedMin = *m.Min
					qc.MRowsAffectedP99 = *m.P95
				}
				// rows_read - Rows_read
				if m, ok := v.Metrics.NumberMetrics["Rows_read"]; ok {
					qc.MRowsReadSum = m.Sum
					qc.MRowsReadMax = *m.Max
					qc.MRowsReadMin = *m.Min
					qc.MRowsReadP99 = *m.P95
				}
				// merge_passes - Merge_passes
				if m, ok := v.Metrics.NumberMetrics["Merge_passes"]; ok {
					qc.MMergePassesSum = m.Sum
					qc.MMergePassesMax = *m.Max
					qc.MMergePassesMin = *m.Min
					qc.MMergePassesP99 = *m.P95
				}
				// innodb_io_r_ops - InnoDB_IO_r_ops
				if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_ops"]; ok {
					qc.MInnodbIoROpsSum = m.Sum
					qc.MInnodbIoROpsMax = *m.Max
					qc.MInnodbIoROpsMin = *m.Min
					qc.MInnodbIoROpsP99 = *m.P95
				}
				// innodb_io_r_bytes - InnoDB_IO_r_bytes
				if m, ok := v.Metrics.NumberMetrics["InnoDB_IO_r_bytes"]; ok {
					qc.MInnodbIoRBytesSum = m.Sum
					qc.MInnodbIoRBytesMax = *m.Max
					qc.MInnodbIoRBytesMin = *m.Min
					qc.MInnodbIoRBytesP99 = *m.P95
				}
				// innodb_io_r_wait - InnoDB_IO_r_wait
				if m, ok := v.Metrics.TimeMetrics["InnoDB_IO_r_wait"]; ok {
					qc.MInnodbIoRWaitSum = float32(m.Sum)
					qc.MInnodbIoRWaitMax = float32(*m.Max)
					qc.MInnodbIoRWaitMin = float32(*m.Min)
					qc.MInnodbIoRWaitP99 = float32(*m.P95)
				}
				// innodb_rec_lock_wait - InnoDB_rec_lock_wait
				if m, ok := v.Metrics.TimeMetrics["InnoDB_rec_lock_wait"]; ok {
					qc.MInnodbRecLockWaitSum = float32(m.Sum)
					qc.MInnodbRecLockWaitMax = float32(*m.Max)
					qc.MInnodbRecLockWaitMin = float32(*m.Min)
					qc.MInnodbRecLockWaitP99 = float32(*m.P95)
				}
				// innodb_queue_wait - InnoDB_queue_wait
				if m, ok := v.Metrics.TimeMetrics["InnoDB_queue_wait"]; ok {
					qc.MInnodbQueueWaitSum = float32(m.Sum)
					qc.MInnodbQueueWaitMax = float32(*m.Max)
					qc.MInnodbQueueWaitMin = float32(*m.Min)
					qc.MInnodbQueueWaitP99 = float32(*m.P95)
				}
				// innodb_pages_distinct - InnoDB_pages_distinct
				if m, ok := v.Metrics.NumberMetrics["InnoDB_pages_distinct"]; ok {
					qc.MInnodbPagesDistinctSum = m.Sum
					qc.MInnodbPagesDistinctMax = *m.Max
					qc.MInnodbPagesDistinctMin = *m.Min
					qc.MInnodbPagesDistinctP99 = *m.P95
				}
				// query_length - Query_length
				if m, ok := v.Metrics.NumberMetrics["Query_length"]; ok {
					qc.MQueryLengthSum = m.Sum
					qc.MQueryLengthMax = *m.Max
					qc.MQueryLengthMin = *m.Min
					qc.MQueryLengthP99 = *m.P95
				}
				// bytes_sent - Bytes_sent
				if m, ok := v.Metrics.NumberMetrics["Bytes_sent"]; ok {
					qc.MBytesSentSum = m.Sum
					qc.MBytesSentMax = *m.Max
					qc.MBytesSentMin = *m.Min
					qc.MBytesSentP99 = *m.P95
				}
				// tmp_tables - Tmp_tables
				if m, ok := v.Metrics.NumberMetrics["Tmp_tables"]; ok {
					qc.MTmpTablesSum = m.Sum
					qc.MTmpTablesMax = *m.Max
					qc.MTmpTablesMin = *m.Min
					qc.MTmpTablesP99 = *m.P95
				}
				// tmp_disk_tables - Tmp_disk_tables
				if m, ok := v.Metrics.NumberMetrics["Tmp_disk_tables"]; ok {
					qc.MTmpDiskTablesSum = m.Sum
					qc.MTmpDiskTablesMax = *m.Max
					qc.MTmpDiskTablesMin = *m.Min
					qc.MTmpDiskTablesP99 = *m.P95
				}
				// tmp_table_sizes - Tmp_table_sizes
				if m, ok := v.Metrics.NumberMetrics["Tmp_table_sizes"]; ok {
					qc.MTmpTableSizesSum = m.Sum
					qc.MTmpTableSizesMax = *m.Max
					qc.MTmpTableSizesMin = *m.Min
					qc.MTmpTableSizesP99 = *m.P95
				}
				// qc_hit - QC_Hit
				if m, ok := v.Metrics.BoolMetrics["QC_Hit"]; ok {
					qc.MQcHitSum = m.Sum
				}
				// full_scan - Full_scan
				if m, ok := v.Metrics.BoolMetrics["Full_scan"]; ok {
					qc.MFullScanSum = m.Sum
				}
				// full_join - Full_join
				if m, ok := v.Metrics.BoolMetrics["Full_join"]; ok {
					qc.MFullJoinSum = m.Sum
				}
				// tmp_table - Tmp_table
				if m, ok := v.Metrics.BoolMetrics["Tmp_table"]; ok {
					qc.MTmpTableSum = m.Sum
				}
				// tmp_table_on_disk - Tmp_table_on_disk
				if m, ok := v.Metrics.BoolMetrics["Tmp_table_on_disk"]; ok {
					qc.MTmpTableOnDiskSum = m.Sum
				}
				// filesort - Filesort
				if m, ok := v.Metrics.BoolMetrics["Filesort"]; ok {
					qc.MFilesortSum = m.Sum
				}
				// filesort_on_disk - Filesort_on_disk
				if m, ok := v.Metrics.BoolMetrics["Filesort_on_disk"]; ok {
					qc.MFilesortOnDiskSum = m.Sum
				}
				// select_full_range_join - Select_full_range_join
				if m, ok := v.Metrics.BoolMetrics["Select_full_range_join"]; ok {
					qc.MSelectFullRangeJoinSum = m.Sum
				}
				// select_range - Select_range
				if m, ok := v.Metrics.BoolMetrics["Select_range"]; ok {
					qc.MSelectRangeSum = m.Sum
				}
				// select_range_check - Select_range_check
				if m, ok := v.Metrics.BoolMetrics["Select_range_check"]; ok {
					qc.MSelectRangeCheckSum = m.Sum
				}
				// sort_range - Sort_range
				if m, ok := v.Metrics.BoolMetrics["Sort_range"]; ok {
					qc.MSortRangeSum = m.Sum
				}
				// sort_rows - Sort_rows
				if m, ok := v.Metrics.BoolMetrics["Sort_rows"]; ok {
					qc.MSortRowsSum = m.Sum
				}
				// sort_scan - Sort_scan
				if m, ok := v.Metrics.BoolMetrics["Sort_scan"]; ok {
					qc.MSortScanSum = m.Sum
				}
				// no_index_used - No_index_used
				if m, ok := v.Metrics.BoolMetrics["No_index_used"]; ok {
					qc.MNoIndexUsedSum = m.Sum
				}
				// no_good_index_used - No_good_index_used
				if m, ok := v.Metrics.BoolMetrics["No_good_index_used"]; ok {
					qc.MNoGoodIndexUsedSum = m.Sum
				}

				am.QueryClass = append(am.QueryClass, qc)
			}

			// No new events in slowlog. Nothing to send to API.
			if i == 0 {
				return closedChannelError{errors.New("closed channel")}
			}
			// Reached end of slowlog. Send all what we have to API.
			return nil
		})

		if err != nil {
			if _, ok := err.(closedChannelError); !ok {
				log.Fatal("transaction error:", err)
			}
			// Channel is closed when reached end of the slowlog.
			// Wait and try read the slowlog again.
			time.Sleep(*newEventWait)
			events = parseSlowLog(*slowLogPath, logOpt)
		}
	}
}

func bulkSend(stream pb.Agent_DataInterchangeClient, fn func(*pb.AgentMessage) error) error {
	am := &pb.AgentMessage{}
	err := fn(am)
	if err != nil {
		return err
	}
	lenQC := len(am.QueryClass)
	if lenQC > 0 {
		if err := stream.Send(am); err != nil {
			return fmt.Errorf("sent error: %v", err)
		}
		fmt.Printf("send to qan %v QC\n", lenQC)
	}
	return nil
}

func parseSlowLog(filename string, o slowlog.Options) <-chan *slowlog.Event {
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		log.Fatal("cannot open slowlog. Use --slow-log=/path/to/slow.log", err)
	}
	p := parser.NewSlowLogParser(file, o)
	go func() {
		err = p.Start()
		if err != nil {
			log.Fatal("cannot start parser", err)
		}
	}()
	return p.EventChan()
}
