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
	"maps"
	"strconv"
	"strings"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateMongoDBCommand is used by Kong for CLI flags and commands.
type UpdateMongoDBCommand struct {
	AddMongoDBCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateMongoDBCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateMongoDBCommand.
func (cmd *UpdateMongoDBCommand) RunCmd() (commands.Result, error) {
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

	tlsCertificateKey, err := cmd.file("tls-certificate-key-file", cmd.TLSCertificateKeyFile)
	if err != nil {
		return nil, err
	}

	tlsCa, err := cmd.file("tls-ca-file", cmd.TLSCaFile)
	if err != nil {
		return nil, err
	}

	body := &mservice.UpdateServiceParamsBodyMongodb{
		Address:                        host,
		Port:                           port,
		Socket:                         socket,
		Environment:                    cmd.str("environment", cmd.Environment),
		Cluster:                        cmd.str("cluster", cmd.Cluster),
		ReplicationSet:                 cmd.str("replication-set", cmd.ReplicationSet),
		Username:                       username,
		Password:                       password,
		AgentPassword:                  agentPassword,
		TLS:                            cmd.boolean("tls", cmd.TLS),
		TLSSkipVerify:                  cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		TLSCertificateKey:              tlsCertificateKey,
		TLSCertificateKeyFilePassword:  cmd.str("tls-certificate-key-file-password", cmd.TLSCertificateKeyFilePassword),
		TLSCa:                          tlsCa,
		AuthenticationMechanism:        cmd.str("authentication-mechanism", cmd.AuthenticationMechanism),
		AuthenticationDatabase:         cmd.str("authentication-database", cmd.AuthenticationDatabase),
		MaxQueryLength:                 cmd.num32("max-query-length", cmd.MaxQueryLength),
		MetricsMode:                    cmd.metricsMode(cmd.MetricsMode),
		CollectionsLimit:               cmd.num32("max-collections-limit", cmd.CollectionsLimit),
		EnableAllCollectors:            cmd.boolean("enable-all-collectors", cmd.EnableAllCollectors),
		EnableDiagnosticDataHistograms: cmd.boolean("enable-diagnostic-data-histograms", cmd.EnableDiagnosticDataHistograms),
		LogLevel:                       cmd.logLevel(cmd.LogLevel),
		ExposeExporter:                 cmd.boolean("expose-exporter", cmd.ExposeExporter),
		ConnectionTimeout:              cmd.connectionTimeout(commands.DurationString(cmd.ConnectionTimeout)),
		SkipConnectionCheck:            cmd.SkipConnectionCheck,
	}

	if cmd.typed["query-source"] {
		body.QANMongodbProfiler = new(cmd.QuerySource == MongodbQuerySourceProfiler)
		body.QANMongodbMongolog = new(cmd.QuerySource == MongodbQuerySourceMongolog)
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyMongodbCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	if cmd.typed["disable-collectors"] {
		body.DisableCollectors = &mservice.UpdateServiceParamsBodyMongodbDisableCollectors{
			Values: cmd.list("disable-collectors", cmd.DisableCollectors),
		}
	}

	if cmd.typed["stats-collections"] {
		body.StatsCollections = &mservice.UpdateServiceParamsBodyMongodbStatsCollections{
			Values: cmd.list("stats-collections", cmd.StatsCollections),
		}
	}

	if cmd.typed["agent-env-vars"] {
		names, err := commands.ValidateEnvironmentVariableNames(cmd.AgentEnvVars)
		if err != nil {
			return nil, err
		}

		body.EnvironmentVariableNames = &mservice.UpdateServiceParamsBodyMongodbEnvironmentVariableNames{
			Values: names,
		}
	}

	target, err := cmd.target(cmd.typedName(cmd.ServiceName, cmd.AddCommonFlags), allServiceTypes["mongodb"])
	if err != nil {
		return nil, err
	}

	secretFlags := map[string][]string{
		"--tls-ca-file":                       {"tls-ca-file"},
		"--tls-certificate-key-file":          {"tls-certificate-key-file"},
		"--tls-certificate-key-file-password": {"tls-certificate-key-file-password"},
	}
	maps.Copy(secretFlags, credentialSecretFlags)

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Mongodb: body}, "MongoDB", mongodbSettings, secretFlags)
}

func mongodbSettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "mongodb_exporter")
	qan := agentOfType(svc, "qan-mongodb-profiler-agent", "qan-mongodb-mongolog-agent")

	options := exporter.MongoDBOptions
	if options == nil {
		options = &mservice.UpdateServiceOKBodyAfterAgentsItems0MongoDBOptions{}
	}

	return append(
		databaseServiceSettings(svc, exporter),
		setting{Flag: "--tls-certificate-key-file", Value: secretSettingValue(options.IsTLSCertificateKeySet)},
		setting{Flag: "--tls-certificate-key-file-password", Value: secretSettingValue(options.IsTLSCertificateKeyFilePasswordSet)},
		setting{Flag: "--authentication-mechanism", Value: options.AuthenticationMechanism},
		setting{Flag: "--authentication-database", Value: options.AuthenticationDatabase},
		setting{Flag: "--query-source", Value: querySource(svc, map[string]string{
			MongodbQuerySourceProfiler: "qan-mongodb-profiler-agent",
			MongodbQuerySourceMongolog: "qan-mongodb-mongolog-agent",
		})},
		setting{Flag: "--max-query-length", Value: strconv.Itoa(int(qan.MaxQueryLength))},
		boolSetting("--enable-all-collectors", options.EnableAllCollectors),
		boolSetting("--enable-diagnostic-data-histograms", options.EnableDiagnosticDataHistograms),
		setting{Flag: "--disable-collectors", Value: strings.Join(exporter.DisabledCollectors, ",")},
		setting{Flag: "--stats-collections", Value: strings.Join(options.StatsCollections, ",")},
		setting{Flag: "--max-collections-limit", Value: strconv.Itoa(int(options.CollectionsLimit))},
		setting{Flag: "--agent-env-vars", Value: strings.Join(exporter.EnvironmentVariableNames, ",")},
	)
}
