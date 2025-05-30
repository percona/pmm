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
				AgentID:      "A12",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
			},
			&models.Agent{
				AgentID:       "A8",
				AgentType:     models.MongoDBExporterType,
				PMMAgentID:    pointer.ToString("A12"),
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
				PMMAgentID:    pointer.ToString("A12"),
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
				PMMAgentID:    pointer.ToString("A12"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
			},
			&models.Agent{
				AgentID:      "A11",
				AgentType:    models.NomadAgentType,
				PMMAgentID:   pointer.ToString("A12"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
				ListenPort:   pointer.ToUint16(8201),
				ExporterOptions: models.ExporterOptions{
					PushMetrics: true,
				},
			},
		} {
			if v, ok := str.(*models.Agent); ok {
				encryptedAgent := models.EncryptAgent(*v)
				str = &encryptedAgent
			}
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
				PMMAgentID:    pointer.ToString("A12"),
				RunsOnNodeID:  nil,
				NodeID:        pointer.ToString("N1"),
				ListenPort:    pointer.ToUint16(8200),
				TLS:           true,
				TLSSkipVerify: true,
			},
			{
				CreatedAt:    now,
				UpdatedAt:    now,
				Status:       models.AgentStatusUnknown,
				AgentID:      "A11",
				AgentType:    models.NomadAgentType,
				PMMAgentID:   pointer.ToString("A12"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
				ListenPort:   pointer.ToUint16(8201),
				ExporterOptions: models.ExporterOptions{
					PushMetrics: true,
				},
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
				PMMAgentID:    pointer.ToStringOrNil("A12"),
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
				PMMAgentID:    pointer.ToStringOrNil("A12"),
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

		agents, err := models.FindAgents(q, models.AgentFilters{PMMAgentID: "A1", AgentType: pointer.To(models.MySQLdExporterType)})
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

		agents, err = models.FindAgents(q, models.AgentFilters{ServiceID: "S1", AgentType: pointer.To(models.MySQLdExporterType)})
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

		agents, err = models.FindAgents(q, models.AgentFilters{ServiceID: "S1", AgentType: pointer.To(models.MongoDBExporterType)})
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
		assert.Empty(t, agents)
		require.NoError(t, err)

		// find all agents without push_metrics
		agents, err = models.FindAgentsForScrapeConfig(q, nil, false)
		assert.Len(t, agents, 5)
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
		require.Len(t, agents, 4, agents)
		assert.Equal(t, []string{"A1", "A12", "A4", models.PMMServerAgentID}, agents)
	})

	t.Run("FindAllAgentsWithoutNomad", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)
		agents, err := models.FindAgents(q, models.AgentFilters{IgnoreNomad: true, PMMAgentID: "A12"})
		require.NoError(t, err)
		require.Len(t, agents, 3)
		assert.Equal(t, "A10", agents[0].AgentID)
		assert.Equal(t, "A8", agents[1].AgentID)
		assert.Equal(t, "A9", agents[2].AgentID)
	})

	t.Run("ChangeAgent", func(t *testing.T) {
		t.Run("ChangeBasicFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing enabled status
			agent, err := models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				Enabled: pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.True(t, agent.Disabled) // Disabled should be true when Enabled is false

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			assert.True(t, persistedAgent.Disabled)

			// Change it back
			agent, err = models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				Enabled: pointer.ToBool(true),
			})
			require.NoError(t, err)
			assert.False(t, agent.Disabled)

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			assert.False(t, persistedAgent.Disabled)
		})

		t.Run("ChangeCustomLabels", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Set custom labels
			customLabels := map[string]string{
				"environment": "test",
				"team":        "qa",
			}
			agent, err := models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				CustomLabels: &customLabels,
			})
			require.NoError(t, err)

			retrievedLabels, err := agent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, retrievedLabels)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			persistedLabels, err := persistedAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, persistedLabels)

			// Clear custom labels
			emptyLabels := map[string]string{}
			agent, err = models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				CustomLabels: &emptyLabels,
			})
			require.NoError(t, err)

			retrievedLabels, err = agent.GetCustomLabels()
			require.NoError(t, err)
			assert.Empty(t, retrievedLabels)

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			persistedLabels, err = persistedAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Empty(t, persistedLabels)
		})

		t.Run("ChangeExporterOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing push metrics
			agent, err := models.ChangeAgent(q, "A5", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					PushMetrics: pointer.ToBool(false),
				},
			})
			require.NoError(t, err)
			assert.False(t, agent.ExporterOptions.PushMetrics)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A5")
			require.NoError(t, err)
			assert.False(t, persistedAgent.ExporterOptions.PushMetrics)

			// Test changing disabled collectors
			disabledCollectors := []string{"collector1", "collector2"}
			agent, err = models.ChangeAgent(q, "A5", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					DisabledCollectors: disabledCollectors,
				},
			})
			require.NoError(t, err)
			assert.Equal(t, disabledCollectors, []string(agent.ExporterOptions.DisabledCollectors))

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A5")
			require.NoError(t, err)
			assert.Equal(t, disabledCollectors, []string(persistedAgent.ExporterOptions.DisabledCollectors))

			// Test changing expose exporter
			agent, err = models.ChangeAgent(q, "A5", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					ExposeExporter: pointer.ToBool(true),
				},
			})
			require.NoError(t, err)
			assert.True(t, agent.ExporterOptions.ExposeExporter)

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A5")
			require.NoError(t, err)
			assert.True(t, persistedAgent.ExporterOptions.ExposeExporter)

			// Test changing metrics scheme and path
			agent, err = models.ChangeAgent(q, "A5", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					MetricsScheme: pointer.ToString("https"),
					MetricsPath:   pointer.ToString("/custom-metrics"),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "https", agent.ExporterOptions.MetricsScheme)
			assert.Equal(t, "/custom-metrics", agent.ExporterOptions.MetricsPath)

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A5")
			require.NoError(t, err)
			assert.Equal(t, "https", persistedAgent.ExporterOptions.MetricsScheme)
			assert.Equal(t, "/custom-metrics", persistedAgent.ExporterOptions.MetricsPath)
		})

		t.Run("ChangeMetricsResolutions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing metrics resolutions
			agent, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					MetricsResolutions: &models.ChangeMetricsResolutionsParams{
						HR: pointer.ToDuration(30 * time.Second),
						MR: pointer.ToDuration(2 * time.Minute),
						LR: pointer.ToDuration(10 * time.Minute),
					},
				},
			})
			require.NoError(t, err)
			assert.Equal(t, 30*time.Second, agent.ExporterOptions.MetricsResolutions.HR)
			assert.Equal(t, 2*time.Minute, agent.ExporterOptions.MetricsResolutions.MR)
			assert.Equal(t, 10*time.Minute, agent.ExporterOptions.MetricsResolutions.LR)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, 30*time.Second, persistedAgent.ExporterOptions.MetricsResolutions.HR)
			assert.Equal(t, 2*time.Minute, persistedAgent.ExporterOptions.MetricsResolutions.MR)
			assert.Equal(t, 10*time.Minute, persistedAgent.ExporterOptions.MetricsResolutions.LR)

			// Test clearing all metrics resolutions (should set to nil)
			agent, err = models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					MetricsResolutions: &models.ChangeMetricsResolutionsParams{
						HR: pointer.ToDuration(0),
						MR: pointer.ToDuration(0),
						LR: pointer.ToDuration(0),
					},
				},
			})
			require.NoError(t, err)
			assert.Nil(t, agent.ExporterOptions.MetricsResolutions)

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Nil(t, persistedAgent.ExporterOptions.MetricsResolutions)
		})

		t.Run("ChangeConnectionFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing username and password
			agent, err := models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				Username:      pointer.ToString("new_user"),
				Password:      pointer.ToString("new_password"),
				AgentPassword: pointer.ToString("agent_pass"),
			})
			require.NoError(t, err)
			assert.Equal(t, "new_user", pointer.GetString(agent.Username))
			assert.Equal(t, "new_password", pointer.GetString(agent.Password))
			assert.Equal(t, "agent_pass", pointer.GetString(agent.AgentPassword))

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			assert.Equal(t, "new_user", pointer.GetString(persistedAgent.Username))
			assert.Equal(t, "new_password", pointer.GetString(persistedAgent.Password))
			assert.Equal(t, "agent_pass", pointer.GetString(persistedAgent.AgentPassword))
		})

		t.Run("ChangePostgreSQLOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing PostgreSQL options
			agent, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				PostgreSQLOptions: &models.ChangePostgreSQLOptions{
					SSLCa:                  pointer.ToString("new_ca"),
					SSLCert:                pointer.ToString("new_cert"),
					SSLKey:                 pointer.ToString("new_key"),
					AutoDiscoveryLimit:     pointer.ToInt32(100),
					MaxExporterConnections: pointer.ToInt32(5),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new_ca", agent.PostgreSQLOptions.SSLCa)
			assert.Equal(t, "new_cert", agent.PostgreSQLOptions.SSLCert)
			assert.Equal(t, "new_key", agent.PostgreSQLOptions.SSLKey)
			assert.Equal(t, int32(100), pointer.GetInt32(agent.PostgreSQLOptions.AutoDiscoveryLimit))
			assert.Equal(t, int32(5), agent.PostgreSQLOptions.MaxExporterConnections)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, "new_ca", persistedAgent.PostgreSQLOptions.SSLCa)
			assert.Equal(t, "new_cert", persistedAgent.PostgreSQLOptions.SSLCert)
			assert.Equal(t, "new_key", persistedAgent.PostgreSQLOptions.SSLKey)
			assert.Equal(t, int32(100), pointer.GetInt32(persistedAgent.PostgreSQLOptions.AutoDiscoveryLimit))
			assert.Equal(t, int32(5), persistedAgent.PostgreSQLOptions.MaxExporterConnections)
		})

		t.Run("ChangeMongoDBOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing MongoDB options
			statsCollections := []string{"stats1", "stats2"}
			agent, err := models.ChangeAgent(q, "A8", &models.ChangeAgentParams{
				MongoDBOptions: &models.ChangeMongoDBOptions{
					TLSCertificateKey:             pointer.ToString("new_cert_key"),
					TLSCertificateKeyFilePassword: pointer.ToString("new_password"),
					TLSCa:                         pointer.ToString("new_ca"),
					AuthenticationMechanism:       pointer.ToString("SCRAM-SHA-256"),
					AuthenticationDatabase:        pointer.ToString("admin"),
					StatsCollections:              statsCollections,
					CollectionsLimit:              pointer.ToInt32(500),
					EnableAllCollectors:           pointer.ToBool(false),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new_cert_key", agent.MongoDBOptions.TLSCertificateKey)
			assert.Equal(t, "new_password", agent.MongoDBOptions.TLSCertificateKeyFilePassword)
			assert.Equal(t, "new_ca", agent.MongoDBOptions.TLSCa)
			assert.Equal(t, "SCRAM-SHA-256", agent.MongoDBOptions.AuthenticationMechanism)
			assert.Equal(t, "admin", agent.MongoDBOptions.AuthenticationDatabase)
			assert.Equal(t, statsCollections, agent.MongoDBOptions.StatsCollections)
			assert.Equal(t, int32(500), agent.MongoDBOptions.CollectionsLimit)
			assert.False(t, agent.MongoDBOptions.EnableAllCollectors)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A8")
			require.NoError(t, err)
			assert.Equal(t, "new_cert_key", persistedAgent.MongoDBOptions.TLSCertificateKey)
			assert.Equal(t, "new_password", persistedAgent.MongoDBOptions.TLSCertificateKeyFilePassword)
			assert.Equal(t, "new_ca", persistedAgent.MongoDBOptions.TLSCa)
			assert.Equal(t, "SCRAM-SHA-256", persistedAgent.MongoDBOptions.AuthenticationMechanism)
			assert.Equal(t, "admin", persistedAgent.MongoDBOptions.AuthenticationDatabase)
			assert.Equal(t, statsCollections, persistedAgent.MongoDBOptions.StatsCollections)
			assert.Equal(t, int32(500), persistedAgent.MongoDBOptions.CollectionsLimit)
			assert.False(t, persistedAgent.MongoDBOptions.EnableAllCollectors)
		})

		t.Run("ChangeQANOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing QAN options
			agent, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				QANOptions: &models.ChangeQANOptions{
					MaxQueryLength:          pointer.ToInt32(2048),
					QueryExamplesDisabled:   pointer.ToBool(true),
					CommentsParsingDisabled: pointer.ToBool(false),
					MaxQueryLogSize:         pointer.ToInt64(1024000),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, int32(2048), agent.QANOptions.MaxQueryLength)
			assert.True(t, agent.QANOptions.QueryExamplesDisabled)
			assert.False(t, agent.QANOptions.CommentsParsingDisabled)
			assert.Equal(t, int64(1024000), agent.QANOptions.MaxQueryLogSize)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, int32(2048), persistedAgent.QANOptions.MaxQueryLength)
			assert.True(t, persistedAgent.QANOptions.QueryExamplesDisabled)
			assert.False(t, persistedAgent.QANOptions.CommentsParsingDisabled)
			assert.Equal(t, int64(1024000), persistedAgent.QANOptions.MaxQueryLogSize)
		})

		t.Run("ChangeAWSOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Create an AWS RDS Exporter for testing
			awsAgent := &models.Agent{
				AgentID:      "AWS1",
				AgentType:    models.RDSExporterType,
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
				AWSOptions: models.AWSOptions{
					AWSAccessKey:               "old-access-key",
					AWSSecretKey:               "old-secret-key",
					RDSBasicMetricsDisabled:    false,
					RDSEnhancedMetricsDisabled: false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			err := q.Insert(awsAgent)
			require.NoError(t, err)

			// Test changing AWS options
			agent, err := models.ChangeAgent(q, "AWS1", &models.ChangeAgentParams{
				AWSOptions: &models.ChangeAWSOptions{
					AWSAccessKey:               pointer.ToString("new-access-key"),
					AWSSecretKey:               pointer.ToString("new-secret-key"),
					RDSBasicMetricsDisabled:    pointer.ToBool(true),
					RDSEnhancedMetricsDisabled: pointer.ToBool(true),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new-access-key", agent.AWSOptions.AWSAccessKey)
			assert.Equal(t, "new-secret-key", agent.AWSOptions.AWSSecretKey)
			assert.True(t, agent.AWSOptions.RDSBasicMetricsDisabled)
			assert.True(t, agent.AWSOptions.RDSEnhancedMetricsDisabled)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "AWS1")
			require.NoError(t, err)
			assert.Equal(t, "new-access-key", persistedAgent.AWSOptions.AWSAccessKey)
			assert.Equal(t, "new-secret-key", persistedAgent.AWSOptions.AWSSecretKey)
			assert.True(t, persistedAgent.AWSOptions.RDSBasicMetricsDisabled)
			assert.True(t, persistedAgent.AWSOptions.RDSEnhancedMetricsDisabled)
		})

		t.Run("ChangeMySQLOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Create a MySQL Exporter for testing
			mysqlAgent := &models.Agent{
				AgentID:      "MYSQL1",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S1"),
				MySQLOptions: models.MySQLOptions{
					TLSCa:                          "old-ca",
					TLSCert:                        "old-cert",
					TLSKey:                         "old-key",
					TableCountTablestatsGroupLimit: 100,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			err := q.Insert(mysqlAgent)
			require.NoError(t, err)

			// Test changing MySQL options
			agent, err := models.ChangeAgent(q, "MYSQL1", &models.ChangeAgentParams{
				MySQLOptions: &models.ChangeMySQLOptions{
					TLSCa:                          pointer.ToString("new-mysql-ca"),
					TLSCert:                        pointer.ToString("new-mysql-cert"),
					TLSKey:                         pointer.ToString("new-mysql-key"),
					TableCountTablestatsGroupLimit: pointer.ToInt32(200),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new-mysql-ca", agent.MySQLOptions.TLSCa)
			assert.Equal(t, "new-mysql-cert", agent.MySQLOptions.TLSCert)
			assert.Equal(t, "new-mysql-key", agent.MySQLOptions.TLSKey)
			assert.Equal(t, int32(200), agent.MySQLOptions.TableCountTablestatsGroupLimit)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "MYSQL1")
			require.NoError(t, err)
			assert.Equal(t, "new-mysql-ca", persistedAgent.MySQLOptions.TLSCa)
			assert.Equal(t, "new-mysql-cert", persistedAgent.MySQLOptions.TLSCert)
			assert.Equal(t, "new-mysql-key", persistedAgent.MySQLOptions.TLSKey)
			assert.Equal(t, int32(200), persistedAgent.MySQLOptions.TableCountTablestatsGroupLimit)
		})

		t.Run("ChangeValkeyOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Create a Valkey Exporter for testing
			valkeyAgent := &models.Agent{
				AgentID:      "VALKEY1",
				AgentType:    models.ExternalExporterType, // Using external exporter type for Valkey
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
				ValkeyOptions: models.ValkeyOptions{
					SSLCa:   "old-valkey-ca",
					SSLCert: "old-valkey-cert",
					SSLKey:  "old-valkey-key",
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			err := q.Insert(valkeyAgent)
			require.NoError(t, err)

			// Test changing Valkey options
			agent, err := models.ChangeAgent(q, "VALKEY1", &models.ChangeAgentParams{
				ValkeyOptions: &models.ChangeValkeyOptions{
					SSLCa:   pointer.ToString("new-valkey-ca"),
					SSLCert: pointer.ToString("new-valkey-cert"),
					SSLKey:  pointer.ToString("new-valkey-key"),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new-valkey-ca", agent.ValkeyOptions.SSLCa)
			assert.Equal(t, "new-valkey-cert", agent.ValkeyOptions.SSLCert)
			assert.Equal(t, "new-valkey-key", agent.ValkeyOptions.SSLKey)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "VALKEY1")
			require.NoError(t, err)
			assert.Equal(t, "new-valkey-ca", persistedAgent.ValkeyOptions.SSLCa)
			assert.Equal(t, "new-valkey-cert", persistedAgent.ValkeyOptions.SSLCert)
			assert.Equal(t, "new-valkey-key", persistedAgent.ValkeyOptions.SSLKey)
		})

		t.Run("ChangeTLSFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing TLS fields
			agent, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				TLS:           pointer.ToBool(false),
				TLSSkipVerify: pointer.ToBool(false),
			})
			require.NoError(t, err)
			assert.False(t, agent.TLS)
			assert.False(t, agent.TLSSkipVerify)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.False(t, persistedAgent.TLS)
			assert.False(t, persistedAgent.TLSSkipVerify)
		})

		t.Run("ChangeLogLevel", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing log level
			logLevel := "debug"
			agent, err := models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				LogLevel: &logLevel,
			})
			require.NoError(t, err)
			assert.Equal(t, "debug", pointer.GetString(agent.LogLevel))

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			assert.Equal(t, "debug", pointer.GetString(persistedAgent.LogLevel))
		})

		t.Run("ChangeListenPort", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing listen port (for external exporter)
			port := uint32(9999)
			agent, err := models.ChangeAgent(q, "A5", &models.ChangeAgentParams{
				ListenPort: &port,
			})
			require.NoError(t, err)
			assert.Equal(t, uint16(9999), pointer.GetUint16(agent.ListenPort))

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A5")
			require.NoError(t, err)
			assert.Equal(t, uint16(9999), pointer.GetUint16(persistedAgent.ListenPort))
		})

		t.Run("InvalidAgentID", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test with non-existent agent ID
			_, err := models.ChangeAgent(q, "INVALID", &models.ChangeAgentParams{
				Enabled: pointer.ToBool(false),
			})
			tests.AssertGRPCError(t, status.New(codes.NotFound, "Agent with ID INVALID not found."), err)
		})

		t.Run("ChangeAzureOptions", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Create an Azure Database Exporter for testing
			azureAgent := &models.Agent{
				AgentID:      "AZURE1",
				AgentType:    models.AzureDatabaseExporterType,
				PMMAgentID:   pointer.ToString("A1"),
				RunsOnNodeID: nil,
				NodeID:       pointer.ToString("N1"),
				AzureOptions: models.AzureOptions{
					SubscriptionID: "old-subscription",
					ClientID:       "old-client-id",
					ClientSecret:   "old-secret",
					TenantID:       "old-tenant",
					ResourceGroup:  "old-group",
				},
				CreatedAt: now,
				UpdatedAt: now,
			}
			err := q.Insert(azureAgent)
			require.NoError(t, err)

			// Test changing Azure options
			agent, err := models.ChangeAgent(q, "AZURE1", &models.ChangeAgentParams{
				AzureOptions: &models.ChangeAzureOptions{
					SubscriptionID: pointer.ToString("new-subscription"),
					ClientID:       pointer.ToString("new-client-id"),
					ClientSecret:   pointer.ToString("new-secret"),
					TenantID:       pointer.ToString("new-tenant"),
					ResourceGroup:  pointer.ToString("new-group"),
				},
			})
			require.NoError(t, err)
			assert.Equal(t, "new-subscription", agent.AzureOptions.SubscriptionID)
			assert.Equal(t, "new-client-id", agent.AzureOptions.ClientID)
			assert.Equal(t, "new-secret", agent.AzureOptions.ClientSecret)
			assert.Equal(t, "new-tenant", agent.AzureOptions.TenantID)
			assert.Equal(t, "new-group", agent.AzureOptions.ResourceGroup)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "AZURE1")
			require.NoError(t, err)
			assert.Equal(t, "new-subscription", persistedAgent.AzureOptions.SubscriptionID)
			assert.Equal(t, "new-client-id", persistedAgent.AzureOptions.ClientID)
			assert.Equal(t, "new-secret", persistedAgent.AzureOptions.ClientSecret)
			assert.Equal(t, "new-tenant", persistedAgent.AzureOptions.TenantID)
			assert.Equal(t, "new-group", persistedAgent.AzureOptions.ResourceGroup)
		})

		t.Run("ChangeMultipleFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing multiple fields at once
			customLabels := map[string]string{"env": "prod"}
			agent, err := models.ChangeAgent(q, "A2", &models.ChangeAgentParams{
				Enabled:      pointer.ToBool(false),
				Username:     pointer.ToString("multi_user"),
				Password:     pointer.ToString("multi_pass"),
				CustomLabels: &customLabels,
				ExporterOptions: &models.ChangeExporterOptions{
					PushMetrics: pointer.ToBool(true),
				},
			})
			require.NoError(t, err)

			assert.True(t, agent.Disabled)
			assert.Equal(t, "multi_user", pointer.GetString(agent.Username))
			assert.Equal(t, "multi_pass", pointer.GetString(agent.Password))
			assert.True(t, agent.ExporterOptions.PushMetrics)

			retrievedLabels, err := agent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, retrievedLabels)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A2")
			require.NoError(t, err)
			assert.True(t, persistedAgent.Disabled)
			assert.Equal(t, "multi_user", pointer.GetString(persistedAgent.Username))
			assert.Equal(t, "multi_pass", pointer.GetString(persistedAgent.Password))
			assert.True(t, persistedAgent.ExporterOptions.PushMetrics)

			persistedLabels, err := persistedAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, persistedLabels)
		})

		t.Run("UnspecifiedFieldsRemainUnchanged", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// First, set up agent A7 (PostgreSQL exporter) with some initial values
			initialCustomLabels := map[string]string{"initial": "value", "env": "test"}
			_, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				Username:     pointer.ToString("initial_user"),
				Password:     pointer.ToString("initial_pass"),
				CustomLabels: &initialCustomLabels,
				LogLevel:     pointer.ToString("info"),
				ExporterOptions: &models.ChangeExporterOptions{
					PushMetrics:    pointer.ToBool(true),
					ExposeExporter: pointer.ToBool(true),
				},
				PostgreSQLOptions: &models.ChangePostgreSQLOptions{
					AutoDiscoveryLimit:     pointer.ToInt32(200),
					MaxExporterConnections: pointer.ToInt32(10),
				},
			})
			require.NoError(t, err)

			// Verify initial state
			initialAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, "initial_user", pointer.GetString(initialAgent.Username))
			assert.Equal(t, "initial_pass", pointer.GetString(initialAgent.Password))
			assert.Equal(t, "info", pointer.GetString(initialAgent.LogLevel))
			assert.True(t, initialAgent.ExporterOptions.PushMetrics)
			assert.True(t, initialAgent.ExporterOptions.ExposeExporter)
			assert.Equal(t, int32(200), pointer.GetInt32(initialAgent.PostgreSQLOptions.AutoDiscoveryLimit))
			assert.Equal(t, int32(10), initialAgent.PostgreSQLOptions.MaxExporterConnections)

			initialLabels, err := initialAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, initialCustomLabels, initialLabels)

			// Now change only the username - all other fields should remain unchanged
			agent, err := models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				Username: pointer.ToString("changed_user"),
			})
			require.NoError(t, err)

			// Verify that only username changed
			assert.Equal(t, "changed_user", pointer.GetString(agent.Username))
			// All other fields should remain unchanged
			assert.Equal(t, "initial_pass", pointer.GetString(agent.Password))
			assert.Equal(t, "info", pointer.GetString(agent.LogLevel))
			assert.True(t, agent.ExporterOptions.PushMetrics)
			assert.True(t, agent.ExporterOptions.ExposeExporter)
			assert.Equal(t, int32(200), pointer.GetInt32(agent.PostgreSQLOptions.AutoDiscoveryLimit))
			assert.Equal(t, int32(10), agent.PostgreSQLOptions.MaxExporterConnections)

			// Verify persistence in database
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, "changed_user", pointer.GetString(persistedAgent.Username))
			assert.Equal(t, "initial_pass", pointer.GetString(persistedAgent.Password))
			assert.Equal(t, "info", pointer.GetString(persistedAgent.LogLevel))
			assert.True(t, persistedAgent.ExporterOptions.PushMetrics)
			assert.True(t, persistedAgent.ExporterOptions.ExposeExporter)
			assert.Equal(t, int32(200), pointer.GetInt32(persistedAgent.PostgreSQLOptions.AutoDiscoveryLimit))
			assert.Equal(t, int32(10), persistedAgent.PostgreSQLOptions.MaxExporterConnections)

			// Custom labels should also remain unchanged
			persistedLabels, err := persistedAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, initialCustomLabels, persistedLabels)

			// Test changing only exporter options - other fields should remain unchanged
			agent, err = models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				ExporterOptions: &models.ChangeExporterOptions{
					PushMetrics: pointer.ToBool(false), // Change this
					// Don't specify ExposeExporter - it should remain true
				},
			})
			require.NoError(t, err)

			// Verify that only PushMetrics changed, ExposeExporter remains true
			assert.False(t, agent.ExporterOptions.PushMetrics)   // Changed
			assert.True(t, agent.ExporterOptions.ExposeExporter) // Unchanged
			// Other fields should still be unchanged
			assert.Equal(t, "changed_user", pointer.GetString(agent.Username))
			assert.Equal(t, "initial_pass", pointer.GetString(agent.Password))
			assert.Equal(t, "info", pointer.GetString(agent.LogLevel))

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.False(t, persistedAgent.ExporterOptions.PushMetrics)   // Changed
			assert.True(t, persistedAgent.ExporterOptions.ExposeExporter) // Unchanged
			assert.Equal(t, "changed_user", pointer.GetString(persistedAgent.Username))
			assert.Equal(t, "initial_pass", pointer.GetString(persistedAgent.Password))
			assert.Equal(t, "info", pointer.GetString(persistedAgent.LogLevel))

			// Test changing only PostgreSQL options - other fields should remain unchanged
			agent, err = models.ChangeAgent(q, "A7", &models.ChangeAgentParams{
				PostgreSQLOptions: &models.ChangePostgreSQLOptions{
					AutoDiscoveryLimit: pointer.ToInt32(500), // Change this
					// Don't specify MaxExporterConnections - it should remain 10
				},
			})
			require.NoError(t, err)

			// Verify that only AutoDiscoveryLimit changed
			assert.Equal(t, int32(500), pointer.GetInt32(agent.PostgreSQLOptions.AutoDiscoveryLimit)) // Changed
			assert.Equal(t, int32(10), agent.PostgreSQLOptions.MaxExporterConnections)                // Unchanged

			// Verify persistence in database
			persistedAgent, err = models.FindAgentByID(q, "A7")
			require.NoError(t, err)
			assert.Equal(t, int32(500), pointer.GetInt32(persistedAgent.PostgreSQLOptions.AutoDiscoveryLimit)) // Changed
			assert.Equal(t, int32(10), persistedAgent.PostgreSQLOptions.MaxExporterConnections)                // Unchanged
		})

		t.Run("ChangeAllFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Test changing ALL possible fields at once for a PostgreSQL exporter (A7)
			// This is the most comprehensive test to ensure the function handles complex scenarios

			customLabels := map[string]string{
				"environment": "production",
				"team":        "platform",
				"region":      "us-west-2",
				"service":     "core-db",
			}

			disabledCollectors := []string{"stat_database", "stat_bgwriter", "stat_wal"}
			mongoCollections := []string{"users", "orders", "products"}

			changeParams := &models.ChangeAgentParams{
				// Basic fields
				Enabled:       pointer.ToBool(false),
				Username:      pointer.ToString("comprehensive_user"),
				Password:      pointer.ToString("comprehensive_password"),
				AgentPassword: pointer.ToString("comprehensive_agent_password"),
				LogLevel:      pointer.ToString("debug"),
				TLS:           pointer.ToBool(false),
				TLSSkipVerify: pointer.ToBool(false),
				ListenPort:    pointer.ToUint32(9090),

				// Custom labels
				CustomLabels: &customLabels,

				// Exporter options
				ExporterOptions: &models.ChangeExporterOptions{
					PushMetrics:        pointer.ToBool(true),
					DisabledCollectors: disabledCollectors,
					ExposeExporter:     pointer.ToBool(true),
					MetricsScheme:      pointer.ToString("https"),
					MetricsPath:        pointer.ToString("/custom-metrics"),
					MetricsResolutions: &models.ChangeMetricsResolutionsParams{
						HR: pointer.ToDuration(15 * time.Second),
						MR: pointer.ToDuration(90 * time.Second),
						LR: pointer.ToDuration(8 * time.Minute),
					},
				},

				// QAN options
				QANOptions: &models.ChangeQANOptions{
					MaxQueryLength:          pointer.ToInt32(4096),
					QueryExamplesDisabled:   pointer.ToBool(false),
					CommentsParsingDisabled: pointer.ToBool(true),
					MaxQueryLogSize:         pointer.ToInt64(2048000),
				},

				// AWS options
				AWSOptions: &models.ChangeAWSOptions{
					AWSAccessKey:               pointer.ToString("comprehensive-aws-key"),
					AWSSecretKey:               pointer.ToString("comprehensive-aws-secret"),
					RDSBasicMetricsDisabled:    pointer.ToBool(true),
					RDSEnhancedMetricsDisabled: pointer.ToBool(false),
				},

				// Azure options
				AzureOptions: &models.ChangeAzureOptions{
					SubscriptionID: pointer.ToString("comprehensive-subscription"),
					ClientID:       pointer.ToString("comprehensive-client-id"),
					ClientSecret:   pointer.ToString("comprehensive-client-secret"),
					TenantID:       pointer.ToString("comprehensive-tenant"),
					ResourceGroup:  pointer.ToString("comprehensive-resource-group"),
				},

				// MongoDB options
				MongoDBOptions: &models.ChangeMongoDBOptions{
					TLSCertificateKey:             pointer.ToString("comprehensive-mongo-cert-key"),
					TLSCertificateKeyFilePassword: pointer.ToString("comprehensive-mongo-password"),
					TLSCa:                         pointer.ToString("comprehensive-mongo-ca"),
					AuthenticationMechanism:       pointer.ToString("SCRAM-SHA-256"),
					AuthenticationDatabase:        pointer.ToString("admin"),
					StatsCollections:              mongoCollections,
					CollectionsLimit:              pointer.ToInt32(1000),
					EnableAllCollectors:           pointer.ToBool(true),
				},

				// MySQL options
				MySQLOptions: &models.ChangeMySQLOptions{
					TLSCa:                          pointer.ToString("comprehensive-mysql-ca"),
					TLSCert:                        pointer.ToString("comprehensive-mysql-cert"),
					TLSKey:                         pointer.ToString("comprehensive-mysql-key"),
					TableCountTablestatsGroupLimit: pointer.ToInt32(500),
				},

				// PostgreSQL-specific options
				PostgreSQLOptions: &models.ChangePostgreSQLOptions{
					SSLCa:                  pointer.ToString("comprehensive-ca-cert"),
					SSLCert:                pointer.ToString("comprehensive-ssl-cert"),
					SSLKey:                 pointer.ToString("comprehensive-ssl-key"),
					AutoDiscoveryLimit:     pointer.ToInt32(150),
					MaxExporterConnections: pointer.ToInt32(8),
				},

				// Valkey options
				ValkeyOptions: &models.ChangeValkeyOptions{
					SSLCa:   pointer.ToString("comprehensive-valkey-ca"),
					SSLCert: pointer.ToString("comprehensive-valkey-cert"),
					SSLKey:  pointer.ToString("comprehensive-valkey-key"),
				},
			}

			agent, err := models.ChangeAgent(q, "A7", changeParams)
			require.NoError(t, err)

			// Build expected agent structure for comparison
			expectedAgent := &models.Agent{
				// Copy basic fields from actual agent (timestamps, IDs, etc. should remain the same)
				AgentID:    agent.AgentID,
				AgentType:  agent.AgentType,
				PMMAgentID: agent.PMMAgentID,
				NodeID:     agent.NodeID,
				ServiceID:  agent.ServiceID,
				CreatedAt:  agent.CreatedAt,
				UpdatedAt:  agent.UpdatedAt,
				Status:     agent.Status,

				// Fields that should be changed
				Disabled:      true, // Enabled=false means Disabled=true
				Username:      pointer.ToString("comprehensive_user"),
				Password:      pointer.ToString("comprehensive_password"),
				AgentPassword: pointer.ToString("comprehensive_agent_password"),
				LogLevel:      pointer.ToString("debug"),
				TLS:           false,
				TLSSkipVerify: false,
				ListenPort:    pointer.ToUint16(9090),

				ExporterOptions: models.ExporterOptions{
					PushMetrics:        true,
					DisabledCollectors: disabledCollectors,
					ExposeExporter:     true,
					MetricsScheme:      "https",
					MetricsPath:        "/custom-metrics",
					MetricsResolutions: &models.MetricsResolutions{
						HR: 15 * time.Second,
						MR: 90 * time.Second,
						LR: 8 * time.Minute,
					},
				},

				QANOptions: models.QANOptions{
					MaxQueryLength:          4096,
					QueryExamplesDisabled:   false,
					CommentsParsingDisabled: true,
					MaxQueryLogSize:         2048000,
				},

				AWSOptions: models.AWSOptions{
					AWSAccessKey:               "comprehensive-aws-key",
					AWSSecretKey:               "comprehensive-aws-secret",
					RDSBasicMetricsDisabled:    true,
					RDSEnhancedMetricsDisabled: false,
				},

				AzureOptions: models.AzureOptions{
					SubscriptionID: "comprehensive-subscription",
					ClientID:       "comprehensive-client-id",
					ClientSecret:   "comprehensive-client-secret",
					TenantID:       "comprehensive-tenant",
					ResourceGroup:  "comprehensive-resource-group",
				},

				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "comprehensive-mongo-cert-key",
					TLSCertificateKeyFilePassword: "comprehensive-mongo-password",
					TLSCa:                         "comprehensive-mongo-ca",
					AuthenticationMechanism:       "SCRAM-SHA-256",
					AuthenticationDatabase:        "admin",
					StatsCollections:              mongoCollections,
					CollectionsLimit:              1000,
					EnableAllCollectors:           true,
				},

				MySQLOptions: models.MySQLOptions{
					TLSCa:                          "comprehensive-mysql-ca",
					TLSCert:                        "comprehensive-mysql-cert",
					TLSKey:                         "comprehensive-mysql-key",
					TableCountTablestatsGroupLimit: 500,
				},

				PostgreSQLOptions: models.PostgreSQLOptions{
					SSLCa:                  "comprehensive-ca-cert",
					SSLCert:                "comprehensive-ssl-cert",
					SSLKey:                 "comprehensive-ssl-key",
					AutoDiscoveryLimit:     pointer.ToInt32(150),
					MaxExporterConnections: 8,
				},

				ValkeyOptions: models.ValkeyOptions{
					SSLCa:   "comprehensive-valkey-ca",
					SSLCert: "comprehensive-valkey-cert",
					SSLKey:  "comprehensive-valkey-key",
				},
			}

			// Set custom labels on expected agent
			err = expectedAgent.SetCustomLabels(customLabels)
			require.NoError(t, err)

			// Compare the structures
			assert.Equal(t, expectedAgent.Disabled, agent.Disabled)
			assert.Equal(t, expectedAgent.Username, agent.Username)
			assert.Equal(t, expectedAgent.Password, agent.Password)
			assert.Equal(t, expectedAgent.AgentPassword, agent.AgentPassword)
			assert.Equal(t, expectedAgent.LogLevel, agent.LogLevel)
			assert.Equal(t, expectedAgent.TLS, agent.TLS)
			assert.Equal(t, expectedAgent.TLSSkipVerify, agent.TLSSkipVerify)
			assert.Equal(t, expectedAgent.ListenPort, agent.ListenPort)

			// Compare options structures
			assert.Equal(t, expectedAgent.ExporterOptions, agent.ExporterOptions)
			assert.Equal(t, expectedAgent.QANOptions, agent.QANOptions)
			assert.Equal(t, expectedAgent.AWSOptions, agent.AWSOptions)
			assert.Equal(t, expectedAgent.AzureOptions, agent.AzureOptions)
			assert.Equal(t, expectedAgent.MongoDBOptions, agent.MongoDBOptions)
			assert.Equal(t, expectedAgent.MySQLOptions, agent.MySQLOptions)
			assert.Equal(t, expectedAgent.PostgreSQLOptions, agent.PostgreSQLOptions)
			assert.Equal(t, expectedAgent.ValkeyOptions, agent.ValkeyOptions)

			// Verify custom labels
			actualLabels, err := agent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, actualLabels)

			// Verify persistence in database by comparing with fresh fetch
			persistedAgent, err := models.FindAgentByID(q, "A7")
			require.NoError(t, err)

			// Compare all the changed fields with persisted agent
			assert.Equal(t, expectedAgent.Disabled, persistedAgent.Disabled)
			assert.Equal(t, expectedAgent.Username, persistedAgent.Username)
			assert.Equal(t, expectedAgent.Password, persistedAgent.Password)
			assert.Equal(t, expectedAgent.AgentPassword, persistedAgent.AgentPassword)
			assert.Equal(t, expectedAgent.LogLevel, persistedAgent.LogLevel)
			assert.Equal(t, expectedAgent.TLS, persistedAgent.TLS)
			assert.Equal(t, expectedAgent.TLSSkipVerify, persistedAgent.TLSSkipVerify)
			assert.Equal(t, expectedAgent.ListenPort, persistedAgent.ListenPort)

			// Compare persisted structures
			assert.Equal(t, expectedAgent.ExporterOptions, persistedAgent.ExporterOptions)
			assert.Equal(t, expectedAgent.QANOptions, persistedAgent.QANOptions)
			assert.Equal(t, expectedAgent.AWSOptions, persistedAgent.AWSOptions)
			assert.Equal(t, expectedAgent.AzureOptions, persistedAgent.AzureOptions)
			assert.Equal(t, expectedAgent.MongoDBOptions, persistedAgent.MongoDBOptions)
			assert.Equal(t, expectedAgent.MySQLOptions, persistedAgent.MySQLOptions)
			assert.Equal(t, expectedAgent.PostgreSQLOptions, persistedAgent.PostgreSQLOptions)
			assert.Equal(t, expectedAgent.ValkeyOptions, persistedAgent.ValkeyOptions)

			// Verify persisted custom labels
			persistedLabels, err := persistedAgent.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, customLabels, persistedLabels)
		})
	})
}
