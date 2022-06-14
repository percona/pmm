// pmm-admin
// Copyright 2019 Percona LLC
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
	"fmt"
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/ha_proxy"
)

var addHAProxyResultT = commands.ParseTemplate(`
HAProxy Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addHAProxyResult struct {
	Service *ha_proxy.AddHAProxyOKBodyService `json:"service"`
}

func (res *addHAProxyResult) Result() {}

func (res *addHAProxyResult) String() string {
	return commands.RenderTemplate(addHAProxyResultT, res)
}

func (cmd *AddHAProxyCmd) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

func (cmd *AddHAProxyCmd) RunCmd() (commands.Result, error) {
	isSupported, err := helpers.IsHAProxySupported()
	if !isSupported {
		return nil, err
	}

	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &ha_proxy.AddHAProxyParams{
		Body: ha_proxy.AddHAProxyBody{
			ServiceName:         cmd.ServiceName,
			Username:            cmd.Username,
			Password:            cmd.Password,
			Scheme:              cmd.Scheme,
			MetricsPath:         cmd.MetricsPath,
			ListenPort:          int64(cmd.ListenPort),
			NodeID:              cmd.NodeID,
			Environment:         cmd.Environment,
			Cluster:             cmd.Cluster,
			ReplicationSet:      cmd.ReplicationSet,
			CustomLabels:        customLabels,
			MetricsMode:         pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			SkipConnectionCheck: cmd.SkipConnectionCheck,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.HAProxy.AddHAProxy(params)
	if err != nil {
		return nil, err
	}

	return &addHAProxyResult{
		Service: resp.Payload.Service,
	}, nil
}
