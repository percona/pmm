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
	"strings"

	"github.com/percona/pmm/admin/commands"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// UpdateExternalCommand is used by Kong for CLI flags and commands.
type UpdateExternalCommand struct {
	AddExternalCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateExternalCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateExternalCommand.
func (cmd *UpdateExternalCommand) RunCmd() (commands.Result, error) {
	err := cmd.rejectTyped("agent-node-id", "service-node-id")
	if err != nil {
		return nil, err
	}

	body, err := externalUpdateBody(&cmd.UpdateCommonFlags, cmd.Username, cmd.Password, cmd.CredentialsSource, cmd.Scheme, cmd.MetricsPath,
		cmd.ListenPort, cmd.Environment, cmd.Cluster, cmd.ReplicationSet, cmd.CustomLabels, cmd.Group, cmd.TLSSkipVerify, cmd.SkipConnectionCheck)
	if err != nil {
		return nil, err
	}
	body.MetricsMode = cmd.metricsMode(cmd.MetricsMode)

	target, err := cmd.target(cmd.typedValue("service-name", cmd.ServiceName), allServiceTypes["external"])
	if err != nil {
		return nil, err
	}

	return cmd.runUpdate(target, mservice.UpdateServiceBody{External: body}, "External", externalSettings, credentialSecretFlags)
}

// UpdateExternalServerlessCommand is used by Kong for CLI flags and commands.
type UpdateExternalServerlessCommand struct {
	AddExternalServerlessCommand
	UpdateCommonFlags
}

// Help returns the detailed help of the command.
func (cmd *UpdateExternalServerlessCommand) Help() string {
	return updateHelp
}

// RunCmd runs the command for UpdateExternalServerlessCommand.
func (cmd *UpdateExternalServerlessCommand) RunCmd() (commands.Result, error) {
	// The address and the other properties of the remote Node belong to the Node, not to the Service.
	err := cmd.rejectTyped("url", "address", "host", "machine-id", "distro", "container-id", "container-name", "node-model", "region", "az")
	if err != nil {
		return nil, err
	}

	body, err := externalUpdateBody(&cmd.UpdateCommonFlags, cmd.Username, cmd.Password, cmd.CredentialsSource, cmd.Scheme, cmd.MetricsPath,
		cmd.ListenPort, cmd.Environment, cmd.Cluster, cmd.ReplicationSet, cmd.CustomLabels, cmd.Group, cmd.TLSSkipVerify, cmd.SkipConnectionCheck)
	if err != nil {
		return nil, err
	}

	target, err := cmd.target(cmd.typedValue("external-name", cmd.Name), allServiceTypes["external"])
	if err != nil {
		return nil, err
	}

	return cmd.runUpdate(target, mservice.UpdateServiceBody{External: body}, "External", externalSettings, credentialSecretFlags)
}

// externalUpdateBody converts the flags shared by both external commands.
func externalUpdateBody(
	f *UpdateCommonFlags, username, password, credentialsSource, scheme, metricsPath string, listenPort uint16,
	environment, cluster, replicationSet string, customLabels map[string]string, group string, tlsSkipVerify, skipConnectionCheck bool,
) (*mservice.UpdateServiceParamsBodyExternal, error) {
	u, p, _, err := f.credentials(username, password, "", credentialsSource)
	if err != nil {
		return nil, err
	}

	body := &mservice.UpdateServiceParamsBodyExternal{
		Username:            u,
		Password:            p,
		Scheme:              f.str("scheme", scheme),
		MetricsPath:         f.str("metrics-path", normalizedMetricsPath(metricsPath)),
		Environment:         f.str("environment", environment),
		Cluster:             f.str("cluster", cluster),
		ReplicationSet:      f.str("replication-set", replicationSet),
		Group:               f.str("group", group),
		TLSSkipVerify:       f.boolean("tls-skip-verify", tlsSkipVerify),
		SkipConnectionCheck: skipConnectionCheck,
	}

	if f.typed["listen-port"] {
		body.ListenPort = new(int64(listenPort))
	}

	if f.typed["custom-labels"] {
		body.CustomLabels = &mservice.UpdateServiceParamsBodyExternalCustomLabels{
			Values: f.labels("custom-labels", customLabels),
		}
	}

	return body, nil
}

// normalizedMetricsPath adds the leading slash, as add does.
func normalizedMetricsPath(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") {
		return "/" + path
	}

	return path
}

func externalSettings(svc *mservice.UpdateServiceOKBodyAfter) []setting {
	exporter := agentOfType(svc, "external-exporter")

	return append(
		commonServiceSettings(svc, exporter),
		setting{"--group", svc.ExternalGroup},
		setting{"--scheme", exporter.MetricsScheme},
		setting{"--metrics-path", exporter.MetricsPath},
		setting{"--listen-port", strconv.FormatInt(exporter.ListenPort, 10)},
		setting{"--tls-skip-verify", boolSettingValue(exporter.TLSSkipVerify)},
	)
}
