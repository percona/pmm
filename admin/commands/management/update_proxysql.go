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
	"strings"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateProxySQLCommand is used by Kong for CLI flags and commands.
type UpdateProxySQLCommand struct {
	AddProxySQLCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateProxySQLCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateProxySQLCommand.
func (cmd *UpdateProxySQLCommand) RunCmd() (commands.Result, error) {
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

	body := &mservice.UpdateServiceParamsBodyProxysql{
		Address:             host,
		Port:                port,
		Socket:              socket,
		Environment:         cmd.str("environment", cmd.Environment),
		Cluster:             cmd.str("cluster", cmd.Cluster),
		ReplicationSet:      cmd.str("replication-set", cmd.ReplicationSet),
		Username:            username,
		Password:            password,
		AgentPassword:       agentPassword,
		TLS:                 cmd.boolean("tls", cmd.TLS),
		TLSSkipVerify:       cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		MetricsMode:         cmd.metricsMode(cmd.MetricsMode),
		LogLevel:            cmd.logLevel(cmd.LogLevel),
		ExposeExporter:      cmd.boolean("expose-exporter", cmd.ExposeExporter),
		ConnectionTimeout:   cmd.connectionTimeout(commands.DurationString(cmd.ConnectionTimeout)),
		SkipConnectionCheck: cmd.SkipConnectionCheck,
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyProxysqlCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	if cmd.typed["disable-collectors"] {
		body.DisableCollectors = &mservice.UpdateServiceParamsBodyProxysqlDisableCollectors{
			Values: cmd.list("disable-collectors", cmd.DisableCollectors),
		}
	}

	target, err := cmd.target(cmd.typedName(cmd.ServiceName, cmd.AddCommonFlags), allServiceTypes["proxysql"])
	if err != nil {
		return nil, err
	}

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Proxysql: body}, "ProxySQL", proxysqlSettings, credentialSecretFlags)
}

func proxysqlSettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "proxysql_exporter")

	return append(
		databaseServiceSettings(svc, exporter),
		setting{"--disable-collectors", strings.Join(exporter.DisabledCollectors, ",")},
	)
}
