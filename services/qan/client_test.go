// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"database/sql"
	"reflect"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestClient(t *testing.T) {
	t.Run("Check CollectRequest conversion", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SetupFixtures)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		ctx := logger.Set(context.Background(), t.Name())
		defer func() {
			assert.NoError(t, db.DBInterface().(*sql.DB).Close())
		}()

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeType:     models.GenericNodeType,
				NodeName:     "test-generic-node",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_node_label": "foo"}`),
				NodeModel:    "test-node-model",
			},
			&models.Agent{
				AgentID:      "/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
			},

			&models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				ServiceType:  models.MySQLServiceType,
				ServiceName:  "test-mysql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.QANMySQLPerfSchemaAgentType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				CustomLabels: []byte(`{"_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},
			&models.AgentService{
				AgentID:   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				ServiceID: "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
			},

			&models.Service{
				ServiceID:    "/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
				ServiceType:  models.PostgreSQLServiceType,
				ServiceName:  "test-postgresql",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_service_label": "bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				AgentType:    models.QANPostgreSQLPgStatementsAgentType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				CustomLabels: []byte(`{"_agent_label": "postgres-baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},
			&models.AgentService{
				AgentID:   "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
				ServiceID: "/service_id/9cffbdd4-3cd2-47f8-a5f9-a749c3d5fee1",
			},

			&models.Service{
				ServiceID:    "/service_id/1fce2502-ecc7-46d4-968b-18d7907f2543",
				ServiceType:  models.MongoDBServiceType,
				ServiceName:  "test-mongodb",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_service_label": "mongo-bar"}`),
			},

			&models.Agent{
				AgentID:      "/agent_id/b153f0d8-34e4-4635-9184-499161b4d12c",
				AgentType:    models.QANMongoDBProfilerAgentType,
				PMMAgentID:   pointer.ToString("/agent_id/217907dc-d34d-4e2e-aa84-a1b765d49853"),
				CustomLabels: []byte(`{"_agent_label": "mongodb-baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			},
			&models.AgentService{
				AgentID:   "/agent_id/b153f0d8-34e4-4635-9184-499161b4d12c",
				ServiceID: "/service_id/1fce2502-ecc7-46d4-968b-18d7907f2543",
			},
		} {
			require.NoError(t, db.Insert(str), "%+v", str)
		}

		t.Run("Test MySQL Metrics conversion", func(t *testing.T) {
			c := new(mockQanCollectorClient)
			c.Test(t)
			client := Client{
				c:  c,
				db: db,
				l:  logrus.WithField("component", "qan-test"),
			}
			c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
			metricsBuckets := []*agentpb.MetricsBucket{
				{
					Common: &agentpb.MetricsBucket_Common{
						Queryid:             "some-query-id",
						Fingerprint:         "SELECT * FROM `city`",
						Schema:              "world",
						AgentId:             "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
						PeriodStartUnixSecs: 1554116340,
						PeriodLengthSecs:    60,
						AgentType:           inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
						Example:             "SELECT /* AllCities */ * FROM city",
						ExampleFormat:       agentpb.ExampleFormat_EXAMPLE,
						ExampleType:         agentpb.ExampleType_RANDOM,
						NumQueries:          1,
						MQueryTimeCnt:       1,
						MQueryTimeSum:       1234,
					},
					Mysql: &agentpb.MetricsBucket_MySQL{
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
					AgentId:             "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					PeriodStartUnixSecs: 1554116340,
					PeriodLengthSecs:    60,
					AgentType:           inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
					Example:             "SELECT /* AllCities */ * FROM city",
					ExampleFormat:       qanpb.ExampleFormat_EXAMPLE,
					ExampleType:         qanpb.ExampleType_RANDOM,
					NumQueries:          1,
					MQueryTimeCnt:       1,
					MQueryTimeSum:       1234,
					ServiceName:         "test-mysql",
					ServiceType:         "mysql",
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
			c := new(mockQanCollectorClient)
			c.Test(t)
			client := Client{
				c:  c,
				db: db,
				l:  logrus.WithField("component", "qan-test"),
			}
			c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
			metricsBuckets := []*agentpb.MetricsBucket{
				{
					Common: &agentpb.MetricsBucket_Common{
						Queryid:     "some-mongo-query-id",
						Fingerprint: "INSERT peoples",
						Database:    "test",
						Schema:      "peoples",
						AgentId:     "/agent_id/b153f0d8-34e4-4635-9184-499161b4d12c",
						AgentType:   inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT,
						NumQueries:  1,
					},
					Mongodb: &agentpb.MetricsBucket_MongoDB{
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
					AgentId:     "/agent_id/b153f0d8-34e4-4635-9184-499161b4d12c",
					AgentType:   inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT,
					NumQueries:  1,
					ServiceName: "test-mongodb",
					ServiceType: "mongodb",
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
			c := new(mockQanCollectorClient)
			c.Test(t)
			client := Client{
				c:  c,
				db: db,
				l:  logrus.WithField("component", "qan-test"),
			}
			c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
			metricsBuckets := []*agentpb.MetricsBucket{
				{
					Common: &agentpb.MetricsBucket_Common{
						Queryid:             "some-query-id",
						Fingerprint:         "SELECT /* AllCities */ * FROM city",
						Schema:              "pmm-agent",
						Tables:              []string{"city"},
						Username:            "pmm-agent",
						AgentId:             "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
						PeriodStartUnixSecs: 1554116340,
						PeriodLengthSecs:    60,
						AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
						NumQueries:          1,
						MQueryTimeCnt:       1,
						MQueryTimeSum:       55,
					},
					Postgresql: &agentpb.MetricsBucket_PostgreSQL{
						MSharedBlksHitCnt: 1,
						MSharedBlksHitSum: 33,
						MRowsCnt:          1,
						MRowsSum:          4079,
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
					AgentId:             "/agent_id/29e14468-d479-4b4d-bfb7-4ac2fb865bac",
					PeriodStartUnixSecs: 1554116340,
					PeriodLengthSecs:    60,
					AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
					NumQueries:          1,
					MQueryTimeCnt:       1,
					MQueryTimeSum:       55,
					ServiceName:         "test-postgresql",
					ServiceType:         "postgresql",
					NodeModel:           "test-node-model",
					Labels: map[string]string{
						"_agent_label":   "postgres-baz",
						"_node_label":    "foo",
						"_service_label": "bar",
					},

					MSharedBlksHitCnt: 1,
					MSharedBlksHitSum: 33,
					MRowsSentCnt:      1,
					MRowsSentSum:      4079,
				},
			}}
			c.AssertCalled(t, "Collect", ctx, expectedRequest)
		})

		t.Run("Test conversion skips bad buckets", func(t *testing.T) {
			c := new(mockQanCollectorClient)
			c.Test(t)
			client := Client{
				c:  c,
				db: db,
				l:  logrus.WithField("component", "qan-test"),
			}
			c.On("Collect", ctx, mock.AnythingOfType(reflect.TypeOf(&qanpb.CollectRequest{}).String())).Return(&qanpb.CollectResponse{}, nil)
			metricsBuckets := []*agentpb.MetricsBucket{
				{
					Common: &agentpb.MetricsBucket_Common{
						AgentId: "no-such-agent",
					},
				},
			}
			err := client.Collect(ctx, metricsBuckets)
			require.NoError(t, err)

			expectedRequest := &qanpb.CollectRequest{MetricsBucket: []*qanpb.MetricsBucket{}}
			c.AssertCalled(t, "Collect", ctx, expectedRequest)
		})
	})
}
