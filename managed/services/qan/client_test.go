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

package qan

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/sqlmetrics"
)

// assertCollected asserts that the client sent expected to qan-api. Protobuf messages carry
// an internal size cache that proto.Size populates and reflect.DeepEqual compares, so
// testify's own argument matching reports a false mismatch here; proto.Equal ignores it.
func assertCollected(t *testing.T, c *mockQanCollectorClient, expected *qanpb.CollectRequest) {
	t.Helper()

	for _, call := range c.Calls {
		if call.Method == "Collect" && proto.Equal(call.Arguments.Get(1).(*qanpb.CollectRequest), expected) {
			return
		}
	}

	t.Errorf("no Collect call carried the expected %d buckets", len(expected.MetricsBucket))
}

func TestClient(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	ctx := logger.Set(t.Context(), t.Name())
	defer func() {
		require.NoError(t, sqlDB.Close())
		assert.Equal(t, 18, reformL.Requests())
	}()

	for _, str := range []reform.Struct{
		&models.Node{
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			NodeType:     models.GenericNodeType,
			NodeName:     "test-generic-node",
			Address:      "1.2.3.4",
			CustomLabels: []byte(`{"_node_label": "foo"}`),
			NodeModel:    "test-node-model",
		},
		&models.Agent{
			AgentID:      "217907dc-d34d-4e2e-aa84-a1b765d49853",
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: new("cc663f36-18ca-40a1-aea9-c6310bb4738d"),
		},

		&models.Service{
			ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
			ServiceType:  models.MySQLServiceType,
			ServiceName:  "test-mysql",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      new("5.6.7.8"),
			Port:         new(uint16(3306)),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
			AgentType:    models.QANMySQLPerfSchemaAgentType,
			PMMAgentID:   new("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    new("014647c3-b2f5-44eb-94f4-d943260a968c"),
			CustomLabels: []byte(`{"_agent_label": "baz"}`),
			ListenPort:   new(uint16(12345)),
		},

		&models.Service{
			ServiceID:    "9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
			ServiceType:  models.PostgreSQLServiceType,
			ServiceName:  "test-postgresql",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      new("5.6.7.8"),
			Port:         new(uint16(5432)),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "29e14468-d479-4b4d-bfb7-4ac2fb865bac",
			AgentType:    models.QANPostgreSQLPgStatementsAgentType,
			PMMAgentID:   new("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    new("9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1"),
			CustomLabels: []byte(`{"_agent_label": "postgres-baz"}`),
			ListenPort:   new(uint16(12345)),
		},

		&models.Service{
			ServiceID:    "1fce2502-ecc7-46d4-968b-18d7907f2543",
			ServiceType:  models.MongoDBServiceType,
			ServiceName:  "test-mongodb",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      new("5.6.7.8"),
			Port:         new(uint16(27017)),
			CustomLabels: []byte(`{"_service_label": "mongo-bar"}`),
		},

		&models.Agent{
			AgentID:      "b153f0d8-34e4-4635-9184-499161b4d12c",
			AgentType:    models.QANMongoDBProfilerAgentType,
			PMMAgentID:   new("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    new("1fce2502-ecc7-46d4-968b-18d7907f2543"),
			CustomLabels: []byte(`{"_agent_label": "mongodb-baz"}`),
			ListenPort:   new(uint16(12345)),
		},
	} {
		require.NoError(t, db.Insert(str), "%+v", str)
	}

	t.Run("Test MySQL Metrics conversion", func(t *testing.T) {
		c := &mockQanCollectorClient{}
		c.Test(t)
		defer c.AssertExpectations(t)

		client := &Client{
			c:  c,
			db: db,
			l:  logrus.WithField("test", t.Name()),
		}
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).Return(&qanpb.CollectResponse{}, nil)
		metricsBuckets := []*agentv1.MetricsBucket{
			{
				Common: &agentv1.MetricsBucket_Common{
					Queryid:             "some-query-id",
					Fingerprint:         "SELECT * FROM `city`",
					Schema:              "world",
					AgentId:             "75bb30d3-ef4a-4147-97a8-621a996611dd",
					PeriodStartUnixSecs: 1554116340,
					PeriodLengthSecs:    60,
					AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
					Example:             "SELECT /* AllCities */ * FROM city",
					ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
					NumQueries:          1,
					MQueryTimeCnt:       1,
					MQueryTimeSum:       1234,
				},
				Mysql: &agentv1.MetricsBucket_MySQL{
					MLockTimeCnt:     1,
					MLockTimeSum:     3456,
					MRowsSentCnt:     1,
					MRowsSentSum:     4079,
					MRowsExaminedCnt: 1,
					MRowsExaminedSum: 4079,
					MFullScanCnt:     1,
					MFullScanSum:     1,
					MNoIndexUsedCnt:  1,
					MNoIndexUsedSum:  1,
				},
			},
		}
		err := client.Collect(ctx, metricsBuckets)
		require.NoError(t, err)

		expectedRequest := &qanpb.CollectRequest{MetricsBucket: []*qanpb.MetricsBucket{
			{
				Queryid:             "some-query-id",
				Fingerprint:         "SELECT * FROM `city`",
				Schema:              "world",
				AgentId:             "75bb30d3-ef4a-4147-97a8-621a996611dd",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:             "SELECT /* AllCities */ * FROM city",
				ExampleType:         qanpb.ExampleType_EXAMPLE_TYPE_RANDOM,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       1234,
				ServiceId:           "014647c3-b2f5-44eb-94f4-d943260a968c",
				ServiceName:         "test-mysql",
				ServiceType:         "mysql",
				NodeId:              "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:            "test-generic-node",
				NodeType:            "generic",
				NodeModel:           "test-node-model",
				Labels: map[string]string{
					"_agent_label":   "baz",
					"_node_label":    "foo",
					"_service_label": "bar",
				},

				MLockTimeCnt:     1,
				MLockTimeSum:     3456,
				MRowsSentCnt:     1,
				MRowsSentSum:     4079,
				MRowsExaminedCnt: 1,
				MRowsExaminedSum: 4079,
				MFullScanCnt:     1,
				MFullScanSum:     1,
				MNoIndexUsedCnt:  1,
				MNoIndexUsedSum:  1,
			},
		}}
		assertCollected(t, c, expectedRequest)
	})

	t.Run("Test MongoDB Metrics conversion", func(t *testing.T) {
		c := &mockQanCollectorClient{}
		c.Test(t)
		defer c.AssertExpectations(t)

		client := &Client{
			c:  c,
			db: db,
			l:  logrus.WithField("test", t.Name()),
		}
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).Return(&qanpb.CollectResponse{}, nil)
		metricsBuckets := []*agentv1.MetricsBucket{
			{
				Common: &agentv1.MetricsBucket_Common{
					Queryid:     "some-mongo-query-id",
					Fingerprint: "INSERT peoples",
					Database:    "test",
					Schema:      "peoples",
					AgentId:     "b153f0d8-34e4-4635-9184-499161b4d12c",
					AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
					NumQueries:  1,
				},
				Mongodb: &agentv1.MetricsBucket_MongoDB{
					MResponseLengthSum: 60,
					MResponseLengthMin: 60,
					MResponseLengthMax: 60,
					MDocsExaminedSum:   20,
					MKeysExaminedSum:   10,
				},
			},
		}
		err := client.Collect(ctx, metricsBuckets)
		require.NoError(t, err)

		expectedRequest := &qanpb.CollectRequest{MetricsBucket: []*qanpb.MetricsBucket{
			{
				Queryid:     "some-mongo-query-id",
				Fingerprint: "INSERT peoples",
				Database:    "test",
				Schema:      "peoples",
				AgentId:     "b153f0d8-34e4-4635-9184-499161b4d12c",
				AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
				NumQueries:  1,
				ServiceId:   "1fce2502-ecc7-46d4-968b-18d7907f2543",
				ServiceName: "test-mongodb",
				ServiceType: "mongodb",
				NodeId:      "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:    "test-generic-node",
				NodeType:    "generic",
				NodeModel:   "test-node-model",
				Labels: map[string]string{
					"_agent_label":   "mongodb-baz",
					"_node_label":    "foo",
					"_service_label": "mongo-bar",
				},

				MResponseLengthSum: 60,
				MResponseLengthMin: 60,
				MResponseLengthMax: 60,
				MDocsExaminedSum:   20,
				MKeysExaminedSum:   10,
			},
		}}
		assertCollected(t, c, expectedRequest)
	})

	t.Run("Test PostgreSQL Metrics conversion", func(t *testing.T) {
		c := &mockQanCollectorClient{}
		c.Test(t)
		defer c.AssertExpectations(t)

		client := &Client{
			c:  c,
			db: db,
			l:  logrus.WithField("test", t.Name()),
		}
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).Return(&qanpb.CollectResponse{}, nil)
		metricsBuckets := []*agentv1.MetricsBucket{
			{
				Common: &agentv1.MetricsBucket_Common{
					Queryid:             "some-query-id",
					Fingerprint:         "SELECT /* AllCities */ * FROM city",
					Schema:              "pmm-agent",
					Tables:              []string{"city"},
					Username:            "pmm-agent",
					AgentId:             "29e14468-d479-4b4d-bfb7-4ac2fb865bac",
					PeriodStartUnixSecs: 1554116340,
					PeriodLengthSecs:    60,
					AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
					NumQueries:          1,
					MQueryTimeCnt:       1,
					MQueryTimeSum:       55,
				},
				Postgresql: &agentv1.MetricsBucket_PostgreSQL{
					MRowsCnt:               1,
					MRowsSum:               4079,
					MSharedBlksHitCnt:      1,
					MSharedBlksHitSum:      33,
					MSharedBlksReadCnt:     1,
					MSharedBlksReadSum:     2,
					MSharedBlksDirtiedCnt:  3,
					MSharedBlksDirtiedSum:  4,
					MSharedBlksWrittenCnt:  5,
					MSharedBlksWrittenSum:  6,
					MLocalBlksHitCnt:       7,
					MLocalBlksHitSum:       8,
					MLocalBlksReadCnt:      9,
					MLocalBlksReadSum:      10,
					MLocalBlksDirtiedCnt:   11,
					MLocalBlksDirtiedSum:   12,
					MLocalBlksWrittenCnt:   13,
					MLocalBlksWrittenSum:   14,
					MTempBlksReadCnt:       15,
					MTempBlksReadSum:       16,
					MTempBlksWrittenCnt:    17,
					MTempBlksWrittenSum:    18,
					MSharedBlkReadTimeCnt:  19,
					MSharedBlkReadTimeSum:  20,
					MSharedBlkWriteTimeCnt: 21,
					MSharedBlkWriteTimeSum: 22,
					MLocalBlkReadTimeCnt:   23,
					MLocalBlkReadTimeSum:   24,
					MLocalBlkWriteTimeCnt:  25,
					MLocalBlkWriteTimeSum:  26,
					MCpuSysTimeCnt:         27,
					MCpuSysTimeSum:         28,
					MCpuUserTimeCnt:        29,
					MCpuUserTimeSum:        30,
				},
			},
		}
		err := client.Collect(ctx, metricsBuckets)
		require.NoError(t, err)

		expectedRequest := &qanpb.CollectRequest{MetricsBucket: []*qanpb.MetricsBucket{
			{
				Queryid:             "some-query-id",
				Fingerprint:         "SELECT /* AllCities */ * FROM city",
				Schema:              "pmm-agent",
				Tables:              []string{"city"},
				Username:            "pmm-agent",
				AgentId:             "29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       55,
				ServiceName:         "test-postgresql",
				ServiceType:         "postgresql",
				ServiceId:           "9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
				NodeId:              "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:            "test-generic-node",
				NodeType:            "generic",
				NodeModel:           "test-node-model",
				Labels: map[string]string{
					"_agent_label":   "postgres-baz",
					"_node_label":    "foo",
					"_service_label": "bar",
				},

				MRowsSentCnt:           1,
				MRowsSentSum:           4079,
				MSharedBlksHitCnt:      1,
				MSharedBlksHitSum:      33,
				MSharedBlksReadCnt:     1,
				MSharedBlksReadSum:     2,
				MSharedBlksDirtiedCnt:  3,
				MSharedBlksDirtiedSum:  4,
				MSharedBlksWrittenCnt:  5,
				MSharedBlksWrittenSum:  6,
				MLocalBlksHitCnt:       7,
				MLocalBlksHitSum:       8,
				MLocalBlksReadCnt:      9,
				MLocalBlksReadSum:      10,
				MLocalBlksDirtiedCnt:   11,
				MLocalBlksDirtiedSum:   12,
				MLocalBlksWrittenCnt:   13,
				MLocalBlksWrittenSum:   14,
				MTempBlksReadCnt:       15,
				MTempBlksReadSum:       16,
				MTempBlksWrittenCnt:    17,
				MTempBlksWrittenSum:    18,
				MSharedBlkReadTimeCnt:  19,
				MSharedBlkReadTimeSum:  20,
				MSharedBlkWriteTimeCnt: 21,
				MSharedBlkWriteTimeSum: 22,
				MLocalBlkReadTimeCnt:   23,
				MLocalBlkReadTimeSum:   24,
				MLocalBlkWriteTimeCnt:  25,
				MLocalBlkWriteTimeSum:  26,
				MCpuSysTimeCnt:         27,
				MCpuSysTimeSum:         28,
				MCpuUserTimeCnt:        29,
				MCpuUserTimeSum:        30,
				HistogramItems:         []string{},
			},
		}}
		assertCollected(t, c, expectedRequest)
	})

	t.Run("Test conversion skips bad buckets", func(t *testing.T) {
		c := &mockQanCollectorClient{}
		c.Test(t)
		defer c.AssertExpectations(t)

		client := &Client{
			c:  c,
			db: db,
			l:  logrus.WithField("test", t.Name()),
		}
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).Return(&qanpb.CollectResponse{}, nil)
		metricsBuckets := []*agentv1.MetricsBucket{
			{
				Common: &agentv1.MetricsBucket_Common{
					AgentId: "no-such-agent",
				},
			},
		}
		err := client.Collect(ctx, metricsBuckets)
		require.NoError(t, err)

		expectedRequest := &qanpb.CollectRequest{MetricsBucket: []*qanpb.MetricsBucket{}}
		assertCollected(t, c, expectedRequest)
		// Exactly one request, so batching can never turn "nothing to send" into no call at
		// all, nor into a second, trailing empty one.
		c.AssertNumberOfCalls(t, "Collect", 1)
	})
}

func TestClientPerformance(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	for _, str := range []reform.Struct{
		&models.Service{
			ServiceID:    "0d350868-4d85-4884-b972-dff130129c23",
			ServiceType:  models.MySQLServiceType,
			ServiceName:  "test-mysql",
			NodeID:       "pmm-server",
			Address:      new("5.6.7.8"),
			Port:         new(uint16(3306)),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			AgentType:    models.QANMySQLPerfSchemaAgentType,
			ServiceID:    new("0d350868-4d85-4884-b972-dff130129c23"),
			PMMAgentID:   new("pmm-server"),
			CustomLabels: []byte(`{"_agent_label": "baz"}`),
			ListenPort:   new(uint16(12345)),
		},
	} {
		require.NoError(t, db.Insert(str), "%+v", str)
	}

	ctx := logger.Set(t.Context(), t.Name())
	c := &mockQanCollectorClient{}
	c.Test(t)
	c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).Return(&qanpb.CollectResponse{}, nil)
	defer c.AssertExpectations(t)

	reformL.Reset()
	defer func() {
		assert.Equal(t, 3, reformL.Requests())
	}()

	client := &Client{
		c:  c,
		db: db,
		l:  logrus.WithField("test", t.Name()),
	}

	const bucketsN = 1000
	metricsBuckets := make([]*agentv1.MetricsBucket, bucketsN)
	for i := range metricsBuckets {
		metricsBuckets[i] = &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid: fmt.Sprintf("bucket %d", i),
				AgentId: "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			},
		}
	}
	err := client.Collect(ctx, metricsBuckets)
	require.NoError(t, err)

	expectedBuckets := make([]*qanpb.MetricsBucket, bucketsN)
	for i := range expectedBuckets {
		expectedBuckets[i] = &qanpb.MetricsBucket{
			Queryid:     fmt.Sprintf("bucket %d", i),
			ServiceName: "test-mysql",
			NodeId:      "pmm-server",
			NodeName:    "pmm-server",
			NodeType:    "container",
			ServiceId:   "0d350868-4d85-4884-b972-dff130129c23",
			ServiceType: "mysql",
			AgentId:     "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			Labels: map[string]string{
				"_agent_label":   "baz",
				"_service_label": "bar",
			},
		}
	}
	assertCollected(t, c, &qanpb.CollectRequest{MetricsBucket: expectedBuckets})
}

func TestClientCollectRequestSize(t *testing.T) {
	// qan-api2 refuses any CollectRequest larger than this and the limit is deliberately
	// not raised; see qan-api2/main.go.
	const qanAPIMaxRecvMsgSize = 20 * 1024 * 1024

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	for _, str := range []reform.Struct{
		&models.Service{
			ServiceID:   "0d350868-4d85-4884-b972-dff130129c23",
			ServiceType: models.MongoDBServiceType,
			ServiceName: "test-mongodb",
			NodeID:      "pmm-server",
			Address:     new("5.6.7.8"),
			Port:        new(uint16(27017)),
		},
		&models.Agent{
			AgentID:    "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			AgentType:  models.QANMongoDBProfilerAgentType,
			ServiceID:  new("0d350868-4d85-4884-b972-dff130129c23"),
			PMMAgentID: new("pmm-server"),
			ListenPort: new(uint16(12345)),
		},
	} {
		require.NoError(t, db.Insert(str), "%+v", str)
	}

	// collect runs Collect over buckets carrying a fingerprint and an example of the given
	// length, and reports the size of every request sent and the query IDs that arrived.
	collect := func(t *testing.T, queryLens []int) ([]int, []string) {
		t.Helper()

		ctx := logger.Set(t.Context(), t.Name())

		var (
			sentSizes    []int
			sentQueryIDs []string
		)
		c := &mockQanCollectorClient{}
		c.Test(t)
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeFor[*qanpb.CollectRequest]().String())).
			Return(&qanpb.CollectResponse{}, nil).
			Run(func(args mock.Arguments) {
				req := args.Get(1).(*qanpb.CollectRequest)
				sentSizes = append(sentSizes, proto.Size(req))
				for _, b := range req.MetricsBucket {
					sentQueryIDs = append(sentQueryIDs, b.Queryid)
				}
			})
		defer c.AssertExpectations(t)

		metricsBuckets := make([]*agentv1.MetricsBucket, len(queryLens))
		for i, queryLen := range queryLens {
			query := strings.Repeat("A", queryLen)
			metricsBuckets[i] = &agentv1.MetricsBucket{
				Common: &agentv1.MetricsBucket_Common{
					Queryid:     fmt.Sprintf("bucket %d", i),
					AgentId:     "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
					Fingerprint: query,
					Example:     query,
				},
			}
		}

		client := &Client{
			c:  c,
			db: db,
			l:  logrus.WithField("test", t.Name()),
		}
		require.NoError(t, client.Collect(ctx, metricsBuckets))

		require.NotEmpty(t, sentSizes, "Collect sent no request at all")
		for i, size := range sentSizes {
			assert.LessOrEqualf(t, size, qanAPIMaxRecvMsgSize,
				"CollectRequest %d of %d is %d bytes, over qan-api2's %d byte limit: "+
					"qan-api2 answers ResourceExhausted and every bucket in that request is lost",
				i+1, len(sentSizes), size, qanAPIMaxRecvMsgSize)
		}

		return sentSizes, sentQueryIDs
	}

	t.Run("A payload over the limit is split, losing nothing", func(t *testing.T) {
		// PMM-14976: a service added with --max-query-length=20480 produces buckets carrying
		// a ~20 KB fingerprint and a ~20 KB example each, so the payload crosses 20 MiB at
		// roughly 500 buckets - far below the bucket count Collect used to slice on.
		const (
			bucketsN = 1500
			queryLen = 20480
		)
		queryLens := make([]int, bucketsN)
		for i := range queryLens {
			queryLens[i] = queryLen
		}

		sentSizes, sentQueryIDs := collect(t, queryLens)
		assert.Greater(t, len(sentSizes), 1, "a payload this large must be split across requests")

		// Splitting must not lose or reorder a single bucket.
		expectedQueryIDs := make([]string, bucketsN)
		for i := range expectedQueryIDs {
			expectedQueryIDs[i] = fmt.Sprintf("bucket %d", i)
		}
		assert.Equal(t, expectedQueryIDs, sentQueryIDs)
	})

	t.Run("A bucket QAN cannot accept costs only itself", func(t *testing.T) {
		// With max_query_length unlimited a single bucket can exceed the limit outright. It
		// has to be dropped: sending it would have QAN reject the request, and Collect stops
		// at the first rejection, so every bucket behind it would be lost too.
		_, sentQueryIDs := collect(t, []int{8, qanAPIMaxRecvMsgSize, 8})
		assert.Equal(t, []string{"bucket 0", "bucket 2"}, sentQueryIDs)
	})
}

func TestNextCollectBatch(t *testing.T) {
	sizesOf := func(n, each int) []int {
		sizes := make([]int, n)
		for i := range sizes {
			sizes[i] = each
		}

		return sizes
	}

	t.Run("Empty input yields an empty batch", func(t *testing.T) {
		assert.Equal(t, 0, nextCollectBatch(nil))
		assert.Equal(t, 0, nextCollectBatch([]int{}))
	})

	t.Run("Small buckets are capped by count", func(t *testing.T) {
		assert.Equal(t, maxCollectRequestBuckets, nextCollectBatch(sizesOf(maxCollectRequestBuckets+1, 8)))
	})

	t.Run("Large buckets are capped by size", func(t *testing.T) {
		const each = maxCollectRequestSize / 4
		assert.Equal(t, 4, nextCollectBatch(sizesOf(10, each)))
	})

	t.Run("The batch stops before crossing the budget", func(t *testing.T) {
		// One byte over on the third bucket, so only two may go.
		assert.Equal(t, 2, nextCollectBatch([]int{maxCollectRequestSize / 2, maxCollectRequestSize / 2, 1}))
	})

	t.Run("A bucket over the budget is sent on its own", func(t *testing.T) {
		assert.Equal(t, 1, nextCollectBatch([]int{maxCollectRequestSize + 1, 8}))
	})
}

func TestDropOversizedBuckets(t *testing.T) {
	client := &Client{l: logrus.WithField("test", t.Name())}

	fits := func(id string, queryLen int) *qanpb.MetricsBucket {
		return &qanpb.MetricsBucket{Queryid: id, Fingerprint: strings.Repeat("A", queryLen)}
	}
	undeliverable := fits("undeliverable", qanCollectRequestLimit)

	idsOf := func(buckets []*qanpb.MetricsBucket) []string {
		ids := make([]string, len(buckets))
		for i, b := range buckets {
			ids[i] = b.Queryid
		}

		return ids
	}

	t.Run("Keeps every bucket QAN would accept", func(t *testing.T) {
		// Over the batching budget but still under what QAN takes: this one must survive and
		// travel in a request of its own.
		big := fits("big", maxCollectRequestSize+1)
		kept, sizes := client.dropOversizedBuckets([]*qanpb.MetricsBucket{fits("a", 8), big})
		assert.Equal(t, []string{"a", "big"}, idsOf(kept))
		assert.Len(t, sizes, 2)
	})

	t.Run("Drops only what QAN would reject, in order", func(t *testing.T) {
		kept, sizes := client.dropOversizedBuckets([]*qanpb.MetricsBucket{
			fits("first", 8), undeliverable, fits("second", 8),
		})
		assert.Equal(t, []string{"first", "second"}, idsOf(kept))
		assert.Len(t, sizes, 2)
	})
}
