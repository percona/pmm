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

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateValkeyCommand is used by Kong for CLI flags and commands.
type UpdateValkeyCommand struct {
	AddValkeyCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateValkeyCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateValkeyCommand.
func (cmd *UpdateValkeyCommand) RunCmd() (commands.Result, error) {
	err := cmd.rejectTyped("node-id", "pmm-agent-id")
	if err != nil {
		return nil, err
	}

	if cmd.typed["disable-collectors"] {
		return nil, errors.New("--disable-collectors is not supported for Valkey")
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

	body := &mservice.UpdateServiceParamsBodyValkey{
		Address:             host,
		Port:                port,
		Socket:              socket,
		Environment:         cmd.str("environment", cmd.Environment),
		Cluster:             cmd.str("cluster", cmd.Cluster),
		ReplicationSet:      cmd.str("replication-set", cmd.ReplicationSet),
		Username:            cmd.str("username", cmd.Username),
		Password:            cmd.str("password", cmd.Password),
		AgentPassword:       cmd.str("agent-password", cmd.AgentPassword),
		TLS:                 cmd.boolean("tls", cmd.TLS),
		TLSSkipVerify:       cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		TLSCa:               tlsCa,
		TLSCert:             tlsCert,
		TLSKey:              tlsKey,
		MetricsMode:         cmd.metricsMode(cmd.MetricsMode),
		LogLevel:            cmd.logLevel(cmd.LogLevel),
		ExposeExporter:      cmd.boolean("expose-exporter", cmd.ExposeExporter),
		ConnectionTimeout:   cmd.connectionTimeout(commands.DurationString(cmd.ConnectionTimeout)),
		SkipConnectionCheck: cmd.SkipConnectionCheck,
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyValkeyCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	target, err := cmd.target(cmd.typedName(cmd.ServiceName, cmd.AddCommonFlags), allServiceTypes["valkey"])
	if err != nil {
		return nil, err
	}

	secretFlags := map[string][]string{
		"--tls-cert": {"tls-cert"},
		"--tls-key":  {"tls-key"},
	}
	maps.Copy(secretFlags, credentialSecretFlags)

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Valkey: body}, "Valkey", valkeySettings, secretFlags)
}

func valkeySettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	return databaseServiceSettings(svc, agentOfType(svc, "valkey_exporter"))
}
