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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/version"
)

func TestAgentHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node with Service",
			},
			&models.Node{
				NodeID:   "N2",
				NodeType: models.GenericNodeType,
				NodeName: "N2 with PushMetrics",
			},

			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service on N1",
				NodeID:      "N1",
				Address:     pointer.ToStringOrNil("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(3306),
			},

			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
			},
			&models.Agent{
				AgentID:      "A2",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S1"),
			},
			&models.Agent{
				AgentID:      "A3",
				AgentType:    models.NodeExporterType,
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
			},
			&models.Agent{
				AgentID:      "A4",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N2"),
			},
			&models.Agent{
				AgentID:      "A5",
				AgentType:    models.NodeExporterType,
				PMMAgentID:   pointer.ToString("A4"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N2"),
				ListenPort:   pointer.ToUint16(8200),
				ExporterOptions: models.ExporterOptions{
					PushMetrics: true,
				},
			},
			&models.Agent{
				AgentID:      "A6",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("A4"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N2"),
				ListenPort:   pointer.ToUint16(8200),
			},
			&models.Agent{
				AgentID:       "A7",
				AgentType:     models.PostgresExporterType,
				PMMAgentID:    pointer.ToString("A4"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
				ExporterOptions: models.ExporterOptions{
					MetricsResolutions: &models.MetricsResolutions{
						HR: 1 * time.Minute,
						MR: 5 * time.Minute,
						LR: 15 * time.Minute,
					},
				},
				PostgreSQLOptions: models.PostgreSQLOptions{
					SSLCa:   "ssl_ca",
					SSLCert: "ssl_cert",
					SSLKey:  "ssl_key",
				},
			},
			&models.Agent{
				AgentID:       "A8",
				AgentType:     models.MongoDBExporterType,
				PMMAgentID:    pointer.ToString("A8"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "tls_certificate_key",
					TLSCertificateKeyFilePassword: "tls_certificate_key_file_password",
					TLSCa:                         "tls_ca",
					AuthenticationMechanism:       "authentication_mechanism",
					AuthenticationDatabase:        "authentication_database",
					StatsCollections:              nil,
					CollectionsLimit:              0, // no limit
				},
			},
			&models.Agent{
				AgentID:       "A9",
				AgentType:     models.MongoDBExporterType,
				PMMAgentID:    pointer.ToString("A9"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "tls_certificate_key",
					TLSCertificateKeyFilePassword: "tls_certificate_key_file_password",
					TLSCa:                         "tls_ca",
					AuthenticationMechanism:       "authentication_mechanism",
					AuthenticationDatabase:        "authentication_database",
					StatsCollections:              []string{"col1", "col2", "col3"},
					CollectionsLimit:              79014,
					EnableAllCollectors:           true,
				},
			},
			&models.Agent{
				AgentID:       "A10",
				AgentType:     models.MongoDBExporterType,
				PMMAgentID:    pointer.ToString("A10"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("AgentsForNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAgents(q, models.AgentFilters{NodeID: "N1"})
		require.NoError(t, err)
		expected := []*models.Agent{
			{
				CreatedAt:     now,
				UpdatedAt:     now,
				Status:        models.AgentStatusUnknown,
				AgentID:       "A10",
				AgentType:     models.MongoDBExporterType,
				PMMAgentID:    pointer.ToString("A10"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
			},
			{
				AgentID:      "A3",
				AgentType:    models.NodeExporterType,
				PMMAgentID:   pointer.ToStringOrNil("A1"),
				RunsOnNodeID: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
				NodeID:       pointer.ToString("N1"),
				Status:       models.AgentStatusUnknown,
			},
			{
				AgentID:       "A7",
				AgentType:     "postgres_exporter",
				NodeID:        pointer.ToStringOrNil("N1"),
				PMMAgentID:    pointer.ToStringOrNil("A4"),
				CreatedAt:     now,
				UpdatedAt:     now,
				Status:        models.AgentStatusUnknown,
				ListenPort:    pointer.ToUint16OrNil(8200),
				TLS:           true,
				TLSSkipVerify: true,
				ExporterOptions: models.ExporterOptions{
					MetricsResolutions: &models.MetricsResolutions{
						HR: 1 * time.Minute,
						MR: 5 * time.Minute,
						LR: 15 * time.Minute,
					},
				},
				PostgreSQLOptions: models.PostgreSQLOptions{
					SSLCa:   "ssl_ca",
					SSLCert: "ssl_cert",
					SSLKey:  "ssl_key",
				},
			},
			{
				AgentID:       "A8",
				AgentType:     "mongodb_exporter",
				NodeID:        pointer.ToStringOrNil("N1"),
				PMMAgentID:    pointer.ToStringOrNil("A8"),
				CreatedAt:     now,
				UpdatedAt:     now,
				Status:        models.AgentStatusUnknown,
				ListenPort:    pointer.ToUint16OrNil(8200),
				TLS:           true,
				TLSSkipVerify: true,
				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "tls_certificate_key",
					TLSCertificateKeyFilePassword: "tls_certificate_key_file_password",
					TLSCa:                         "tls_ca",
					AuthenticationMechanism:       "authentication_mechanism",
					AuthenticationDatabase:        "authentication_database",
					StatsCollections:              nil,
					CollectionsLimit:              0, // no limit
				},
			},
			{
				AgentID:       "A9",
				AgentType:     "mongodb_exporter",
				NodeID:        pointer.ToStringOrNil("N1"),
				PMMAgentID:    pointer.ToStringOrNil("A9"),
				CreatedAt:     now,
				UpdatedAt:     now,
				Status:        models.AgentStatusUnknown,
				ListenPort:    pointer.ToUint16OrNil(8200),
				TLS:           true,
				TLSSkipVerify: true,
				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "tls_certificate_key",
					TLSCertificateKeyFilePassword: "tls_certificate_key_file_password",
					TLSCa:                         "tls_ca",
					AuthenticationMechanism:       "authentication_mechanism",
					AuthenticationDatabase:        "authentication_database",
					StatsCollections:              []string{"col1", "col2", "col3"},
					CollectionsLimit:              79014,
					EnableAllCollectors:           true,
				},
			},
		}
		assert.Equal(t, expected, agents)
	})

	t.Run("AgentsRunningByPMMAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: "A1"})
		require.NoError(t, err)
		expected := []*models.Agent{{
			AgentID:      "A2",
			AgentType:    models.MySQLdExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			ServiceID:    pointer.ToString("S1"),
			RunsOnNodeID: nil,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       models.AgentStatusUnknown,
		}, {
			AgentID:      "A3",
			AgentType:    models.NodeExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			NodeID:       pointer.ToString("N1"),
			RunsOnNodeID: nil,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       models.AgentStatusUnknown,
		}}
		assert.Equal(t, expected, agents)
	})

	t.Run("AgentsRunningByPMMAgentAndType", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: "A1", AgentType: pointerToAgentType(models.MySQLdExporterType)})
		require.NoError(t, err)
		expected := []*models.Agent{{
			AgentID:      "A2",
			AgentType:    models.MySQLdExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			ServiceID:    pointer.ToString("S1"),
			RunsOnNodeID: nil,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       models.AgentStatusUnknown,
		}}
		assert.Equal(t, expected, agents)
	})

	t.Run("AgentsForService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAgents(q, models.AgentFilters{ServiceID: "S1"})
		require.NoError(t, err)
		expected := []*models.Agent{{
			AgentID:      "A2",
			AgentType:    models.MySQLdExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			ServiceID:    pointer.ToString("S1"),
			RunsOnNodeID: nil,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       models.AgentStatusUnknown,
		}}
		assert.Equal(t, expected, agents)

		agents, err = models.FindAgents(q, models.AgentFilters{ServiceID: "S1", AgentType: pointerToAgentType(models.MySQLdExporterType)})
		require.NoError(t, err)
		expected = []*models.Agent{{
			AgentID:      "A2",
			AgentType:    models.MySQLdExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			ServiceID:    pointer.ToString("S1"),
			RunsOnNodeID: nil,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       models.AgentStatusUnknown,
		}}
		assert.Equal(t, expected, agents)

		agents, err = models.FindAgents(q, models.AgentFilters{ServiceID: "S1", AgentType: pointerToAgentType(models.MongoDBExporterType)})
		require.NoError(t, err)
		assert.Equal(t, []*models.Agent{}, agents)
	})

	t.Run("RemoveAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agent, err := models.RemoveAgent(q, "", models.RemoveRestrict)
		assert.Nil(t, agent)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Agent ID.`), err)

		agent, err = models.RemoveAgent(q, models.PMMServerAgentID, models.RemoveRestrict)
		assert.Nil(t, agent)
		tests.AssertGRPCError(t, status.New(codes.PermissionDenied, `pmm-agent on PMM Server can't be removed.`), err)

		agent, err = models.RemoveAgent(q, "A0", models.RemoveRestrict)
		assert.Nil(t, agent)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID A0 not found.`), err)

		agent, err = models.RemoveAgent(q, "A1", models.RemoveRestrict)
		assert.Nil(t, agent)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `pmm-agent with ID A1 has agents.`), err)

		expected := &models.Agent{
			AgentID:      "A1",
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString("N1"),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		agent, err = models.RemoveAgent(q, "A1", models.RemoveCascade)
		assert.Equal(t, expected, agent)
		assert.NoError(t, err)
		_, err = models.FindAgentByID(q, "A1")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID A1 not found.`), err)
	})

	t.Run("FindPMMAgentsForNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindPMMAgentsRunningOnNode(q, "N1")
		require.NoError(t, err)
		assert.Equal(t, "A1", agents[0].AgentID)

		// find with non existing node.
		agents, err = models.FindPMMAgentsRunningOnNode(q, "X1")
		require.NoError(t, err)
		assert.Empty(t, agents)
	})

	t.Run("FindPMMAgentsForServicesOnNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindPMMAgentsForServicesOnNode(q, "N1")
		require.NoError(t, err)
		t.Log(agents, err)
		assert.Equal(t, "A1", agents[0].AgentID)
	})

	t.Run("FindPMMAgentsForService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindPMMAgentsForService(q, "S1")
		require.NoError(t, err)
		t.Log(agents, err)
		assert.Equal(t, "A1", agents[0].AgentID)

		// find with non existing service.
		_, err = models.FindPMMAgentsForService(q, "X1")
		require.Error(t, err)
	})

	t.Run("CreateExternalExporter", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			agent, err := models.CreateExternalExporter(q, &models.CreateExternalExporterParams{
				RunsOnNodeID: "N1",
				ServiceID:    "S1",
				ListenPort:   9104,
			})
			require.NoError(t, err)
			assert.Equal(t, &models.Agent{
				AgentID:      agent.AgentID,
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: pointer.ToString("N1"),
				ServiceID:    pointer.ToString("S1"),
				ListenPort:   pointer.ToUint16(9104),
				ExporterOptions: models.ExporterOptions{
					MetricsPath:   "/metrics",
					MetricsScheme: "http",
				},
				CreatedAt: now,
				UpdatedAt: now,
			}, agent)
		})
		t.Run("Invalid listen port", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			agent, err := models.CreateExternalExporter(q, &models.CreateExternalExporterParams{
				RunsOnNodeID: "N1",
				ServiceID:    "S1",
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Listen port should be between 1 and 65535."), err)
			require.Nil(t, agent)
		})
	})

	t.Run("TestFindPMMAgentsForVersion", func(t *testing.T) {
		l := logrus.WithField("component", "test")
		agentInvalid := &models.Agent{
			Version: pointer.ToString("invalid"),
		}
		agent260 := &models.Agent{
			Version: pointer.ToString("2.6.0"),
		}
		agent270 := &models.Agent{
			Version: pointer.ToString("2.7.0"),
		}
		agents := []*models.Agent{agentInvalid, agent260, agent270}

		result := models.FindPMMAgentsForVersion(l, agents, nil)
		assert.Equal(t, []*models.Agent{agentInvalid, agent260, agent270}, result)

		result = models.FindPMMAgentsForVersion(l, agents, version.MustParse("2.5.0"))
		assert.Equal(t, []*models.Agent{agent260, agent270}, result)

		result = models.FindPMMAgentsForVersion(l, agents, version.MustParse("2.7.0"))
		assert.Equal(t, []*models.Agent{agent270}, result)

		result = models.FindPMMAgentsForVersion(l, agents, version.MustParse("2.42.777"))
		assert.Empty(t, result)
	})

	t.Run("FindAgentsForScrapeConfig", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAgentsForScrapeConfig(q, pointer.ToString("A4"), true)
		require.NoError(t, err)
		assert.Equal(t, "A5", agents[0].AgentID)

		// find with empty response.
		agents, err = models.FindAgentsForScrapeConfig(q, pointer.ToString("A1"), true)
		assert.Equal(t, 0, len(agents))
		require.NoError(t, err)

		// find all agents without push_metrics
		agents, err = models.FindAgentsForScrapeConfig(q, nil, false)
		assert.Equal(t, 5, len(agents))
		assert.Equal(t, "A10", agents[0].AgentID)
		assert.Equal(t, "A8", agents[1].AgentID)
		assert.Equal(t, "A9", agents[2].AgentID)
		assert.Equal(t, "A6", agents[3].AgentID)
		assert.Equal(t, "A7", agents[4].AgentID)

		require.NoError(t, err)
	})

	t.Run("FindAllPMMAgentsIDs", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.FindAllPMMAgentsIDs(q)
		require.NoError(t, err)
		require.Len(t, agents, 3, agents)
		assert.Equal(t, []string{"A1", "A4", models.PMMServerAgentID}, agents)
	})
}

func pointerToAgentType(agentType models.AgentType) *models.AgentType {
	return &agentType
}
