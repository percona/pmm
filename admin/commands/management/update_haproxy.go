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
	"strconv"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateHAProxyCommand is used by Kong for CLI flags and commands.
type UpdateHAProxyCommand struct {
	AddHAProxyCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateHAProxyCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateHAProxyCommand.
func (cmd *UpdateHAProxyCommand) RunCmd() (commands.Result, error) {
	err := cmd.rejectTyped("node-id")
	if err != nil {
		return nil, err
	}

	username, password, _, err := cmd.credentials(cmd.Username, cmd.Password, "", cmd.CredentialsSource)
	if err != nil {
		return nil, err
	}

	body := &mservice.UpdateServiceParamsBodyHaproxy{
		Username:            username,
		Password:            password,
		Scheme:              cmd.str("scheme", cmd.Scheme),
		MetricsPath:         cmd.str("metrics-path", normalizedMetricsPath(cmd.MetricsPath)),
		Environment:         cmd.str("environment", cmd.Environment),
		Cluster:             cmd.str("cluster", cmd.Cluster),
		ReplicationSet:      cmd.str("replication-set", cmd.ReplicationSet),
		MetricsMode:         cmd.metricsMode(cmd.MetricsMode),
		TLSSkipVerify:       cmd.boolean("tls-skip-verify", cmd.TLSSkipVerify),
		SkipConnectionCheck: cmd.SkipConnectionCheck,
	}

	if cmd.typed["listen-port"] {
		body.ListenPort = new(int64(cmd.ListenPort))
	}

	if cmd.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyHaproxyCustomLabels{
			Values: cmd.labels("custom-labels", cmd.CustomLabels),
		}
	}

	target, err := cmd.target(cmd.typedValue("name", cmd.ServiceName), allServiceTypes["haproxy"])
	if err != nil {
		return nil, err
	}

	return cmd.runUpdate(target, mservice.UpdateServiceBody{Haproxy: body}, "HAProxy", haproxySettings, credentialSecretFlags)
}

func haproxySettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "external-exporter")

	return append(
		commonServiceSettings(svc, exporter),
		setting{"--scheme", exporter.MetricsScheme},
		setting{"--metrics-path", exporter.MetricsPath},
		setting{"--listen-port", strconv.FormatInt(exporter.ListenPort, 10)},
		setting{"--tls-skip-verify", boolSettingValue(exporter.TLSSkipVerify)},
	)
}
