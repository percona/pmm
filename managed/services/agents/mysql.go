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
	"bytes"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
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

	if pmmAgentVersion.IsFeatureSupported(version.MysqlExporterPluginCollector) {
		args = append(args, "--collect.plugins")
	}

	if !pmmAgentVersion.IsFeatureSupported(version.MysqlExporterMySQL8_4) {
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

	textFiles := exporter.Files()
	if textFiles != nil && !pmmAgentVersion.IsFeatureSupported(version.MysqlExporterMySQL8_4) {
		for k := range textFiles {
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
	}

	if exporter.TLSSkipVerify {
		if pmmAgentVersion.IsFeatureSupported(version.MysqlExporterMySQL8_4) {
			args = append(args, "--tls.insecure-skip-verify")
		} else {
			args = append(args, "--mysql.ssl-skip-verify")
		}
	}

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, false)

	sort.Strings(args)

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles:          textFiles,
	}

	switch {
	case !pmmAgentVersion.IsFeatureSupported(version.MysqlExporterMySQL8_4):
		env := []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: ""}, nil, pmmAgentVersion)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()),
		}
		res.Env = env
	case textFiles != nil:
		cfg, err := buildMyCnfConfig(service, exporter, textFiles)
		if err != nil {
			return nil, err
		}
		res.TextFiles["myCnf"] = cfg
		res.Args = append(res.Args, "--config.my-cnf="+tdp.Left+" .TextFiles.myCnf "+tdp.Right)

		if err := ensureAuthParams(exporter, res, pmmAgentVersion, version.MysqlExporterMySQL8_4, true); err != nil {
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

// https://dev.mysql.com/doc/refman/8.4/en/mysql-command-options.html
// https://dev.mysql.com/doc/refman/8.4/en/connection-options.html#encrypted-connection-options
const myCnfTemplate = `[client]
{{if .Host}}host={{ .Host }}{{end}}
{{if .Port}}port={{ .Port }}{{end}}
{{if .User}}user={{ .User }}{{end}}
{{if .Password}}password={{ .Password }}{{end}}
{{if .Socket}}socket={{ .Socket }}{{end}}
{{if .CaFile}}ssl-ca={{ .CaFile }}{{end}}
{{if .CertFile}}ssl-cert={{ .CertFile }}{{end}}
{{if .KeyFile}}ssl-key={{ .KeyFile }}{{end}}
`

// BuildMyCnfConfig builds my.cnf configuration for MySQL connection.
func buildMyCnfConfig(service *models.Service, agent *models.Agent, files map[string]string) (string, error) {
	tmpl, err := template.New("myCnf").Parse(myCnfTemplate)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse my.cnf template")
	}
	tdp := agent.TemplateDelimiters(service)

	var configBuffer bytes.Buffer
	myCnfParams := struct {
		User      string
		Password  string
		Socket    string
		Host      string
		Port      int
		CaFile    string
		CertFile  string
		KeyFile   string
		MyCnfPath string
	}{
		User:     pointer.GetString(agent.Username),
		Password: pointer.GetString(agent.Password),
		Host:     pointer.GetString(service.Address),
		Port:     int(pointer.GetUint16(service.Port)),
	}

	if files["tlsCa"] != "" {
		myCnfParams.CaFile = tdp.Left + " .TextFiles.tlsCa " + tdp.Right
	}
	if files["tlsCert"] != "" {
		myCnfParams.CertFile = tdp.Left + " .TextFiles.tlsCert " + tdp.Right
	}
	if files["tlsKey"] != "" {
		myCnfParams.KeyFile = tdp.Left + " .TextFiles.tlsKey " + tdp.Right
	}

	if service.Socket != nil {
		myCnfParams.Socket = *service.Socket
	}

	if err = tmpl.Execute(&configBuffer, myCnfParams); err != nil {
		return "", errors.Wrap(err, "Failed to execute myCnf template")
	}

	return configBuffer.String(), nil
}
