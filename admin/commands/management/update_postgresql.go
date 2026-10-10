// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package management

import (
	"errors"
	"maps"
	"strconv"
	"strings"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdatePostgreSQLCommand is used by Kong for CLI flags and commands.
type UpdatePostgreSQLCommand struct {
	AddPostgreSQLCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdatePostgreSQLCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdatePostgreSQLCommand.
func (cmd *UpdatePostgreSQLCommand) RunCmd() (commands.Result, error) {
	err := cmd.rejectTyped("node-id", "pmm-agent-id")
	if err != nil {
		return nil, err
	}

	username, password, agentPassword, err := cmd.credentials(cmd.Username, cmd.Password, cmd.AgentPassword, cmd.CredentialsSource)
	if err != nil {
		return nil, err
	}

	host, port, socket, err := cmd.endpoint(cmd.Address, cmd.AddCommonFlags, cmd.Socket)
	if err != nil {
		return nil, err
	}

	tlsCa, err := cmd.file("tls-ca-file", cmd.TLSCAFile)
	if err != nil {
		return nil, err
	}

	tlsCert, err := cmd.file("tls-cert-file", cmd.TLSCertFile)
	if err != nil {
		return nil, err
	}

	tlsKey, err := cmd.file("tls-key-file", cmd.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	body := &mservice.UpdateServiceParamsBodyPostgresql{
		Address:                host,
		Port:                   port,
		Socket:                 socket,
		Database:               cmd.str("database", cmd.Database),
		Environment:            cmd.str("environment", cmd.Environment),
		Cluster:                cmd.str("cluster", cmd.Cluster),
		ReplicationSet:         cmd.str("replication-set", cmd.ReplicationSet),
		Username:               username,
		Password:               password,
		AgentPassword:          agentPassword,
		TLS:                    cmd.boolean("tls", cmd.TLS),
		TLSSkipVerify:          cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		TLSCa:                  tlsCa,
		TLSCert:                tlsCert,
		TLSKey:                 tlsKey,
		MaxQueryLength:         cmd.num32("max-query-length", cmd.MaxQueryLength),
		DisableQueryExamples:   cmd.boolean("disable-queryexamples", cmd.DisableQueryExamples),
		MetricsMode:            cmd.metricsMode(cmd.MetricsMode),
		AutoDiscoveryLimit:     cmd.num32("auto-discovery-limit", cmd.AutoDiscoveryLimit),
		MaxExporterConnections: cmd.num32("max-exporter-connections", cmd.MaxExporterConnections),
		LogLevel:               cmd.logLevel(cmd.LogLevel),
		ExposeExporter:         cmd.boolean("expose-exporter", cmd.ExposeExporter),
		ConnectionTimeout:      cmd.connectionTimeout(commands.DurationString(cmd.ConnectionTimeout)),
		SkipConnectionCheck:    cmd.SkipConnectionCheck,
	}

	if cmd.typed["query-source"] {
		switch cmd.QuerySource {
		case "pgstatements", "pgstatmonitor", "none":
		default:
			return nil, errors.New("--query-source must be one of: pgstatements, pgstatmonitor, none")
		}

		body.QANPostgresqlPgstatementsAgent = new(cmd.QuerySource == "pgstatements")
		body.QANPostgresqlPgstatmonitorAgent = new(cmd.QuerySource == "pgstatmonitor")
	}

	if cmd.typed["comments-parsing"] {
		body.DisableCommentsParsing = new(!cmd.CommentsParsingEnabled())
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyPostgresqlCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	if cmd.typed["disable-collectors"] {
		body.DisableCollectors = &mservice.UpdateServiceParamsBodyPostgresqlDisableCollectors{
			Values: cmd.list("disable-collectors", cmd.DisableCollectors),
		}
	}

	target, err := cmd.target(cmd.typedName(cmd.ServiceName, cmd.AddCommonFlags), allServiceTypes["postgresql"])
	if err != nil {
		return nil, err
	}

	secretFlags := map[string][]string{
		"--tls-cert-file": {"tls-cert-file"},
		"--tls-key-file":  {"tls-key-file"},
	}
	maps.Copy(secretFlags, credentialSecretFlags)

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Postgresql: body}, "PostgreSQL", postgresqlSettings, secretFlags)
}

func postgresqlSettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "postgres_exporter")
	qan := agentOfType(svc, "qan-postgresql-pgstatmonitor-agent", "qan-postgresql-pgstatements-agent")

	var sslKeySet bool
	var autoDiscoveryLimit, maxExporterConnections int32
	if exporter.PostgresqlOptions != nil {
		sslKeySet = exporter.PostgresqlOptions.IsSslKeySet
		autoDiscoveryLimit = exporter.PostgresqlOptions.AutoDiscoveryLimit
		maxExporterConnections = exporter.PostgresqlOptions.MaxExporterConnections
	}

	return append(
		databaseServiceSettings(svc, exporter),
		setting{"--database", svc.DatabaseName},
		setting{"--tls-key-file", secretSettingValue(sslKeySet)},
		setting{"--query-source", querySource(svc, map[string]string{
			"pgstatements":  "qan-postgresql-pgstatements-agent",
			"pgstatmonitor": "qan-postgresql-pgstatmonitor-agent",
		})},
		setting{"--max-query-length", strconv.Itoa(int(qan.MaxQueryLength))},
		setting{"--disable-queryexamples", boolSettingValue(qan.QueryExamplesDisabled)},
		setting{"--comments-parsing", commentsParsingSettingValue(qan.CommentsParsingDisabled)},
		setting{"--disable-collectors", strings.Join(exporter.DisabledCollectors, ",")},
		setting{"--auto-discovery-limit", strconv.Itoa(int(autoDiscoveryLimit))},
		setting{"--max-exporter-connections", strconv.Itoa(int(maxExporterConnections))},
	)
}
