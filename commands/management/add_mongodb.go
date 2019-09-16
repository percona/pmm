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
	"strings"

	"github.com/percona/pmm/api/managementpb/json/client"
	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"

	"github.com/percona/pmm-admin/agentlocal"
	"github.com/percona/pmm-admin/commands"
)

var addMongoDBResultT = commands.ParseTemplate(`
MongoDB Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addMongoDBResult struct {
	Service *mongodb.AddMongoDBOKBodyService `json:"service"`
}

func (res *addMongoDBResult) Result() {}

func (res *addMongoDBResult) String() string {
	return commands.RenderTemplate(addMongoDBResultT, res)
}

type addMongoDBCommand struct {
	AddressPort    string
	NodeID         string
	PMMAgentID     string
	ServiceName    string
	Username       string
	Password       string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string

	QuerySource string
	UseProfiler bool // TODO remove it https://jira.percona.com/browse/PMM-4704

	SkipConnectionCheck bool
	TLS                 bool
	TLSSkipVerify       bool
}

func (cmd *addMongoDBCommand) Run() (commands.Result, error) {
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

	host, portS, err := net.SplitHostPort(cmd.AddressPort)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}

	// ignore query source if old flags are present for compatibility
	useProfiler := cmd.UseProfiler
	if !useProfiler {
		switch cmd.QuerySource {
		case "profiler":
			useProfiler = true
		case "none":
			// nothing
		}
	}

	params := &mongodb.AddMongoDBParams{
		Body: mongodb.AddMongoDBBody{
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

			QANMongodbProfiler: useProfiler,

			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.MongoDB.AddMongoDB(params)
	if err != nil {
		return nil, err
	}

	return &addMongoDBResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command
var (
	AddMongoDB  = new(addMongoDBCommand)
	AddMongoDBC = AddC.Command("mongodb", "Add MongoDB to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	serviceName := hostname + "-mongodb"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddMongoDBC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddMongoDB.ServiceName)

	AddMongoDBC.Arg("address", "MongoDB address and port (default: 127.0.0.1:27017)").Default("127.0.0.1:27017").StringVar(&AddMongoDB.AddressPort)

	AddMongoDBC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddMongoDB.NodeID)
	AddMongoDBC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddMongoDB.PMMAgentID)

	AddMongoDBC.Flag("username", "MongoDB username").StringVar(&AddMongoDB.Username)
	AddMongoDBC.Flag("password", "MongoDB password").StringVar(&AddMongoDB.Password)

	querySources := []string{"profiler", "none"} // TODO add "auto"
	querySourceHelp := fmt.Sprintf("Source of queries, one of: %s (default: %s)", strings.Join(querySources, ", "), querySources[0])
	AddMongoDBC.Flag("query-source", querySourceHelp).Default(querySources[0]).EnumVar(&AddMongoDB.QuerySource, querySources...)
	AddMongoDBC.Flag("use-profiler", "Run QAN profiler agent").Hidden().BoolVar(&AddMongoDB.UseProfiler)

	AddMongoDBC.Flag("environment", "Environment name").StringVar(&AddMongoDB.Environment)
	AddMongoDBC.Flag("cluster", "Cluster name").StringVar(&AddMongoDB.Cluster)
	AddMongoDBC.Flag("replication-set", "Replication set name").StringVar(&AddMongoDB.ReplicationSet)
	AddMongoDBC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddMongoDB.CustomLabels)

	AddMongoDBC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddMongoDB.SkipConnectionCheck)
	AddMongoDBC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddMongoDB.TLS)
	AddMongoDBC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddMongoDB.TLSSkipVerify)
}
