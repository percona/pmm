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

package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/suite"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

type PostgresExporterConfigTestSuite struct {
	suite.Suite

	pmmAgentVersion *version.Parsed
	postgresql      *models.Service
	exporter        *models.Agent
	expected        *agentpb.SetStateRequest_AgentProcess
}

func (s *PostgresExporterConfigTestSuite) SetupTest() {
	s.pmmAgentVersion = version.MustParse("2.15.1")
	s.postgresql = &models.Service{
		Address:      pointer.ToString("1.2.3.4"),
		Port:         pointer.ToUint16(5432),
		DatabaseName: "postgres",
	}
	s.exporter = &models.Agent{
		AgentID:       "agent-id",
		AgentType:     models.PostgresExporterType,
		Username:      pointer.ToString("username"),
		Password:      pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword: pointer.ToString("agent-password"),
	}
	s.expected = &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
}

func (s *PostgresExporterConfigTestSuite) TestConfig() {
	actual, err := postgresExporterConfig(s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(s.expected.Args, actual.Args)
	s.Require().Equal(s.expected.Env, actual.Env)
	s.Require().Equal(s.expected, actual)
}

func (s *PostgresExporterConfigTestSuite) TestDatabaseName() {
	s.Run("Set", func() {
		s.postgresql.DatabaseName = "db1"
		s.expected.Env[0] = "DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/db1?connect_timeout=1&sslmode=disable"

		actual, err := postgresExporterConfig(s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
		s.NoError(err, "Failed to create exporter config")

		s.Require().Equal(s.expected.Env, actual.Env)
	})

	s.Run("NotSet", func() {
		s.postgresql.DatabaseName = ""

		s.Require().PanicsWithValue("database name not set", func() {
			_, err := postgresExporterConfig(s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
			s.NoError(err, "Failed to create exporter config")
		})
	})
}

func (s *PostgresExporterConfigTestSuite) TestEmptyPassword() {
	s.exporter.Password = nil

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Assert().Equal("DATA_SOURCE_NAME=postgres://username@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestEmptyUsername() {
	s.exporter.Username = nil

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Assert().Equal("DATA_SOURCE_NAME=postgres://:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestEmptyUsernameAndPassword() {
	s.exporter.Username = nil
	s.exporter.Password = nil

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Assert().Equal("DATA_SOURCE_NAME=postgres://1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestSocket() {
	s.exporter.Username = nil
	s.exporter.Password = nil
	s.postgresql.Address = nil
	s.postgresql.Port = nil
	s.postgresql.Socket = pointer.ToString("/var/run/postgres")

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Assert().Equal("DATA_SOURCE_NAME=postgres:///postgres?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestDisabledCollectors() {
	s.pmmAgentVersion = &version.Parsed{}
	s.postgresql.Address = nil
	s.postgresql.Port = nil
	s.postgresql.Socket = pointer.ToString("/var/run/postgres")
	s.exporter.DisabledCollectors = []string{"custom_query.hr", "custom_query.hr.directory"}

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--web.listen-address=:{{ .listen_port }}",
		},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(expected.Args, actual.Args)
}

func (s *PostgresExporterConfigTestSuite) TestAutoDiscovery() {
	s.pmmAgentVersion = version.MustParse("2.16.0")

	s.postgresql = &models.Service{
		Address:      pointer.ToString("1.2.3.4"),
		Port:         pointer.ToUint16(5432),
		DatabaseName: "postgres",
	}
	s.exporter = &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.PostgresExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.expected = &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--auto-discover-databases",
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--exclude-databases=template0,template1,postgres,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(s.expected.Args, actual.Args)
	s.Require().Equal(s.expected.Env, actual.Env)
	s.Require().Equal(s.expected, actual)
}

func (s *PostgresExporterConfigTestSuite) TestAzureTimeout() {
	s.pmmAgentVersion = version.MustParse("2.16.0")

	s.postgresql = &models.Service{
		Address:      pointer.ToString("1.2.3.4"),
		Port:         pointer.ToUint16(5432),
		DatabaseName: "postgres",
	}
	s.exporter = &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.PostgresExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
		AzureOptions: &models.AzureOptions{
			SubscriptionID: "subscription_id",
			ClientID:       "client_id",
			ClientSecret:   "client_secret",
			TenantID:       "tenant_id",
			ResourceGroup:  "resource_group",
		},
	}

	actual, err := postgresExporterConfig(s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.expected = &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--auto-discover-databases",
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--exclude-databases=template0,template1,postgres,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=5&sslmode=disable",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "client_secret"},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(s.expected.Args, actual.Args)
	s.Require().Equal(s.expected.Env, actual.Env)
	s.Require().Equal(s.expected, actual)
}

func TestPostgresExporterConfigTestSuite(t *testing.T) {
	suite.Run(t, &PostgresExporterConfigTestSuite{})
}
