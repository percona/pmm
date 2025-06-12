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
	"os"
	"strings"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

const (
	// DefaultGroupExternalExporter defines default group name for external exporter.
	DefaultGroupExternalExporter = "external"
	// DefaultServiceNameSuffix defines default service suffix for external exporter.
	DefaultServiceNameSuffix = "-external"
)

var addExternalResultT = commands.ParseTemplate(`
External Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
Group       : {{ .Service.Group }}
`)

type addExternalResult struct {
	Service *mservice.AddServiceOKBodyExternalService `json:"service"`
}

func (res *addExternalResult) Result() {}

func (res *addExternalResult) String() string {
	return commands.RenderTemplate(addExternalResultT, res)
}

// AddExternalCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddExternalCommand struct {
	ServiceName         string            `default:"${hostname}${externalDefaultServiceName}" help:"Service name (autodetected default: ${hostname}${externalDefaultServiceName})"`
	RunsOnNodeID        string            `name:"agent-node-id" help:"Node ID where agent runs (default is autodetected)"`
	Username            string            `help:"External username"`
	Password            string            `help:"External password"`
	CredentialsSource   string            `type:"existingfile" help:"Credentials provider"`
	Scheme              string            `placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string            `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16            `placeholder:"port" required:"" help:"Listen port of external exporter for scraping metrics. (Required)"`
	NodeID              string            `name:"service-node-id" help:"Node ID where service runs (default is autodetected)"`
	Environment         string            `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string            `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string            `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	Group               string            `default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	SkipConnectionCheck bool              `help:"Skip exporter connection checks"`

	flags.MetricsModeFlags
}

// GetCredentials returns the credentials for AddExternalCommand.
func (cmd *AddExternalCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

// RunCmd runs the command for AddExternalCommand.
func (cmd *AddExternalCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(&cmd.CustomLabels)

	if cmd.RunsOnNodeID == "" || cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.RunsOnNodeID == "" {
			cmd.RunsOnNodeID = status.NodeID
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	hostname, _ := os.Hostname()
	defaultServiceName := hostname + DefaultServiceNameSuffix

	if cmd.Group != DefaultGroupExternalExporter && cmd.ServiceName == defaultServiceName {
		cmd.ServiceName = fmt.Sprintf("%s-%s", strings.TrimSuffix(cmd.ServiceName, DefaultServiceNameSuffix), cmd.Group)
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			External: &mservice.AddServiceParamsBodyExternal{
				RunsOnNodeID:        cmd.RunsOnNodeID,
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
				CustomLabels:        *customLabels,
				MetricsMode:         cmd.MetricsModeFlags.MetricsMode.EnumValue(),
				Group:               cmd.Group,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
			},
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ManagementService.AddService(params)
	if err != nil {
		return nil, err
	}

	return &addExternalResult{
		Service: resp.Payload.External.Service,
	}, nil
}
