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

// Package qan contains business logic of working with QAN.
package qan

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/stringset"
)

// Client represents qan-api client for data collection.
type Client struct {
	c   qanCollectorClient
	odc qanpb.ObjectDetailsClient
	db  *reform.DB
	l   *logrus.Entry
}

// NewClient returns new client for given gRPC connection.
func NewClient(cc *grpc.ClientConn, db *reform.DB) *Client {
	return &Client{
		c:   qanpb.NewCollectorClient(cc),
		odc: qanpb.NewObjectDetailsClient(cc),
		db:  db,
		l:   logrus.WithField("component", "qan"),
	}
}

// collectAgents returns Agents referenced by metricsBuckets.
func collectAgents(q *reform.Querier, metricsBuckets []*agentpb.MetricsBucket) (map[string]*models.Agent, error) {
	agentIDs := make(map[string]struct{})
	for _, m := range metricsBuckets {
		if id := m.Common.AgentId; id != "" {
			agentIDs[id] = struct{}{}
		}
	}

	agents, err := models.FindAgentsByIDs(q, stringset.ToSlice(agentIDs))
	if err != nil {
		return nil, err
	}

	m := make(map[string]*models.Agent, len(agents))
	for _, agent := range agents {
		m[agent.AgentID] = agent
	}
	return m, nil
}

// collectServices returns Services referenced by Agents.
func collectServices(q *reform.Querier, agents map[string]*models.Agent) (map[string]*models.Service, error) {
	serviceIDs := make(map[string]struct{})
	for _, a := range agents {
		if id := a.ServiceID; id != nil {
			serviceIDs[*id] = struct{}{}
		}
	}

	return models.FindServicesByIDs(q, stringset.ToSlice(serviceIDs))
}

// collectNodes returns Nodes referenced by Services.
func collectNodes(q *reform.Querier, services map[string]*models.Service) (map[string]*models.Node, error) {
	nodeIDs := make(map[string]struct{})
	for _, s := range services {
		if id := s.NodeID; id != "" {
			nodeIDs[id] = struct{}{}
		}
	}

	nodes, err := models.FindNodesByIDs(q, stringset.ToSlice(nodeIDs))
	if err != nil {
		return nil, err
	}

	m := make(map[string]*models.Node, len(nodes))
	for _, node := range nodes {
		m[node.NodeID] = node
	}
	return m, nil
}

// QueryExists check if query value in request exists in clickhouse.
// This avoid receiving custom queries.
func (c *Client) QueryExists(ctx context.Context, serviceID, query string) error {
	qanReq := &qanpb.QueryExistsRequest{
		Serviceid: serviceID,
		Query:     query,
	}
	c.l.Debugf("%+v", qanReq)
	resp, err := c.odc.QueryExists(ctx, qanReq)
	if err != nil {
		return err
	}
	if !resp.Value {
		return fmt.Errorf("given query is not valid")
	}

	return nil
}

// ExplainFingerprintByQueryID get query for given query ID.
// This avoid receiving custom queries.
func (c *Client) ExplainFingerprintByQueryID(ctx context.Context, serviceID, queryID string) (*qanpb.ExplainFingerprintByQueryIDReply, error) {
	qanReq := &qanpb.ExplainFingerprintByQueryIDRequest{
		Serviceid: serviceID,
		QueryId:   queryID,
	}
	c.l.Debugf("%+v", qanReq)
	res, err := c.odc.ExplainFingerprintByQueryID(ctx, qanReq)
	if err != nil {
		return res, err
	}

	return res, nil
}

// SchemaByQueryID returns schema for given queryID and serviceID.
func (c *Client) SchemaByQueryID(ctx context.Context, serviceID, queryID string) (*qanpb.SchemaByQueryIDReply, error) {
	qanReq := &qanpb.SchemaByQueryIDRequest{
		ServiceId: serviceID,
		QueryId:   queryID,
	}
	c.l.Debugf("%+v", qanReq)
	res, err := c.odc.SchemaByQueryID(ctx, qanReq)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Collect adds labels to the data from pmm-agent and sends it to qan-api.
func (c *Client) Collect(ctx context.Context, metricsBuckets []*agentpb.MetricsBucket) error {
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			c.l.Warnf("Collect for %d buckets took %s.", len(metricsBuckets), dur)
		}
	}()

	agents, err := collectAgents(c.db.Querier, metricsBuckets)
	if err != nil {
		return err
	}
	services, err := collectServices(c.db.Querier, agents)
	if err != nil {
		return err
	}
	nodes, err := collectNodes(c.db.Querier, services)
	if err != nil {
		return err
	}

	convertedMetricsBuckets := make([]*qanpb.MetricsBucket, 0, len(metricsBuckets))
	for _, m := range metricsBuckets {
		agent := agents[m.Common.AgentId]
		if agent == nil {
			c.l.Errorf("No Agent with ID %q for bucket with query_id %q, can't add labels.", m.Common.AgentId, m.Common.Queryid)
			continue
		}

		serviceID := pointer.GetString(agent.ServiceID)
		service := services[serviceID]
		if service == nil {
			c.l.Errorf("No Service with ID %q for bucket with query_id %q, can't add labels.", serviceID, m.Common.Queryid)
			continue
		}

		node := nodes[service.NodeID]
		if node == nil {
			c.l.Errorf("No Node with ID %q for bucket with query_id %q, can't add labels.", service.NodeID, m.Common.Queryid)
			continue
		}

		labels, err := models.MergeLabels(node, service, agent)
		if err != nil {
			c.l.Error(err)
			continue
		}

		mb := &qanpb.MetricsBucket{
			Queryid:              m.Common.Queryid,
			ExplainFingerprint:   m.Common.ExplainFingerprint,
			PlaceholdersCount:    m.Common.PlaceholdersCount,
			Fingerprint:          m.Common.Fingerprint,
			ServiceName:          service.ServiceName,
			Database:             m.Common.Database,
			Schema:               m.Common.Schema,
			Tables:               m.Common.Tables,
			Username:             m.Common.Username,
			ClientHost:           m.Common.ClientHost,
			NodeId:               node.NodeID,
			NodeName:             node.NodeName,
			NodeType:             string(node.NodeType),
			ServiceId:            service.ServiceID,
			ServiceType:          string(service.ServiceType),
			AgentId:              m.Common.AgentId,
			AgentType:            m.Common.AgentType,
			PeriodStartUnixSecs:  m.Common.PeriodStartUnixSecs,
			PeriodLengthSecs:     m.Common.PeriodLengthSecs,
			Example:              m.Common.Example,
			ExampleFormat:        convertExampleFormat(m.Common.ExampleFormat), //nolint:staticcheck
			ExampleType:          convertExampleType(m.Common.ExampleType),
			IsTruncated:          m.Common.IsTruncated,
			NumQueriesWithErrors: m.Common.NumQueriesWithErrors,
			Errors:               m.Common.Errors,
			NumQueries:           m.Common.NumQueries,
			MQueryTimeCnt:        m.Common.MQueryTimeCnt,
			MQueryTimeSum:        m.Common.MQueryTimeSum,
			MQueryTimeMin:        m.Common.MQueryTimeMin,
			MQueryTimeMax:        m.Common.MQueryTimeMax,
			MQueryTimeP99:        m.Common.MQueryTimeP99,
		}

		switch {
		case m.Mysql != nil:
			fillMySQL(mb, m.Mysql)
		case m.Mongodb != nil:
			fillMongoDB(mb, m.Mongodb)
		case m.Postgresql != nil:
			fillPostgreSQL(mb, m.Postgresql)
		}

		// Ordered the same as fields in MetricsBucket
		for labelName, field := range map[string]*string{
			"machine_id":      &mb.MachineId,
			"container_id":    &mb.ContainerId,
			"container_name":  &mb.ContainerName,
			"node_model":      &mb.NodeModel,
			"region":          &mb.Region,
			"az":              &mb.Az,
			"environment":     &mb.Environment,
			"cluster":         &mb.Cluster,
			"replication_set": &mb.ReplicationSet,
		} {
			value := labels[labelName]
			delete(labels, labelName)
			if *field != "" {
				if *field == value {
					c.l.Debugf("%q is not empty, but has the same value %q.", labelName, *field)
				} else {
					c.l.Warnf("%q is not empty: overwriting %q with %q.", labelName, *field, value)
				}
			}
			*field = value
		}

		for _, labelName := range []string{
			"agent_id",
			"agent_type",
			"service_id",
			"service_type",
			"service_name",
			"node_id",
			"node_type",
			"node_name",
		} {
			delete(labels, labelName)
		}

		for k, l := range m.Common.Comments {
			labels[k] = l
		}
		mb.Labels = labels

		convertedMetricsBuckets = append(convertedMetricsBuckets, mb)
	}

	// Slice metrics, so request to qan-api is not too big
	const bucketSize = 25000
	from, to := 0, bucketSize
	// Send at least one time, even though it's empty
	for from <= len(convertedMetricsBuckets) {
		if to > len(convertedMetricsBuckets) {
			to = len(convertedMetricsBuckets)
		}
		qanReq := &qanpb.CollectRequest{
			MetricsBucket: convertedMetricsBuckets[from:to],
		}
		c.l.Debugf("%+v", qanReq)
		res, err := c.c.Collect(ctx, qanReq)
		if err != nil {
			return errors.Wrap(err, "failed to sent CollectRequest to QAN")
		}
		c.l.Debugf("%+v", res)

		from += bucketSize
		to += bucketSize
	}

	return nil
}

//nolint:staticcheck
func convertExampleFormat(exampleFormat agentpb.ExampleFormat) qanpb.ExampleFormat {
	switch exampleFormat {
	case agentpb.ExampleFormat_EXAMPLE:
		return qanpb.ExampleFormat_EXAMPLE
	case agentpb.ExampleFormat_FINGERPRINT:
		return qanpb.ExampleFormat_FINGERPRINT
	default:
		return qanpb.ExampleFormat_EXAMPLE_FORMAT_INVALID
	}
}

func convertExampleType(exampleType agentpb.ExampleType) qanpb.ExampleType {
	switch exampleType {
	case agentpb.ExampleType_RANDOM:
		return qanpb.ExampleType_RANDOM
	case agentpb.ExampleType_SLOWEST:
		return qanpb.ExampleType_SLOWEST
	case agentpb.ExampleType_FASTEST:
		return qanpb.ExampleType_FASTEST
	case agentpb.ExampleType_WITH_ERROR:
		return qanpb.ExampleType_WITH_ERROR
	default:
		return qanpb.ExampleType_EXAMPLE_TYPE_INVALID
	}
}

func fillMySQL(mb *qanpb.MetricsBucket, bm *agentpb.MetricsBucket_MySQL) {
	mb.MLockTimeCnt = bm.MLockTimeCnt
	mb.MLockTimeSum = bm.MLockTimeSum
	mb.MLockTimeMin = bm.MLockTimeMin
	mb.MLockTimeMax = bm.MLockTimeMax
	mb.MLockTimeP99 = bm.MLockTimeP99

	mb.MRowsSentCnt = bm.MRowsSentCnt
	mb.MRowsSentSum = bm.MRowsSentSum
	mb.MRowsSentMin = bm.MRowsSentMin
	mb.MRowsSentMax = bm.MRowsSentMax
	mb.MRowsSentP99 = bm.MRowsSentP99

	mb.MRowsExaminedCnt = bm.MRowsExaminedCnt
	mb.MRowsExaminedSum = bm.MRowsExaminedSum
	mb.MRowsExaminedMin = bm.MRowsExaminedMin
	mb.MRowsExaminedMax = bm.MRowsExaminedMax
	mb.MRowsExaminedP99 = bm.MRowsExaminedP99

	mb.MRowsAffectedCnt = bm.MRowsAffectedCnt
	mb.MRowsAffectedSum = bm.MRowsAffectedSum
	mb.MRowsAffectedMin = bm.MRowsAffectedMin
	mb.MRowsAffectedMax = bm.MRowsAffectedMax
	mb.MRowsAffectedP99 = bm.MRowsAffectedP99

	mb.MRowsReadCnt = bm.MRowsReadCnt
	mb.MRowsReadSum = bm.MRowsReadSum
	mb.MRowsReadMin = bm.MRowsReadMin
	mb.MRowsReadMax = bm.MRowsReadMax
	mb.MRowsReadP99 = bm.MRowsReadP99

	mb.MMergePassesCnt = bm.MMergePassesCnt
	mb.MMergePassesSum = bm.MMergePassesSum
	mb.MMergePassesMin = bm.MMergePassesMin
	mb.MMergePassesMax = bm.MMergePassesMax
	mb.MMergePassesP99 = bm.MMergePassesP99

	mb.MInnodbIoROpsCnt = bm.MInnodbIoROpsCnt
	mb.MInnodbIoROpsSum = bm.MInnodbIoROpsSum
	mb.MInnodbIoROpsMin = bm.MInnodbIoROpsMin
	mb.MInnodbIoROpsMax = bm.MInnodbIoROpsMax
	mb.MInnodbIoROpsP99 = bm.MInnodbIoROpsP99

	mb.MInnodbIoRBytesCnt = bm.MInnodbIoRBytesCnt
	mb.MInnodbIoRBytesSum = bm.MInnodbIoRBytesSum
	mb.MInnodbIoRBytesMin = bm.MInnodbIoRBytesMin
	mb.MInnodbIoRBytesMax = bm.MInnodbIoRBytesMax
	mb.MInnodbIoRBytesP99 = bm.MInnodbIoRBytesP99

	mb.MInnodbIoRWaitCnt = bm.MInnodbIoRWaitCnt
	mb.MInnodbIoRWaitSum = bm.MInnodbIoRWaitSum
	mb.MInnodbIoRWaitMin = bm.MInnodbIoRWaitMin
	mb.MInnodbIoRWaitMax = bm.MInnodbIoRWaitMax
	mb.MInnodbIoRWaitP99 = bm.MInnodbIoRWaitP99

	mb.MInnodbRecLockWaitCnt = bm.MInnodbRecLockWaitCnt
	mb.MInnodbRecLockWaitSum = bm.MInnodbRecLockWaitSum
	mb.MInnodbRecLockWaitMin = bm.MInnodbRecLockWaitMin
	mb.MInnodbRecLockWaitMax = bm.MInnodbRecLockWaitMax
	mb.MInnodbRecLockWaitP99 = bm.MInnodbRecLockWaitP99

	mb.MInnodbQueueWaitCnt = bm.MInnodbQueueWaitCnt
	mb.MInnodbQueueWaitSum = bm.MInnodbQueueWaitSum
	mb.MInnodbQueueWaitMin = bm.MInnodbQueueWaitMin
	mb.MInnodbQueueWaitMax = bm.MInnodbQueueWaitMax
	mb.MInnodbQueueWaitP99 = bm.MInnodbQueueWaitP99

	mb.MInnodbPagesDistinctCnt = bm.MInnodbPagesDistinctCnt
	mb.MInnodbPagesDistinctSum = bm.MInnodbPagesDistinctSum
	mb.MInnodbPagesDistinctMin = bm.MInnodbPagesDistinctMin
	mb.MInnodbPagesDistinctMax = bm.MInnodbPagesDistinctMax
	mb.MInnodbPagesDistinctP99 = bm.MInnodbPagesDistinctP99

	mb.MQueryLengthCnt = bm.MQueryLengthCnt
	mb.MQueryLengthSum = bm.MQueryLengthSum
	mb.MQueryLengthMin = bm.MQueryLengthMin
	mb.MQueryLengthMax = bm.MQueryLengthMax
	mb.MQueryLengthP99 = bm.MQueryLengthP99

	mb.MBytesSentCnt = bm.MBytesSentCnt
	mb.MBytesSentSum = bm.MBytesSentSum
	mb.MBytesSentMin = bm.MBytesSentMin
	mb.MBytesSentMax = bm.MBytesSentMax
	mb.MBytesSentP99 = bm.MBytesSentP99

	mb.MTmpTablesCnt = bm.MTmpTablesCnt
	mb.MTmpTablesSum = bm.MTmpTablesSum
	mb.MTmpTablesMin = bm.MTmpTablesMin
	mb.MTmpTablesMax = bm.MTmpTablesMax
	mb.MTmpTablesP99 = bm.MTmpTablesP99

	mb.MTmpDiskTablesCnt = bm.MTmpDiskTablesCnt
	mb.MTmpDiskTablesSum = bm.MTmpDiskTablesSum
	mb.MTmpDiskTablesMin = bm.MTmpDiskTablesMin
	mb.MTmpDiskTablesMax = bm.MTmpDiskTablesMax
	mb.MTmpDiskTablesP99 = bm.MTmpDiskTablesP99

	mb.MTmpTableSizesCnt = bm.MTmpTableSizesCnt
	mb.MTmpTableSizesSum = bm.MTmpTableSizesSum
	mb.MTmpTableSizesMin = bm.MTmpTableSizesMin
	mb.MTmpTableSizesMax = bm.MTmpTableSizesMax
	mb.MTmpTableSizesP99 = bm.MTmpTableSizesP99

	mb.MQcHitCnt = bm.MQcHitCnt
	mb.MQcHitSum = bm.MQcHitSum

	mb.MFullScanCnt = bm.MFullScanCnt
	mb.MFullScanSum = bm.MFullScanSum

	mb.MFullJoinCnt = bm.MFullJoinCnt
	mb.MFullJoinSum = bm.MFullJoinSum

	mb.MTmpTableCnt = bm.MTmpTableCnt
	mb.MTmpTableSum = bm.MTmpTableSum

	mb.MTmpTableOnDiskCnt = bm.MTmpTableOnDiskCnt
	mb.MTmpTableOnDiskSum = bm.MTmpTableOnDiskSum

	mb.MFilesortCnt = bm.MFilesortCnt
	mb.MFilesortSum = bm.MFilesortSum

	mb.MFilesortOnDiskCnt = bm.MFilesortOnDiskCnt
	mb.MFilesortOnDiskSum = bm.MFilesortOnDiskSum

	mb.MSelectFullRangeJoinCnt = bm.MSelectFullRangeJoinCnt
	mb.MSelectFullRangeJoinSum = bm.MSelectFullRangeJoinSum

	mb.MSelectRangeCnt = bm.MSelectRangeCnt
	mb.MSelectRangeSum = bm.MSelectRangeSum

	mb.MSelectRangeCheckCnt = bm.MSelectRangeCheckCnt
	mb.MSelectRangeCheckSum = bm.MSelectRangeCheckSum

	mb.MSortRangeCnt = bm.MSortRangeCnt
	mb.MSortRangeSum = bm.MSortRangeSum

	mb.MSortRowsCnt = bm.MSortRowsCnt
	mb.MSortRowsSum = bm.MSortRowsSum

	mb.MSortScanCnt = bm.MSortScanCnt
	mb.MSortScanSum = bm.MSortScanSum

	mb.MNoIndexUsedCnt = bm.MNoIndexUsedCnt
	mb.MNoIndexUsedSum = bm.MNoIndexUsedSum

	mb.MNoGoodIndexUsedCnt = bm.MNoGoodIndexUsedCnt
	mb.MNoGoodIndexUsedSum = bm.MNoGoodIndexUsedSum
}

func fillMongoDB(mb *qanpb.MetricsBucket, bm *agentpb.MetricsBucket_MongoDB) {
	mb.MDocsReturnedCnt = bm.MDocsReturnedCnt
	mb.MDocsReturnedSum = bm.MDocsReturnedSum
	mb.MDocsReturnedMin = bm.MDocsReturnedMin
	mb.MDocsReturnedMax = bm.MDocsReturnedMax
	mb.MDocsReturnedP99 = bm.MDocsReturnedP99

	mb.MResponseLengthCnt = bm.MResponseLengthCnt
	mb.MResponseLengthSum = bm.MResponseLengthSum
	mb.MResponseLengthMin = bm.MResponseLengthMin
	mb.MResponseLengthMax = bm.MResponseLengthMax
	mb.MResponseLengthP99 = bm.MResponseLengthP99

	mb.MDocsScannedCnt = bm.MDocsScannedCnt
	mb.MDocsScannedSum = bm.MDocsScannedSum
	mb.MDocsScannedMin = bm.MDocsScannedMin
	mb.MDocsScannedMax = bm.MDocsScannedMax
	mb.MDocsScannedP99 = bm.MDocsScannedP99
}

func fillPostgreSQL(mb *qanpb.MetricsBucket, bp *agentpb.MetricsBucket_PostgreSQL) {
	mb.MRowsSentCnt = bp.MRowsCnt
	mb.MRowsSentSum = bp.MRowsSum

	mb.MSharedBlksHitCnt = bp.MSharedBlksHitCnt
	mb.MSharedBlksHitSum = bp.MSharedBlksHitSum
	mb.MSharedBlksReadCnt = bp.MSharedBlksReadCnt
	mb.MSharedBlksReadSum = bp.MSharedBlksReadSum
	mb.MSharedBlksDirtiedCnt = bp.MSharedBlksDirtiedCnt
	mb.MSharedBlksDirtiedSum = bp.MSharedBlksDirtiedSum
	mb.MSharedBlksWrittenCnt = bp.MSharedBlksWrittenCnt
	mb.MSharedBlksWrittenSum = bp.MSharedBlksWrittenSum

	mb.MLocalBlksHitCnt = bp.MLocalBlksHitCnt
	mb.MLocalBlksHitSum = bp.MLocalBlksHitSum
	mb.MLocalBlksReadCnt = bp.MLocalBlksReadCnt
	mb.MLocalBlksReadSum = bp.MLocalBlksReadSum
	mb.MLocalBlksDirtiedCnt = bp.MLocalBlksDirtiedCnt
	mb.MLocalBlksDirtiedSum = bp.MLocalBlksDirtiedSum
	mb.MLocalBlksWrittenCnt = bp.MLocalBlksWrittenCnt
	mb.MLocalBlksWrittenSum = bp.MLocalBlksWrittenSum

	mb.MTempBlksReadCnt = bp.MTempBlksReadCnt
	mb.MTempBlksReadSum = bp.MTempBlksReadSum
	mb.MTempBlksWrittenCnt = bp.MTempBlksWrittenCnt
	mb.MTempBlksWrittenSum = bp.MTempBlksWrittenSum

	mb.MSharedBlkReadTimeCnt = bp.MSharedBlkReadTimeCnt
	mb.MSharedBlkReadTimeSum = bp.MSharedBlkReadTimeSum
	mb.MSharedBlkWriteTimeCnt = bp.MSharedBlkWriteTimeCnt
	mb.MSharedBlkWriteTimeSum = bp.MSharedBlkWriteTimeSum
	mb.MLocalBlkReadTimeCnt = bp.MLocalBlkReadTimeCnt
	mb.MLocalBlkReadTimeSum = bp.MLocalBlkReadTimeSum
	mb.MLocalBlkWriteTimeCnt = bp.MLocalBlkWriteTimeCnt
	mb.MLocalBlkWriteTimeSum = bp.MLocalBlkWriteTimeSum

	mb.MCpuSysTimeCnt = bp.MCpuSysTimeCnt
	mb.MCpuSysTimeSum = bp.MCpuSysTimeSum

	mb.MCpuUserTimeCnt = bp.MCpuUserTimeCnt
	mb.MCpuUserTimeSum = bp.MCpuUserTimeSum

	mb.MPlansCallsCnt = bp.MPlansCallsCnt
	mb.MPlansCallsSum = bp.MPlansCallsSum

	mb.MWalRecordsCnt = bp.MWalRecordsCnt
	mb.MWalRecordsSum = bp.MWalRecordsSum

	mb.MWalFpiCnt = bp.MWalFpiCnt
	mb.MWalFpiSum = bp.MWalFpiSum

	mb.MWalBytesCnt = bp.MWalBytesCnt
	mb.MWalBytesSum = bp.MWalBytesSum

	mb.MPlanTimeSum = bp.MPlanTimeSum
	mb.MPlanTimeMin = bp.MPlanTimeMin
	mb.MPlanTimeMax = bp.MPlanTimeMax

	mb.CmdType = bp.CmdType

	mb.TopQueryid = bp.TopQueryid
	mb.TopQuery = bp.TopQuery
	mb.ApplicationName = bp.ApplicationName
	mb.Planid = bp.Planid
	mb.QueryPlan = bp.QueryPlan
	mb.HistogramItems = convertHistogramItems(bp.HistogramItems)
}

func convertHistogramItems(items []*agentpb.HistogramItem) []string {
	res := []string{}
	for _, v := range items {
		item := &qanpb.HistogramItem{
			Range:     v.Range,
			Frequency: v.Frequency,
		}

		json, err := json.Marshal(item)
		if err != nil {
			continue
		}

		res = append(res, string(json))
	}

	return res
}
