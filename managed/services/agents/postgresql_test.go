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
	"fmt"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

type PostgresExporterConfigTestSuite struct {
	suite.Suite

	pmmAgentVersion *version.Parsed
	postgresql      *models.Service
	exporter        *models.Agent
	expected        *agentv1.SetStateRequest_AgentProcess
	node            *models.Node
}

func (s *PostgresExporterConfigTestSuite) SetupTest() {
	s.pmmAgentVersion = version.MustParse("2.15.1")
	s.node = &models.Node{
		Address: "1.2.3.4",
	}
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
	s.expected = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(s.pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
}

func (s *PostgresExporterConfigTestSuite) TestConfig() {
	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
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

		actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
		s.NoError(err, "Failed to create exporter config")

		s.Require().Equal(s.expected.Env, actual.Env)
	})

	s.Run("NotSet", func() {
		s.postgresql.DatabaseName = ""

		s.Require().PanicsWithValue("database name not set", func() {
			_, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
			s.NoError(err, "Failed to create exporter config")
		})
	})
}

func (s *PostgresExporterConfigTestSuite) TestEmptyPassword() {
	s.exporter.Password = nil

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Equal("DATA_SOURCE_NAME=postgres://username@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestEmptyUsername() {
	s.exporter.Username = nil

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Equal("DATA_SOURCE_NAME=postgres://:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestEmptyUsernameAndPassword() {
	s.exporter.Username = nil
	s.exporter.Password = nil

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Equal("DATA_SOURCE_NAME=postgres://1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestSocket() {
	s.exporter.Username = nil
	s.exporter.Password = nil
	s.postgresql.Address = nil
	s.postgresql.Port = nil
	s.postgresql.Socket = pointer.ToString("/var/run/postgres")

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.Equal("DATA_SOURCE_NAME=postgres:///postgres?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable", actual.Env[0])
}

func (s *PostgresExporterConfigTestSuite) TestDisabledCollectors() {
	s.pmmAgentVersion = version.MustParse("2.42.0")
	s.postgresql.Address = nil
	s.postgresql.Port = nil
	s.postgresql.Socket = pointer.ToString("/var/run/postgres")
	s.exporter.ExporterOptions.DisabledCollectors = []string{"custom_query.hr", "custom_query.hr.directory", "locks"}

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, exposeSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--auto-discover-databases",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory={{ .paths_base }}/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory={{ .paths_base }}/collectors/custom-queries/postgresql/medium-resolution",
			"--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--no-collector.locks",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			"--web.config={{ .TextFiles.webConfigPlaceholder }}",
		},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(expected.Args, actual.Args)
}

func TestAutoDiscovery(t *testing.T) {
	const discoveryFlag = "--auto-discover-databases"
	const excludedFlag = "--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin"

	pmmAgentVersion := version.MustParse("2.12.0")
	node := &models.Node{
		Address: "1.2.3.4",
	}

	postgresql := &models.Service{
		Address:      pointer.ToString("1.2.3.4"),
		Port:         pointer.ToUint16(5432),
		DatabaseName: "postgres",
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.PostgresExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}

	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}

	t.Run("Not supported version - disabled", func(t *testing.T) {
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		assert.NotContains(t, res.Args, discoveryFlag)
		assert.NotContains(t, res.Args, excludedFlag)
	})

	t.Run("Supported version - enabled", func(t *testing.T) {
		pmmAgentVersion = version.MustParse("2.16.0")
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, discoveryFlag)
		assert.Contains(t, res.Args, excludedFlag)
	})

	t.Run("Database count more than limit - disabled", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			AutoDiscoveryLimit: 5,
			DatabaseCount:      10,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.NotContains(t, res.Args, discoveryFlag)
		assert.NotContains(t, res.Args, excludedFlag)
	})

	t.Run("Database count equal to limit - enabled", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			AutoDiscoveryLimit: 5,
			DatabaseCount:      5,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, discoveryFlag)
		assert.Contains(t, res.Args, excludedFlag)
	})

	t.Run("Database count less than limit - enabled", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			AutoDiscoveryLimit: 5,
			DatabaseCount:      3,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, discoveryFlag)
		assert.Contains(t, res.Args, excludedFlag)
	})

	t.Run("Negative limit - disabled", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			AutoDiscoveryLimit: -1,
			DatabaseCount:      3,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.NotContains(t, res.Args, discoveryFlag)
		assert.NotContains(t, res.Args, excludedFlag)
	})

	t.Run("Default - enabled", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			AutoDiscoveryLimit: 0,
			DatabaseCount:      3,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, discoveryFlag)
		assert.Contains(t, res.Args, excludedFlag)
	})
}

func TestMaxConnections(t *testing.T) {
	const maxConnectionsFlag = "--max-connections"

	pmmAgentVersion := version.MustParse("2.42.0")
	node := &models.Node{
		Address: "1.2.3.4",
	}

	postgresql := &models.Service{
		Address:      pointer.ToString("1.2.3.4"),
		Port:         pointer.ToUint16(5432),
		DatabaseName: "postgres",
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.PostgresExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
		PostgreSQLOptions: &models.PostgreSQLOptions{
			MaxExporterConnections: 10,
		},
	}
	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--auto-discover-databases",
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, "{{", "}}") + "/collectors/custom-queries/postgresql/medium-resolution",
			"--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			"--web.config={{ .TextFiles.webConfigPlaceholder }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
		},
		TextFiles: map[string]string{
			"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
		}, RedactWords: []string{"s3cur3 p@$$w0r4."},
	}

	t.Run("Not supported version - disabled", func(t *testing.T) {
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, version.MustParse("2.41.0"))
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		assert.NotContains(t, res.Args, maxConnectionsFlag)
	})

	t.Run("Supported version - enabled", func(t *testing.T) {
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, fmt.Sprintf("%s=%d", maxConnectionsFlag, 10))
	})

	t.Run("Max exporter connections set to 0 - ignore", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			MaxExporterConnections: 0,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.NotContains(t, res.Args, maxConnectionsFlag)
	})

	t.Run("Max exporter connections set to 5 - apply", func(t *testing.T) {
		exporter.PostgreSQLOptions = &models.PostgreSQLOptions{
			MaxExporterConnections: 5,
		}
		res, err := postgresExporterConfig(node, postgresql, exporter, redactSecrets, pmmAgentVersion)
		assert.NoError(t, err)
		assert.Contains(t, res.Args, fmt.Sprintf("%s=%d", maxConnectionsFlag, 5))
	})
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

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.expected = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
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
			"--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
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

func (s *PostgresExporterConfigTestSuite) TestPrometheusWebConfig() {
	s.pmmAgentVersion = version.MustParse("2.31.0")

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
		TLS:       true,
	}

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.expected = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
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
			"--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			"--web.config={{ .TextFiles.webConfigPlaceholder }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=verify-ca",
		},
		TextFiles: map[string]string{
			"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(s.expected.Args, actual.Args)
	s.Require().Equal(s.expected.Env, actual.Env)
	s.Require().Equal(s.expected, actual)
}

func (s *PostgresExporterConfigTestSuite) TestSSLSni() {
	s.pmmAgentVersion = version.MustParse("2.41.0")

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
		TLS:       true,
	}

	actual, err := postgresExporterConfig(s.node, s.postgresql, s.exporter, redactSecrets, s.pmmAgentVersion)
	s.NoError(err, "Failed to create exporter config")

	s.expected = &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
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
			"--exclude-databases=template0,template1,cloudsqladmin,pmm-managed-dev,azure_maintenance,rdsadmin",
			"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			"--web.config={{ .TextFiles.webConfigPlaceholder }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=verify-ca&sslsni=0",
		},
		TextFiles: map[string]string{
			"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(s.T(), actual.Args)
	s.Require().Equal(s.expected.Args, actual.Args)
	s.Require().Equal(s.expected.Env, actual.Env)
	s.Require().Equal(s.expected, actual)
}

func TestPostgresExporterConfigTestSuite(t *testing.T) {
	suite.Run(t, &PostgresExporterConfigTestSuite{})
}
