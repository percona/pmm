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
	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

var addValkeyResultT = commands.ParseTemplate(`
Valkey Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addValkeyResult struct {
	Service        *mservice.AddServiceOKBodyValkeyService        `json:"service"`
	ValkeyExporter *mservice.AddServiceOKBodyValkeyValkeyExporter `json:"valkey_exporter,omitempty"`
}

func (res *addValkeyResult) Result() {}

func (res *addValkeyResult) String() string {
	return commands.RenderTemplate(addValkeyResultT, res)
}

// AddValkeyCommand is used by Kong for CLI flags and commands.
type AddValkeyCommand struct {
	ServiceName         string            `name:"name" arg:"" default:"${hostname}-valkey" help:"Service name (autodetected default: ${hostname}-valkey)"`
	Address             string            `arg:"" optional:"" help:"Valkey address and port (default: 127.0.0.1:6379)"`
	Socket              string            `help:"Path to Valkey socket"`
	NodeID              string            `help:"Node ID (default is autodetected)"`
	PMMAgentID          string            `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username            string            `default:"root" help:"Valkey username"`
	Password            string            `help:"Valkey password"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	Environment         string            `help:"Environment name"`
	Cluster             string            `help:"Cluster name"`
	ReplicationSet      string            `help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	TLSCaFile           string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile         string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile          string            `name:"tls-key" help:"Path to client key file"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	ExposeExporter      bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`

	AddCommonFlags
	flags.MetricsModeFlags
	flags.CommentsParsingFlags
	flags.LogLevelNoFatalFlags
}

// GetServiceName returns the service name for AddValkeyCommand.
func (cmd *AddValkeyCommand) GetServiceName() string {
	return cmd.ServiceName
}

// GetAddress returns the address for AddValkeyCommand.
func (cmd *AddValkeyCommand) GetAddress() string {
	return cmd.Address
}

// GetDefaultAddress returns the default address for AddValkeyCommand.
func (cmd *AddValkeyCommand) GetDefaultAddress() string {
	return "127.0.0.1:6379"
}

// GetSocket returns the socket for AddValkeyCommand.
func (cmd *AddValkeyCommand) GetSocket() string {
	return cmd.Socket
}

// RunCmd runs the command for AddValkeyCommand.
func (cmd *AddValkeyCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	var (
		err                    error
		tlsCa, tlsCert, tlsKey string
	)
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCaFile)
		if err != nil {
			return nil, err
		}

		tlsCert, err = commands.ReadFile(cmd.TLSCertFile)
		if err != nil {
			return nil, err
		}

		tlsKey, err = commands.ReadFile(cmd.TLSKeyFile)
		if err != nil {
			return nil, err
		}
	}

	if cmd.PMMAgentID == "" || cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.PMMAgentID == "" {
			cmd.PMMAgentID = status.AgentID
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)
	if err != nil {
		return nil, err
	}

	params := &mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Valkey: &mservice.AddServiceParamsBodyValkey{
				NodeID:         cmd.NodeID,
				ServiceName:    serviceName,
				Address:        host,
				Socket:         socket,
				Port:           int64(port),
				ExposeExporter: cmd.ExposeExporter,
				PMMAgentID:     cmd.PMMAgentID,
				Environment:    cmd.Environment,
				Cluster:        cmd.Cluster,
				ReplicationSet: cmd.ReplicationSet,
				Username:       cmd.Username,
				Password:       cmd.Password,
				AgentPassword:  cmd.AgentPassword,
				CustomLabels:   customLabels,

				SkipConnectionCheck: cmd.SkipConnectionCheck,

				TLS:               cmd.TLS,
				TLSSkipVerify:     cmd.TLSSkipVerify,
				TLSCa:             tlsCa,
				TLSCert:           tlsCert,
				TLSKey:            tlsKey,
				MetricsMode:       cmd.MetricsModeFlags.MetricsMode.EnumValue(),
				DisableCollectors: commands.ParseDisableCollectors(cmd.DisableCollectors),
				LogLevel:          cmd.LogLevelNoFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ManagementService.AddService(params)
	if err != nil {
		return nil, err
	}

	return &addValkeyResult{
		Service:        resp.Payload.Valkey.Service,
		ValkeyExporter: resp.Payload.Valkey.ValkeyExporter,
	}, nil
}
