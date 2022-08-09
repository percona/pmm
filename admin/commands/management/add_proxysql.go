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
	"os"
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	proxysql "github.com/percona/pmm/api/managementpb/json/client/proxy_sql"
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

type addProxySQLCommand struct {
	Address           string
	Socket            string
	NodeID            string
	PMMAgentID        string
	ServiceName       string
	Username          string
	Password          string
	AgentPassword     string
	CredentialsSource string
	Environment       string
	Cluster           string
	ReplicationSet    string
	CustomLabels      string
	MetricsMode       string
	DisableCollectors string

	SkipConnectionCheck bool
	TLS                 bool
	TLSSkipVerify       bool
}

func (cmd *addProxySQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *addProxySQLCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *addProxySQLCommand) GetDefaultAddress() string {
	if cmd.CredentialsSource != "" {
		// address might be specified in credentials source file
		return ""
	}

	return "127.0.0.1:6032"
}

func (cmd *addProxySQLCommand) GetSocket() string {
	return cmd.Socket
}

func (cmd *addProxySQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
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

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd)
	if err != nil {
		return nil, err
	}

	params := &proxysql.AddProxySQLParams{
		Body: proxysql.AddProxySQLBody{
			NodeID:            cmd.NodeID,
			ServiceName:       serviceName,
			Address:           host,
			Port:              int64(port),
			Socket:            socket,
			PMMAgentID:        cmd.PMMAgentID,
			Environment:       cmd.Environment,
			Cluster:           cmd.Cluster,
			ReplicationSet:    cmd.ReplicationSet,
			Username:          cmd.Username,
			Password:          cmd.Password,
			AgentPassword:     cmd.AgentPassword,
			CredentialsSource: cmd.CredentialsSource,

			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
			MetricsMode:         pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:            &addLogLevel,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ProxySQL.AddProxySQL(params)
	if err != nil {
		return nil, err
	}

	return &addProxySQLResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command
var (
	AddProxySQL  addProxySQLCommand
	AddProxySQLC = AddC.Command("proxysql", "Add ProxySQL to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	serviceName := hostname + "-proxysql"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddProxySQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddProxySQL.ServiceName)

	AddProxySQLC.Arg("address", "ProxySQL address and port (default: 127.0.0.1:6032)").StringVar(&AddProxySQL.Address)
	AddProxySQLC.Flag("socket", "Path to ProxySQL socket").StringVar(&AddProxySQL.Socket)

	AddProxySQLC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddProxySQL.NodeID)
	AddProxySQLC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddProxySQL.PMMAgentID)

	AddProxySQLC.Flag("username", "ProxySQL username").Default("admin").StringVar(&AddProxySQL.Username)
	AddProxySQLC.Flag("password", "ProxySQL password").Default("admin").StringVar(&AddProxySQL.Password)
	AddProxySQLC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddProxySQL.AgentPassword)
	AddProxySQLC.Flag("credentials-source", "Credentials provider").StringVar(&AddProxySQL.CredentialsSource)

	AddProxySQLC.Flag("environment", "Environment name").StringVar(&AddProxySQL.Environment)
	AddProxySQLC.Flag("cluster", "Cluster name").StringVar(&AddProxySQL.Cluster)
	AddProxySQLC.Flag("replication-set", "Replication set name").StringVar(&AddProxySQL.ReplicationSet)
	AddProxySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddProxySQL.CustomLabels)

	AddProxySQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddProxySQL.SkipConnectionCheck)
	AddProxySQLC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddProxySQL.TLS)
	AddProxySQLC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddProxySQL.TLSSkipVerify)
	AddProxySQLC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").
		Default("auto").
		EnumVar(&AddProxySQL.MetricsMode, metricsModes...)

	AddProxySQLC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(&AddProxySQL.DisableCollectors)

	addGlobalFlags(AddProxySQLC)
}
