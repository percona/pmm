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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

var (
	postgresExporterAutodiscoveryVersion = version.MustParse("2.15.99")
	postgresExporterWebConfigVersion     = version.MustParse("2.30.99")
	postgresSSLSniVersion                = version.MustParse("2.40.99")
	postgresMaxExporterConnsVersion      = version.MustParse("2.41.2-0")
)

const defaultAutoDiscoveryDatabaseLimit = 50

func postgresExcludedDatabases() []string {
	return []string{"template0", "template1", "postgres", "cloudsqladmin", "pmm-managed-dev", "azure_maintenance", "rdsadmin"}
}

// postgresExporterConfig returns desired configuration of postgres_exporter process.
func postgresExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentpb.SetStateRequest_AgentProcess, error) {
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
		case exporter.PostgreSQLOptions == nil:
			autoDiscovery = true
		case exporter.PostgreSQLOptions.AutoDiscoveryLimit == 0: // server defined
			autoDiscovery = exporter.PostgreSQLOptions.DatabaseCount <= defaultAutoDiscoveryDatabaseLimit
		case exporter.PostgreSQLOptions.AutoDiscoveryLimit < 0: // always disabled
		default:
			autoDiscovery = exporter.PostgreSQLOptions.DatabaseCount <= exporter.PostgreSQLOptions.AutoDiscoveryLimit
		}
	}
	if autoDiscovery {
		args = append(args,
			"--auto-discover-databases",
			fmt.Sprintf("--exclude-databases=%s", strings.Join(postgresExcludedDatabases(), ",")))
	}

	if !pmmAgentVersion.Less(postgresMaxExporterConnsVersion) &&
		exporter.PostgreSQLOptions != nil &&
		exporter.PostgreSQLOptions.MaxExporterConnections != 0 {
		args = append(args, "--max-connections="+strconv.Itoa(int(exporter.PostgreSQLOptions.MaxExporterConnections)))
	}

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsPath)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.DisabledCollectors)

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, false)

	sort.Strings(args)

	dnsParams := models.DSNParams{
		DialTimeout:              1 * time.Second,
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}

	if exporter.AzureOptions != nil {
		dnsParams.DialTimeout = 5 * time.Second
	}

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, dnsParams, nil, pmmAgentVersion)),
		},
		TextFiles: exporter.Files(),
	}

	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}

	if err := ensureAuthParams(exporter, res, pmmAgentVersion, postgresExporterWebConfigVersion); err != nil {
		return nil, err
	}

	return res, nil
}

// qanPostgreSQLPgStatementsAgentConfig returns desired configuration of qan-postgresql-pgstatements-agent built-in agent.
func qanPostgreSQLPgStatementsAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	dnsParams := models.DSNParams{
		DialTimeout:              5 * time.Second,
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                   inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
		Dsn:                    agent.DSN(service, dnsParams, nil, pmmAgentVersion),
		MaxQueryLength:         agent.MaxQueryLength,
		DisableCommentsParsing: agent.CommentsParsingDisabled,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}

// qanPostgreSQLPgStatMonitorAgentConfig returns desired configuration of qan-postgresql-pgstatmonitor-agent built-in agent.
func qanPostgreSQLPgStatMonitorAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	dnsParams := models.DSNParams{
		DialTimeout:              1 * time.Second,
		Database:                 service.DatabaseName,
		PostgreSQLSupportsSSLSNI: !pmmAgentVersion.Less(postgresSSLSniVersion),
	}
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                   inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
		Dsn:                    agent.DSN(service, dnsParams, nil, pmmAgentVersion),
		DisableQueryExamples:   agent.QueryExamplesDisabled,
		MaxQueryLength:         agent.MaxQueryLength,
		DisableCommentsParsing: agent.CommentsParsingDisabled,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
