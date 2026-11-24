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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/version"
)

var (
	postgresExporterAutodiscoveryVersion = version.MustParse("2.15.99")
	postgresExporterWebConfigVersion     = version.MustParse("2.30.99")
	postgresSSLSniVersion                = version.MustParse("2.41.0-0")
	postgresMaxExporterConnsVersion      = version.MustParse("2.41.2-0")
)

// pmmEnabledPostgresExporterCollectors are the collectors that PMM runs although postgres_exporter doesn't by
// default. Disabling one of them turns it off.
var pmmEnabledPostgresExporterCollectors = []string{
	// pg_wal_segments and pg_wal_size_bytes; pg_ls_waldir needs a superuser or a member of pg_monitor.
	"wal",
}

const defaultAutoDiscoveryDatabaseLimit = 50

func pmmServerPostgresPasswordFile() string {
	if !secrets.Exists(secrets.PostgresPassword) {
		return ""
	}
	return secrets.Path(secrets.PostgresPassword)
}

func postgresExcludedDatabases() []string {
	return []string{"template0", "template1", "cloudsqladmin", "pmm-managed-dev", "azure_maintenance", "rdsadmin"}
}

// postgresExporterConfig returns desired configuration of postgres_exporter process.
// The node argument is the pmm-agent's Node, serviceNode is the monitored Service's Node.
// The passwordFile argument is the file with the password of PMM Server's PostgreSQL user, or empty when there is none.
func postgresExporterConfig(node, serviceNode *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed, passwordFile string,
) (*agentv1.SetStateRequest_AgentProcess, error) {
	if service.DatabaseName == "" {
		panic("database name not set")
	}

	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		// LR
		"--collect.custom_query.lr",

		// MR
		"--collect.custom_query.mr",

		// HR
		"--collect.custom_query.hr",

		"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/low-resolution",
		"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/medium-resolution",
		"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/high-resolution",
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
	}

	autoDiscovery := false
	if !pmmAgentVersion.Less(postgresExporterAutodiscoveryVersion) {
		switch {
		case exporter.PostgreSQLOptions.AutoDiscoveryLimit == nil:
			autoDiscovery = true
		case pointer.GetInt32(exporter.PostgreSQLOptions.AutoDiscoveryLimit) == 0: // server defined
			autoDiscovery = exporter.PostgreSQLOptions.DatabaseCount <= defaultAutoDiscoveryDatabaseLimit
		case pointer.GetInt32(exporter.PostgreSQLOptions.AutoDiscoveryLimit) < 0: // always disabled
		default:
			autoDiscovery = exporter.PostgreSQLOptions.DatabaseCount <= pointer.GetInt32(exporter.PostgreSQLOptions.AutoDiscoveryLimit)
		}
	}
	if autoDiscovery {
		args = append(args,
			"--auto-discover-databases",
			"--exclude-databases="+strings.Join(postgresExcludedDatabases(), ","))
	}

	if !pmmAgentVersion.Less(postgresMaxExporterConnsVersion) &&
		exporter.PostgreSQLOptions.MaxExporterConnections != 0 {
		args = append(args, "--max-connections="+strconv.Itoa(int(exporter.PostgreSQLOptions.MaxExporterConnections)))
	}

	if exporter.ExporterOptions.MetricsPath != "" {
		args = append(args, "--web.telemetry-path="+exporter.ExporterOptions.MetricsPath)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.ExporterOptions.DisabledCollectors)
	args = append(args, collectors.PostgresExporterCollectorArgs(
		slices.Concat(pmmEnabledPostgresExporterCollectors, exporter.ExporterOptions.EnabledCollectors),
		exporter.ExporterOptions.DisabledCollectors,
		pmmAgentVersion,
	)...)

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, false)

	sort.Strings(args)

	dsnParams := models.DSNParams{
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}

	dsnParams.DialTimeout = dbExporterDialTimeout(serviceNode, exporter)

	env := []string{"DATA_SOURCE_NAME=" + exporter.DSN(service, dsnParams, nil, pmmAgentVersion)}
	if passwordFile != "" && service.ServiceName == models.PMMServerPostgreSQLServiceName &&
		pointer.GetString(exporter.PMMAgentID) == models.PMMServerAgentID {
		env = passwordFileEnv(service, exporter, dsnParams, tdp, pmmAgentVersion, passwordFile)
	}

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env:                env,
		TextFiles:          exporter.Files(),
	}

	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}

	err := ensureAuthParams(exporter, res, pmmAgentVersion, postgresExporterWebConfigVersion, false)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// passwordFileEnv returns the environment that makes postgres_exporter read its password from a file and build the
// data source name from the other variables.
func passwordFileEnv(
	service *models.Service,
	exporter *models.Agent,
	dsnParams models.DSNParams,
	tdp *models.DelimiterPair,
	pmmAgentVersion *version.Parsed,
	file string,
) []string {
	withoutCredentials := *exporter
	withoutCredentials.Username = nil
	withoutCredentials.Password = nil
	uri := strings.TrimPrefix(withoutCredentials.DSN(service, dsnParams, tdp, pmmAgentVersion), "postgres://")

	return []string{
		"DATA_SOURCE_URI=" + uri,
		"DATA_SOURCE_USER=" + pointer.GetString(exporter.Username),
		"DATA_SOURCE_PASS_FILE=" + file,
	}
}

// qanPostgreSQLPgStatementsAgentConfig returns desired configuration of qan-postgresql-pgstatements-agent built-in agent.
func qanPostgreSQLPgStatementsAgentConfig(
	service *models.Service,
	agent *models.Agent,
	pmmAgentVersion *version.Parsed,
) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	dnsParams := models.DSNParams{
		DialTimeout:              5 * time.Second, //nolint:mnd
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                   inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
		Dsn:                    agent.DSN(service, dnsParams, nil, pmmAgentVersion),
		MaxQueryLength:         agent.QANOptions.MaxQueryLength,
		DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}

// qanPostgreSQLPgStatMonitorAgentConfig returns desired configuration of qan-postgresql-pgstatmonitor-agent built-in agent.
func qanPostgreSQLPgStatMonitorAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	dnsParams := models.DSNParams{
		DialTimeout:              1 * time.Second,
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                   inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
		Dsn:                    agent.DSN(service, dnsParams, nil, pmmAgentVersion),
		DisableQueryExamples:   agent.QANOptions.QueryExamplesDisabled,
		MaxQueryLength:         agent.QANOptions.MaxQueryLength,
		DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
