// Copyright (C) 2023 Percona LLC
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

// Package models holds the qan data-access layer: ingestion and serving.
package models

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/reflect/protoreflect"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/utils/ddsketch"
)

const (
	dedupTTL        = 3 * time.Minute
	maxDedupEntries = 500000
)

// insertCols lists the metrics_raw columns we populate, in the order bucketArgs returns values.
var insertCols = []string{
	"queryid", "service_id", "service_name", "service_type", "`database`", "`schema`", "username", "client_host",
	"cluster", "environment", "replication_set", "node_id", "node_name", "az", "region", "container_name",
	"cmd_type", "agent_id", "agent_type", "labels",
	"fingerprint", "tables", "explain_fingerprint", "placeholders_count", "example", "example_metrics", "query_plan", "planid", "plan_summary", "is_truncated", "example_type",
	"period_start", "period_length", "sketch_layout",
	"num_queries", "num_queries_with_errors", "num_queries_with_warnings",
	"m_query_time_sum", "m_query_time_cnt", "m_query_time_min", "m_query_time_max", "m_query_time_sketch",
	"m_lock_time_sum", "m_lock_time_cnt", "m_lock_time_min", "m_lock_time_max", "m_lock_time_sketch",
	"m_rows_sent_sum", "m_rows_sent_cnt", "m_rows_sent_min", "m_rows_sent_max",
	"m_rows_examined_sum", "m_rows_examined_cnt", "m_rows_examined_min", "m_rows_examined_max",
	"m_rows_affected_sum", "m_rows_affected_cnt", "m_rows_affected_min", "m_rows_affected_max",
	"m_bytes_sent_sum", "m_bytes_sent_cnt", "m_bytes_sent_min", "m_bytes_sent_max",
	"m_sum", "m_cnt",
}

// coreMetrics are stored as typed columns; everything else goes to the long-tail maps.
var coreMetrics = map[string]bool{
	"query_time": true, "lock_time": true, "rows_sent": true,
	"rows_examined": true, "rows_affected": true, "bytes_sent": true,
}

type longTailField struct {
	fd     protoreflect.FieldDescriptor
	metric string
}

// longTailSum/Cnt are the proto float fields routed to the m_sum/m_cnt maps. Built
// once from the descriptor so new proto metrics auto-route without code changes.
var longTailSum, longTailCnt []longTailField

func init() {
	fields := (&qanv1.MetricsBucket{}).ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.FloatKind {
			continue
		}
		name := string(fd.Name())
		if !strings.HasPrefix(name, "m_") {
			continue
		}
		body := name[len("m_"):]
		switch {
		case strings.HasSuffix(body, "_sum"):
			if metric := strings.TrimSuffix(body, "_sum"); !coreMetrics[metric] {
				longTailSum = append(longTailSum, longTailField{fd, metric})
			}
		case strings.HasSuffix(body, "_cnt"):
			if metric := strings.TrimSuffix(body, "_cnt"); !coreMetrics[metric] {
				longTailCnt = append(longTailCnt, longTailField{fd, metric})
			}
		}
	}
}

var (
	bucketsIngested = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "qan", Subsystem: "ingestion", Name: "buckets_ingested_total",
		Help: "Total number of metrics buckets written to metrics_raw.",
	})
	bucketsDeduped = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "qan", Subsystem: "ingestion", Name: "buckets_deduped_total",
		Help: "Total number of duplicate metrics buckets dropped by the idempotency cache.",
	})
)

// Ingestor writes incoming metrics buckets into metrics_raw.
type Ingestor struct {
	conn  driver.Conn
	dedup *dedupCache
	l     *logrus.Entry
}

// NewIngestor returns an Ingestor writing through conn.
func NewIngestor(conn driver.Conn) *Ingestor {
	return &Ingestor{
		conn:  conn,
		dedup: newDedupCache(dedupTTL),
		l:     logrus.WithField("component", "ingestion"),
	}
}

// Save deduplicates and bulk-inserts buckets into metrics_raw via a native columnar batch.
func (i *Ingestor) Save(ctx context.Context, buckets []*qanv1.MetricsBucket) error {
	if len(buckets) == 0 {
		return nil
	}

	fresh := make([]*qanv1.MetricsBucket, 0, len(buckets))
	var deduped int
	for _, mb := range buckets {
		if i.dedup.seenBefore(idempotencyKey(mb)) {
			deduped++
			continue
		}
		fresh = append(fresh, mb)
	}
	bucketsDeduped.Add(float64(deduped))
	if len(fresh) == 0 {
		return nil
	}

	batch, err := i.conn.PrepareBatch(ctx, "INSERT INTO metrics_raw ("+strings.Join(insertCols, ", ")+")")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}
	for _, mb := range fresh {
		err = batch.Append(bucketArgs(mb)...)
		if err != nil {
			_ = batch.Abort()
			return fmt.Errorf("append bucket: %w", err)
		}
	}
	err = batch.Send()
	if err != nil {
		return fmt.Errorf("send batch: %w", err)
	}

	bucketsIngested.Add(float64(len(fresh)))
	i.l.Debugf("Ingested %d buckets, deduped %d.", len(fresh), deduped)
	return nil
}

// bucketArgs returns the metrics_raw column values for mb, in insertCols order.
func bucketArgs(mb *qanv1.MetricsBucket) []any {
	sums, cnts := longTail(mb)

	labels := mb.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	tables := mb.Tables
	if tables == nil {
		tables = []string{}
	}
	emptySketch := map[uint16]uint64{}

	return []any{
		mb.Queryid, mb.ServiceId, mb.ServiceName, mb.ServiceType, mb.Database, mb.Schema, mb.Username, mb.ClientHost,
		mb.Cluster, mb.Environment, mb.ReplicationSet, mb.NodeId, mb.NodeName, mb.Az, mb.Region, mb.ContainerName,
		mb.CmdType, mb.AgentId, mb.AgentType.String(), labels,
		mb.Fingerprint, tables, mb.ExplainFingerprint, mb.PlaceholdersCount, mb.Example, mb.ExampleMetrics, mb.QueryPlan, mb.Planid, mb.PlanSummary, boolToUint8(mb.IsTruncated), mb.ExampleType.String(),
		time.Unix(int64(mb.PeriodStartUnixSecs), 0).UTC(), mb.PeriodLengthSecs, uint16(ddsketch.LayoutVersion),
		float64(mb.NumQueries), float64(mb.NumQueriesWithErrors), float64(mb.NumQueriesWithWarnings),
		float64(mb.MQueryTimeSum), uint64(mb.MQueryTimeCnt), mb.MQueryTimeMin, mb.MQueryTimeMax, sketchToMap(mb.MQueryTimeSketch),
		float64(mb.MLockTimeSum), uint64(mb.MLockTimeCnt), mb.MLockTimeMin, mb.MLockTimeMax, emptySketch,
		float64(mb.MRowsSentSum), uint64(mb.MRowsSentCnt), mb.MRowsSentMin, mb.MRowsSentMax,
		float64(mb.MRowsExaminedSum), uint64(mb.MRowsExaminedCnt), mb.MRowsExaminedMin, mb.MRowsExaminedMax,
		float64(mb.MRowsAffectedSum), uint64(mb.MRowsAffectedCnt), mb.MRowsAffectedMin, mb.MRowsAffectedMax,
		float64(mb.MBytesSentSum), uint64(mb.MBytesSentCnt), mb.MBytesSentMin, mb.MBytesSentMax,
		sums, cnts,
	}
}

// longTail extracts the non-core metric sums/counts into sparse maps (nonzero only).
func longTail(mb *qanv1.MetricsBucket) (map[string]float64, map[string]uint64) {
	r := mb.ProtoReflect()
	sums := make(map[string]float64)
	cnts := make(map[string]uint64)
	for _, f := range longTailSum {
		if v := r.Get(f.fd).Float(); v != 0 {
			sums[f.metric] = v
		}
	}
	for _, f := range longTailCnt {
		if v := r.Get(f.fd).Float(); v != 0 {
			cnts[f.metric] = uint64(v)
		}
	}
	return sums, cnts
}

// idempotencyKey identifies a bucket so at-least-once retries don't double-count.
func idempotencyKey(mb *qanv1.MetricsBucket) uint64 {
	h := fnv.New64a()
	for _, s := range []string{mb.AgentId, mb.Queryid, mb.Database, mb.Schema, mb.Username, mb.ClientHost} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], mb.PeriodStartUnixSecs)
	_, _ = h.Write(b[:])
	return h.Sum64()
}

// sketchToMap converts the wire sketch (uint32 bucket keys) to the ClickHouse column type (uint16 keys).
func sketchToMap(s map[uint32]uint64) map[uint16]uint64 {
	out := make(map[uint16]uint64, len(s))
	for k, v := range s {
		out[uint16(k)] = v
	}
	return out
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// dedupCache is a bounded, TTL-based set of recently-seen idempotency keys.
type dedupCache struct {
	mu   sync.Mutex
	seen map[uint64]time.Time
	ttl  time.Duration
}

func newDedupCache(ttl time.Duration) *dedupCache {
	return &dedupCache{seen: make(map[uint64]time.Time), ttl: ttl}
}

// seenBefore reports whether key was seen within the TTL, recording it otherwise.
func (d *dedupCache) seenBefore(key uint64) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if exp, ok := d.seen[key]; ok && exp.After(now) {
		return true
	}
	if len(d.seen) >= maxDedupEntries {
		for k, exp := range d.seen {
			if !exp.After(now) {
				delete(d.seen, k)
			}
		}
	}
	d.seen[key] = now.Add(d.ttl)
	return false
}
