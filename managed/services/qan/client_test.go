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
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/sqlmetrics"
)

func TestClient(t *testing.T) {
	sqlDB := testdb.Open(t, database.SetupFixtures, nil)
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	ctx := logger.Set(context.Background(), t.Name())
	defer func() {
		assert.NoError(t, sqlDB.Close())
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
			RunsOnNodeID: pointer.ToString("cc663f36-18ca-40a1-aea9-c6310bb4738d"),
		},

		&models.Service{
			ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
			ServiceType:  models.MySQLServiceType,
			ServiceName:  "test-mysql",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      pointer.ToString("5.6.7.8"),
			Port:         pointer.ToUint16(3306),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
			AgentType:    models.QANMySQLPerfSchemaAgentType,
			PMMAgentID:   pointer.ToString("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    pointer.ToString("014647c3-b2f5-44eb-94f4-d943260a968c"),
			CustomLabels: []byte(`{"_agent_label": "baz"}`),
			ListenPort:   pointer.ToUint16(12345),
		},

		&models.Service{
			ServiceID:    "9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
			ServiceType:  models.PostgreSQLServiceType,
			ServiceName:  "test-postgresql",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      pointer.ToString("5.6.7.8"),
			Port:         pointer.ToUint16(5432),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "29e14468-d479-4b4d-bfb7-4ac2fb865bac",
			AgentType:    models.QANPostgreSQLPgStatementsAgentType,
			PMMAgentID:   pointer.ToString("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    pointer.ToString("9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1"),
			CustomLabels: []byte(`{"_agent_label": "postgres-baz"}`),
			ListenPort:   pointer.ToUint16(12345),
		},

		&models.Service{
			ServiceID:    "1fce2502-ecc7-46d4-968b-18d7907f2543",
			ServiceType:  models.MongoDBServiceType,
			ServiceName:  "test-mongodb",
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:      pointer.ToString("5.6.7.8"),
			Port:         pointer.ToUint16(27017),
			CustomLabels: []byte(`{"_service_label": "mongo-bar"}`),
		},

		&models.Agent{
			AgentID:      "b153f0d8-34e4-4635-9184-499161b4d12c",
			AgentType:    models.QANMongoDBProfilerAgentType,
			PMMAgentID:   pointer.ToString("217907dc-d34d-4e2e-aa84-a1b765d49853"),
			ServiceID:    pointer.ToString("1fce2502-ecc7-46d4-968b-18d7907f2543"),
			CustomLabels: []byte(`{"_agent_label": "mongodb-baz"}`),
			ListenPort:   pointer.ToUint16(12345),
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
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
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
		c.AssertCalled(t, "Collect", ctx, expectedRequest)
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
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
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
			},
		}}
		c.AssertCalled(t, "Collect", ctx, expectedRequest)
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
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
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
		c.AssertCalled(t, "Collect", ctx, expectedRequest)
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
		c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
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
		c.AssertCalled(t, "Collect", ctx, expectedRequest)
	})
}

func TestClientPerformance(t *testing.T) {
	sqlDB := testdb.Open(t, database.SetupFixtures, nil)
	reformL := sqlmetrics.NewReform("test", "test", t.Logf)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reformL)
	defer func() {
		assert.NoError(t, sqlDB.Close())
	}()

	for _, str := range []reform.Struct{
		&models.Service{
			ServiceID:    "0d350868-4d85-4884-b972-dff130129c23",
			ServiceType:  models.MySQLServiceType,
			ServiceName:  "test-mysql",
			NodeID:       "pmm-server",
			Address:      pointer.ToString("5.6.7.8"),
			Port:         pointer.ToUint16(3306),
			CustomLabels: []byte(`{"_service_label": "bar"}`),
		},

		&models.Agent{
			AgentID:      "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			AgentType:    models.QANMySQLPerfSchemaAgentType,
			ServiceID:    pointer.ToString("0d350868-4d85-4884-b972-dff130129c23"),
			PMMAgentID:   pointer.ToString("pmm-server"),
			CustomLabels: []byte(`{"_agent_label": "baz"}`),
			ListenPort:   pointer.ToUint16(12345),
		},
	} {
		require.NoError(t, db.Insert(str), "%+v", str)
	}

	ctx := logger.Set(context.Background(), t.Name())
	c := &mockQanCollectorClient{}
	c.Test(t)
	c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
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
			NodeType:    "generic",
			ServiceId:   "0d350868-4d85-4884-b972-dff130129c23",
			ServiceType: "mysql",
			AgentId:     "6b74c6bf-642d-43f0-bee1-0faddd1a2e28",
			Labels: map[string]string{
				"_agent_label":   "baz",
				"_service_label": "bar",
			},
		}
	}
	c.AssertCalled(t, "Collect", ctx, &qanpb.CollectRequest{MetricsBucket: expectedBuckets})
}
