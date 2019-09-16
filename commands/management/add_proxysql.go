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
	"net"
	"os"
	"strconv"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/managementpb/json/client"
	proxysql "github.com/percona/pmm/api/managementpb/json/client/proxy_sql"

	"github.com/percona/pmm-admin/agentlocal"
	"github.com/percona/pmm-admin/commands"
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
	AddressPort    string
	NodeID         string
	NodeName       string
	PMMAgentID     string
	ServiceName    string
	Username       string
	Password       string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string

	AddNode       bool
	AddNodeParams addNodeParams

	SkipConnectionCheck bool
	TLS                 bool
	TLSSkipVerify       bool
}

func (cmd *addProxySQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.PMMAgentID == "" || (cmd.NodeID == "" && cmd.NodeName == "") {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.PMMAgentID == "" {
			cmd.PMMAgentID = status.AgentID
		}
		if cmd.NodeID == "" && cmd.NodeName == "" {
			cmd.NodeID = status.NodeID
		}
	}

	host, portS, err := net.SplitHostPort(cmd.AddressPort)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}

	params := &proxysql.AddProxySQLParams{
		Body: proxysql.AddProxySQLBody{
			NodeID:         cmd.NodeID,
			ServiceName:    cmd.ServiceName,
			Address:        host,
			Port:           int64(port),
			PMMAgentID:     cmd.PMMAgentID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			Username:       cmd.Username,
			Password:       cmd.Password,

			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
		},
		Context: commands.Ctx,
	}
	if cmd.NodeName != "" {
		if cmd.AddNode {
			nodeCustomLabels, err := commands.ParseCustomLabels(cmd.AddNodeParams.CustomLabels)
			if err != nil {
				return nil, err
			}
			params.Body.AddNode = &proxysql.AddProxySQLParamsBodyAddNode{
				Az:            cmd.AddNodeParams.Az,
				ContainerID:   cmd.AddNodeParams.ContainerID,
				ContainerName: cmd.AddNodeParams.ContainerName,
				CustomLabels:  nodeCustomLabels,
				Distro:        cmd.AddNodeParams.Distro,
				MachineID:     cmd.AddNodeParams.MachineID,
				NodeModel:     cmd.AddNodeParams.NodeModel,
				NodeName:      cmd.NodeName,
				NodeType:      pointer.ToString(allNodeTypes[cmd.AddNodeParams.NodeType]),
				Region:        cmd.AddNodeParams.Region,
			}
		} else {
			params.Body.NodeName = cmd.NodeName
		}
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
	AddProxySQL  = new(addProxySQLCommand)
	AddProxySQLC = AddC.Command("proxysql", "Add ProxySQL to monitoring")
)

func init() {
	AddProxySQLC.Arg("address", "ProxySQL address and port (default: 127.0.0.1:3306)").Default("127.0.0.1:6032").StringVar(&AddProxySQL.AddressPort)

	hostname, _ := os.Hostname()
	serviceName := hostname + "-proxysql"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddProxySQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddProxySQL.ServiceName)

	AddProxySQLC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddProxySQL.NodeID)
	AddProxySQLC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddProxySQL.PMMAgentID)

	AddProxySQLC.Flag("username", "ProxySQL username").Default("admin").StringVar(&AddProxySQL.Username)
	AddProxySQLC.Flag("password", "ProxySQL password").Default("admin").StringVar(&AddProxySQL.Password)

	AddProxySQLC.Flag("environment", "Environment name").StringVar(&AddProxySQL.Environment)
	AddProxySQLC.Flag("cluster", "Cluster name").StringVar(&AddProxySQL.Cluster)
	AddProxySQLC.Flag("replication-set", "Replication set name").StringVar(&AddProxySQL.ReplicationSet)
	AddProxySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddProxySQL.CustomLabels)

	AddProxySQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddProxySQL.SkipConnectionCheck)
	AddProxySQLC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddProxySQL.TLS)
	AddProxySQLC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddProxySQL.TLSSkipVerify)

	AddProxySQLC.Flag("add-node", "Add new node").BoolVar(&AddProxySQL.AddNode)
	AddProxySQLC.Flag("node-name", "Node name").StringVar(&AddProxySQL.NodeName)
	addNodeFlags(AddProxySQLC, &AddProxySQL.AddNodeParams)
}
