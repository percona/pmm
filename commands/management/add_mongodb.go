// pmm-admin
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package management

import (
	"fmt"
	"net"
	"os"
	"strconv"

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
	AddressPort string
	ServiceName string
	Username    string
	Password    string
	UseExporter bool
	UseProfiler bool
}

func (cmd *addMongoDBCommand) Run() (commands.Result, error) {
	status, err := agentlocal.GetStatus()
	if err != nil {
		return nil, err
	}

	host, portS, err := net.SplitHostPort(cmd.AddressPort)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}

	params := &mongodb.AddMongoDBParams{
		Body: mongodb.AddMongoDBBody{
			PMMAgentID:  status.AgentID,
			NodeID:      status.NodeID,
			ServiceName: cmd.ServiceName,
			Address:     host,
			Port:        int64(port),

			MongodbExporter: cmd.UseExporter,
			Username:        cmd.Username,
			Password:        cmd.Password,

			QANMongodbProfiler: cmd.UseProfiler,
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
	AddMongoDBC = AddC.Command("mongodb", "Add MongoDB to monitoring.")
)

func init() {
	AddMongoDBC.Arg("address", "MongoDB address and port. Default: 127.0.0.1:27017.").Default("127.0.0.1:27017").StringVar(&AddMongoDB.AddressPort)

	hostname, _ := os.Hostname()
	serviceName := hostname + "-mongodb"
	serviceNameHelp := fmt.Sprintf("Service name. Default: %s.", serviceName)
	AddMongoDBC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddMongoDB.ServiceName)

	AddMongoDBC.Flag("username", "MongoDB username.").StringVar(&AddMongoDB.Username)
	AddMongoDBC.Flag("password", "MongoDB password.").StringVar(&AddMongoDB.Password)
	AddMongoDBC.Flag("use-profiler", "Run QAN profiler agent.").BoolVar(&AddMongoDB.UseProfiler)
	AddMongoDBC.Flag("use-exporter", "Run mongodb_exporter.").BoolVar(&AddMongoDB.UseExporter)
}
