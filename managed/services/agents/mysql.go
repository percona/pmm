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
	"time"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

var (
	mysqlExporterVersionWithPluginCollector = version.MustParse("2.36.0-0")
	// TODO: put back 3.2.0 when 3.1.0 is released.
	v3_2_0 = version.MustParse("3.1.0-0")
)

// mysqldExporterConfig returns desired configuration of mysqld_exporter process.
func mysqldExporterConfig(
	node *models.Node,
	service *models.Service,
	exporter *models.Agent,
	redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentv1.SetStateRequest_AgentProcess, error) {
	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		// LR
		"--collect.binlog_size",
		"--collect.engine_tokudb_status",
		"--collect.global_variables",
		"--collect.heartbeat",
		"--collect.info_schema.clientstats",
		"--collect.info_schema.userstats",
		"--collect.perf_schema.eventsstatements",
		"--collect.custom_query.lr",

		// MR
		"--collect.engine_innodb_status",
		"--collect.info_schema.innodb_cmp",
		"--collect.info_schema.innodb_cmpmem",
		"--collect.info_schema.processlist",
		"--collect.info_schema.query_response_time",
		"--collect.perf_schema.eventswaits",
		"--collect.perf_schema.file_events",
		"--collect.slave_status",
		"--collect.custom_query.mr",

		// HR
		"--collect.global_status",
		"--collect.info_schema.innodb_metrics",
		"--collect.custom_query.hr",
		"--collect.standard.go",
		"--collect.standard.process",

		"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/low-resolution",
		"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/medium-resolution",
		"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/high-resolution",

		"--exporter.max-idle-conns=3",
		"--exporter.max-open-conns=3",
		"--exporter.conn-max-lifetime=55s",
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if !pmmAgentVersion.Less(mysqlExporterVersionWithPluginCollector) {
		args = append(args, "--collect.plugins")
	}

	if pmmAgentVersion.Less(v3_2_0) {
		args = append(args, "--exporter.global-conn-pool")
	}

	if exporter.IsMySQLTablestatsGroupEnabled() {
		// keep in sync with Prometheus scrape configs generator
		tablestatsGroup := []string{
			// LR
			"--collect.info_schema.innodb_tablespaces",
			"--collect.auto_increment.columns",
			"--collect.info_schema.tables",
			"--collect.info_schema.tablestats",
			"--collect.perf_schema.indexiowaits",
			"--collect.perf_schema.tableiowaits",
			"--collect.perf_schema.file_instances",

			// MR
			"--collect.perf_schema.tablelocks",
		}
		args = append(args, tablestatsGroup...)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.ExporterOptions.DisabledCollectors)

	if exporter.ExporterOptions.MetricsPath != "" {
		args = append(args, "--web.telemetry-path="+exporter.ExporterOptions.MetricsPath)
	}

	files := exporter.Files()
	if files != nil {
		if pmmAgentVersion.Less(v3_2_0) {
			for k := range files {
				switch k {
				case "tlsCa":
					args = append(args, "--mysql.ssl-ca-file="+tdp.Left+" .TextFiles.tlsCa "+tdp.Right)
				case "tlsCert":
					args = append(args, "--mysql.ssl-cert-file="+tdp.Left+" .TextFiles.tlsCert "+tdp.Right)
				case "tlsKey":
					args = append(args, "--mysql.ssl-key-file="+tdp.Left+" .TextFiles.tlsKey "+tdp.Right)
				default:
					continue
				}
			}
		} else {
			for k := range files {
				switch k {
				case "tlsCa":
					args = append(args, "--tls.ssl-ca="+tdp.Left+" .TextFiles.tlsCa "+tdp.Right)
				case "tlsCert":
					args = append(args, "--tls.ssl-cert="+tdp.Left+" .TextFiles.tlsCert "+tdp.Right)
				case "tlsKey":
					args = append(args, "--tls.ssl-key="+tdp.Left+" .TextFiles.tlsKey "+tdp.Right)
				default:
					continue
				}
			}
		}

		if exporter.TLSSkipVerify {
			if pmmAgentVersion.Less(v3_2_0) {
				args = append(args, "--mysql.ssl-skip-verify")
			} else {
				args = append(args, "--tls.insecure-skip-verify")
			}
		}
	}

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, false)

	sort.Strings(args)

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles:          files,
	}
	if pmmAgentVersion.Less(v3_2_0) {
		env := []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil, pmmAgentVersion)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()),
		}
		res.Env = env
	} else {
		cfg, err := exporter.BuildMyCnfConfig(service)
		if err != nil {
			return nil, err
		}
		res.TextFiles["myCnf"] = cfg
		res.Args = append(res.Args, "--config.my-cnf="+tdp.Left+" .TextFiles.myCnf "+tdp.Right)

		if err := ensureAuthParams(exporter, res, pmmAgentVersion, v3_2_0, true); err != nil {
			return nil, err
		}
	}

	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res, nil
}

// qanMySQLPerfSchemaAgentConfig returns desired configuration of qan-mysql-perfschema built-in agent.
func qanMySQLPerfSchemaAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                   inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:                    agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil, pmmAgentVersion),
		MaxQueryLength:         agent.QANOptions.MaxQueryLength,
		DisableQueryExamples:   agent.QANOptions.QueryExamplesDisabled,
		DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		TlsSkipVerify: agent.TLSSkipVerify,
	}
}

// qanMySQLSlowlogAgentConfig returns desired configuration of qan-mysql-slowlog built-in agent.
func qanMySQLSlowlogAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                   inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_SLOWLOG_AGENT,
		Dsn:                    agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil, pmmAgentVersion),
		MaxQueryLength:         agent.QANOptions.MaxQueryLength,
		DisableQueryExamples:   agent.QANOptions.QueryExamplesDisabled,
		DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
		MaxQueryLogSize:        agent.QANOptions.MaxQueryLogSize,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		TlsSkipVerify: agent.TLSSkipVerify,
	}
}
