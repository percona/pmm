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

	"github.com/alecthomas/units"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateMySQLCommand is used by Kong for CLI flags and commands.
type UpdateMySQLCommand struct {
	AddMySQLCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateMySQLCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateMySQLCommand.
func (cmd *UpdateMySQLCommand) RunCmd() (commands.Result, error) {
	err := cmd.rejectTyped("node-id", "pmm-agent-id", "create-user")
	if err != nil {
		return nil, err
	}

	host, port, socket, err := cmd.endpoint(cmd.Address, cmd.AddCommonFlags, cmd.Socket)
	if err != nil {
		return nil, err
	}

	tlsCa, err := cmd.file("tls-ca", cmd.TLSCaFile)
	if err != nil {
		return nil, err
	}

	tlsCert, err := cmd.file("tls-cert", cmd.TLSCertFile)
	if err != nil {
		return nil, err
	}

	tlsKey, err := cmd.file("tls-key", cmd.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	tablestatsGroupTableLimit, err := cmd.tablestatsGroupTableLimit()
	if err != nil {
		return nil, err
	}

	body := &mservice.UpdateServiceParamsBodyMysql{
		Address:                   host,
		Port:                      port,
		Socket:                    socket,
		Environment:               cmd.str("environment", cmd.Environment),
		Cluster:                   cmd.str("cluster", cmd.Cluster),
		ReplicationSet:            cmd.str("replication-set", cmd.ReplicationSet),
		Username:                  cmd.str("username", cmd.Username),
		Password:                  cmd.str("password", cmd.Password),
		AgentPassword:             cmd.str("agent-password", cmd.AgentPassword),
		TLS:                       cmd.boolean("tls", cmd.TLS),
		TLSSkipVerify:             cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		TLSCa:                     tlsCa,
		TLSCert:                   tlsCert,
		TLSKey:                    tlsKey,
		MaxQueryLength:            cmd.num32("max-query-length", cmd.MaxQueryLength),
		DisableQueryExamples:      cmd.boolean("disable-queryexamples", cmd.DisableQueryExamples),
		TablestatsGroupTableLimit: tablestatsGroupTableLimit,
		MetricsMode:               cmd.metricsMode(cmd.MetricsMode),
		LogLevel:                  cmd.logLevel(cmd.LogLevel),
		ExposeExporter:            cmd.boolean("expose-exporter", cmd.ExposeExporter),
		ConnectionTimeout:         cmd.connectionTimeout(commands.DurationString(cmd.ConnectionTimeout)),
		SkipConnectionCheck:       cmd.SkipConnectionCheck,
	}

	if cmd.typed["query-source"] {
		body.QANMysqlSlowlog = new(cmd.QuerySource == MysqlQuerySourceSlowLog)
		body.QANMysqlPerfschema = new(cmd.QuerySource == MysqlQuerySourcePerfSchema)
	}

	if cmd.typed["size-slow-logs"] {
		body.MaxSlowlogFileSize = new(strconv.FormatInt(int64(cmd.MaxSlowlogFileSize), 10))
	}

	if cmd.typed["comments-parsing"] {
		body.DisableCommentsParsing = new(!cmd.CommentsParsingEnabled())
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyMysqlCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	if cmd.typed["extra-dsn-params"] {
		body.ExtraDsnParams = &mservice.UpdateServiceParamsBodyMysqlExtraDsnParams{
			Values: cmd.labels("extra-dsn-params", cmd.ExtraDSNParams),
		}
	}

	if cmd.typed["disable-collectors"] {
		body.DisableCollectors = &mservice.UpdateServiceParamsBodyMysqlDisableCollectors{
			Values: cmd.list("disable-collectors", cmd.DisableCollectors),
		}
	}

	target, err := cmd.target(cmd.typedName(cmd.ServiceName, cmd.AddCommonFlags), allServiceTypes["mysql"])
	if err != nil {
		return nil, err
	}

	secretFlags := map[string][]string{
		"--tls-cert": {"tls-cert"},
		"--tls-key":  {"tls-key"},
	}
	maps.Copy(secretFlags, credentialSecretFlags)

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Mysql: body}, "MySQL", mysqlSettings, secretFlags)
}

// tablestatsGroupTableLimit combines --disable-tablestats and --disable-tablestats-limit the way add does;
// turning tablestats back on without a limit restores the default limit.
func (cmd *UpdateMySQLCommand) tablestatsGroupTableLimit() (*int32, error) {
	disable, limit := cmd.typed["disable-tablestats"], cmd.typed["disable-tablestats-limit"]
	switch {
	case disable && cmd.DisableTablestats:
		if limit && cmd.DisableTablestatsLimit != 0 {
			return nil, errors.New("both --disable-tablestats and --disable-tablestats-limit are passed")
		}

		return new(int32(-1)), nil
	case disable || limit:
		return new(int32(cmd.DisableTablestatsLimit)), nil
	default:
		return nil, nil //nolint:nilnil
	}
}

func mysqlSettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "mysqld_exporter")
	qan := agentOfType(svc, "qan-mysql-slowlog-agent", "qan-mysql-perfschema-agent")
	slowlog := agentOfType(svc, "qan-mysql-slowlog-agent")

	var tlsKeySet bool
	var extraDSNParams map[string]string
	if exporter.MysqlOptions != nil {
		tlsKeySet = exporter.MysqlOptions.IsTLSKeySet
		extraDSNParams = exporter.MysqlOptions.ExtraDsnParams
	}

	limit := exporter.TableCountTablestatsGroupLimit
	disableTablestatsLimit := ""
	if limit > 0 {
		disableTablestatsLimit = strconv.Itoa(int(limit))
	}

	return append(
		databaseServiceSettings(svc, exporter),
		setting{"--tls-key", secretSettingValue(tlsKeySet)},
		setting{"--extra-dsn-params", labelsSettingValue(extraDSNParams)},
		setting{"--query-source", querySource(svc, map[string]string{
			MysqlQuerySourceSlowLog:    "qan-mysql-slowlog-agent",
			MysqlQuerySourcePerfSchema: "qan-mysql-perfschema-agent",
		})},
		setting{"--max-query-length", strconv.Itoa(int(qan.MaxQueryLength))},
		setting{"--disable-queryexamples", boolSettingValue(qan.QueryExamplesDisabled)},
		setting{"--comments-parsing", commentsParsingSettingValue(qan.CommentsParsingDisabled)},
		setting{"--size-slow-logs", slowlogSizeSettingValue(slowlog.MaxQueryLogSize)},
		setting{"--disable-tablestats", boolSettingValue(limit < 0)},
		setting{"--disable-tablestats-limit", disableTablestatsLimit},
		setting{"--disable-collectors", strings.Join(exporter.DisabledCollectors, ",")},
	)
}

func commentsParsingSettingValue(disabled bool) string {
	if disabled {
		return "off"
	}

	return "on"
}

// slowlogSizeSettingValue formats the stored slowlog rotation size the way --size-slow-logs takes it;
// a stored zero means that rotation is disabled.
func slowlogSizeSettingValue(size string) string {
	if size == "" {
		return ""
	}

	n, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return size
	}

	if n == 0 {
		return "-1"
	}

	return units.Base2Bytes(n).String()
}
