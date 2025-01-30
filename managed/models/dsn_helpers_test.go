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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestFindDSNByServiceID(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
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
			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service on N1",
				NodeID:      "N1",
				Socket:      pointer.ToStringOrNil("/var/run/mysqld/mysqld.sock"),
			},
			&models.Service{
				ServiceID:   "S2",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service-2 on N1",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(3306),
			},
			&models.Service{
				ServiceID:   "S3",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service-3 on N1",
				NodeID:      "N1",
				Socket:      pointer.ToStringOrNil("/var/run/mysqld/mysqld.sock"),
			},
			&models.Service{
				ServiceID:   "S4",
				ServiceType: models.MongoDBServiceType,
				ServiceName: "Service-4 on N1",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(27017),
			},
			&models.Agent{
				AgentID:      "PA1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
			},
			&models.Agent{
				AgentID:      "PA2",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
				Version:      pointer.ToString("2.12.0"),
			},
			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S1"),
			},
			&models.Agent{
				AgentID:      "A2",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				Username:     pointer.ToString("pmm-user"),
				ServiceID:    pointer.ToString("S2"),
			},
			&models.Agent{
				AgentID:      "A3",
				AgentType:    models.QANMySQLSlowlogAgentType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				Username:     pointer.ToString("pmm-user"),
				ServiceID:    pointer.ToString("S2"),
			},
			&models.Agent{
				AgentID:      "A4",
				AgentType:    models.QANMySQLPerfSchemaAgentType,
				PMMAgentID:   pointer.ToString("PA1"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S2"),
			},
			&models.Agent{
				AgentID:      "A6",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA2"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S3"),
			},
			&models.Agent{
				AgentID:      "A7",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToString("PA2"),
				RunsOnNodeID: nil,
				ServiceID:    pointer.ToString("S3"),
				TLS:          true,
				MySQLOptions: models.MySQLOptions{
					TLSCa:   "content-of-tls-ca",
					TLSCert: "content-of-tls-cert",
					TLSKey:  "content-of-tls-key",
				},
			},
			&models.Agent{
				AgentID:      "A8",
				AgentType:    models.MongoDBExporterType,
				PMMAgentID:   pointer.ToString("PA2"),
				RunsOnNodeID: nil,
				Username:     pointer.ToString("pmm-user{{"),
				ServiceID:    pointer.ToString("S4"),
				TLS:          true,
				MongoDBOptions: models.MongoDBOptions{
					TLSCertificateKey:             "content-of-tls-certificate-key",
					TLSCertificateKeyFilePassword: "passwordoftls",
					TLSCa:                         "content-of-tls-ca",
				},
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

	t.Run("FindDSNByServiceIDandPMMAgentIDWithNoAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, _, err := models.FindDSNByServiceIDandPMMAgentID(q, "S3", "PA1", "test")
		require.Error(t, err)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Couldn't resolve dsn, as there should be one agent"), err)
	})

	t.Run("FindDSNByServiceIDandPMMAgentID", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		dsn, agent, err := models.FindDSNByServiceIDandPMMAgentID(q, "S2", "PA1", "test")
		require.NoError(t, err)
		expected := "pmm-user@tcp(127.0.0.1:3306)/test?clientFoundRows=true&parseTime=true&timeout=1s"
		assert.Equal(t, expected, dsn)
		assert.NotNil(t, agent)
	})

	t.Run("FindDSNWithSocketByServiceIDandPMMAgentID", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		dsn, agent, err := models.FindDSNByServiceIDandPMMAgentID(q, "S1", "PA1", "test")
		require.NoError(t, err)
		expected := "unix(/var/run/mysqld/mysqld.sock)/test?timeout=1s"
		assert.Equal(t, expected, dsn)
		assert.NotNil(t, agent)
	})

	t.Run("FindDSNWithFilesByServiceIDandPMMAgentID", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		dsn, agent, err := models.FindDSNByServiceIDandPMMAgentID(q, "S4", "PA2", "test")
		require.NoError(t, err)
		expected := "mongodb://pmm-user%7B%7B@127.0.0.1:27017/test?connectTimeoutMS=1000" +
			"&directConnection=true" +
			"&serverSelectionTimeoutMS=1000&ssl=true" +
			"&tlsCaFile=[[.TextFiles.caFilePlaceholder]]" +
			"&tlsCertificateKeyFile=[[.TextFiles.certificateKeyFilePlaceholder]]" +
			"&tlsCertificateKeyFilePassword=passwordoftls"
		assert.Equal(t, expected, dsn)
		expectedFiles := map[string]string{
			"caFilePlaceholder":             "content-of-tls-ca",
			"certificateKeyFilePlaceholder": "content-of-tls-certificate-key",
		}
		assert.Equal(t, expectedFiles, agent.Files())
	})

	t.Run("FindDSNByServiceIDandPMMAgentIDWithTwoAgentsOfSameType", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, _, err := models.FindDSNByServiceIDandPMMAgentID(q, "S3", "PA2", "test")
		require.Error(t, err)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Couldn't resolve dsn, as there should be only one agent"), err)
	})
}
