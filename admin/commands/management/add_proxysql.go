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
	"github.com/percona/pmm/api/management/v1/json/client"
	proxysql "github.com/percona/pmm/api/management/v1/json/client/proxy_sql_service"
)

var addProxySQLResultT = commands.ParseTemplate(`
ProxySQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addProxySQLResult struct {
	Service *proxysql.AddProxySQLOKBodyService `json:"service"`
}

func (res *addProxySQLResult) Result() {}

func (res *addProxySQLResult) String() string {
	return commands.RenderTemplate(addProxySQLResultT, res)
}

// AddProxySQLCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddProxySQLCommand struct {
	ServiceName         string            `name:"name" arg:"" default:"${hostname}-proxysql" help:"Service name (autodetected default: ${hostname}-proxysql)"`
	Address             string            `arg:"" optional:"" help:"ProxySQL address and port (default: 127.0.0.1:6032)"`
	Socket              string            `help:"Path to ProxySQL socket"`
	NodeID              string            `help:"Node ID (default is autodetected)"`
	PMMAgentID          string            `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username            string            `default:"admin" help:"ProxySQL username"`
	Password            string            `default:"admin" help:"ProxySQL password"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	CredentialsSource   string            `type:"existingfile" help:"Credentials provider"`
	Environment         string            `help:"Environment name"`
	Cluster             string            `help:"Cluster name"`
	ReplicationSet      string            `help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	MetricsMode         string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	ExposeExporter      bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`

	AddCommonFlags
	AddLogLevelFatalFlags
}

func (cmd *AddProxySQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *AddProxySQLCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *AddProxySQLCommand) GetDefaultAddress() string {
	return "127.0.0.1:6032"
}

func (cmd *AddProxySQLCommand) GetSocket() string {
	return cmd.Socket
}

func (cmd *AddProxySQLCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.AgentPassword = creds.AgentPassword
	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

func (cmd *AddProxySQLCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

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

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &proxysql.AddProxySQLParams{
		Body: proxysql.AddProxySQLBody{
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

			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
			MetricsMode:         pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:            &cmd.AddLogLevel,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ProxySQLService.AddProxySQL(params)
	if err != nil {
		return nil, err
	}

	return &addProxySQLResult{
		Service: resp.Payload.Service,
	}, nil
}
