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

// AddHAProxyCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddHAProxyCommand struct {
	ServiceName         string            `name:"name" arg:"" default:"${hostname}-haproxy" help:"Service name (autodetected default: ${hostname}-haproxy)"`
	Username            string            `help:"HAProxy username"`
	Password            string            `help:"HAProxy password"`
	CredentialsSource   string            `type:"existingfile" help:"Credentials provider"`
	Scheme              string            `placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string            `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16            `placeholder:"port" required:"" help:"Listen port of haproxy exposing the metrics for scraping metrics (Required)"`
	NodeID              string            `help:"Node ID (default is autodetected)"`
	Environment         string            `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string            `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string            `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	MetricsMode         string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
}

// GetCredentials returns the credentials for AddHAProxyCommand.
func (cmd *AddHAProxyCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

// RunCmd runs the command for AddHAProxyCommand.
func (cmd *AddHAProxyCommand) RunCmd() (commands.Result, error) {
	isSupported, err := helpers.IsHAProxySupported()
	if !isSupported {
		return nil, err
	}

	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

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
