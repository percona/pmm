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
	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"
)

const (
	mongodbQuerySourceProfiler = "profiler"
	mongodbQuerySourceNone     = "none"
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

	QuerySource string

	SkipConnectionCheck           bool
	TLS                           bool
	TLSSkipVerify                 bool
	TLSCertificateKeyFile         string
	TLSCertificateKeyFilePassword string
	TLSCaFile                     string
	AuthenticationMechanism       string
	AuthenticationDatabase        string

	EnableAllCollectors bool
	StatsCollections    string
	CollectionsLimit    int32
}

func (cmd *addMongoDBCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *addMongoDBCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *addMongoDBCommand) GetDefaultAddress() string {
	if cmd.CredentialsSource != "" {
		// address might be specified in credentials source file
		return ""
	}

	return "127.0.0.1:27017"
}

func (cmd *addMongoDBCommand) GetSocket() string {
	return cmd.Socket
}

func (cmd *addMongoDBCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	tlsCertificateKey, err := commands.ReadFile(cmd.TLSCertificateKeyFile)
	if err != nil {
		return nil, err
	}
	tlsCa, err := commands.ReadFile(cmd.TLSCaFile)
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

	params := &mongodb.AddMongoDBParams{
		Body: mongodb.AddMongoDBBody{
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

			QANMongodbProfiler: cmd.QuerySource == mongodbQuerySourceProfiler,

			CustomLabels:                  customLabels,
			SkipConnectionCheck:           cmd.SkipConnectionCheck,
			TLS:                           cmd.TLS,
			TLSSkipVerify:                 cmd.TLSSkipVerify,
			TLSCertificateKey:             tlsCertificateKey,
			TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
			TLSCa:                         tlsCa,
			AuthenticationMechanism:       cmd.AuthenticationMechanism,
			AuthenticationDatabase:        cmd.AuthenticationDatabase,

			MetricsMode: pointer.ToString(strings.ToUpper(cmd.MetricsMode)),

			EnableAllCollectors: cmd.EnableAllCollectors,
			DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
			StatsCollections:    commands.ParseDisableCollectors(cmd.StatsCollections),
			CollectionsLimit:    cmd.CollectionsLimit,
			LogLevel:            &addLogLevel,
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
	AddMongoDB  addMongoDBCommand
	AddMongoDBC = AddC.Command("mongodb", "Add MongoDB to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	serviceName := hostname + "-mongodb"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddMongoDBC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddMongoDB.ServiceName)

	AddMongoDBC.Arg("address", "MongoDB address and port (default: 127.0.0.1:27017)").StringVar(&AddMongoDB.Address)

	AddMongoDBC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddMongoDB.NodeID)
	AddMongoDBC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddMongoDB.PMMAgentID)

	AddMongoDBC.Flag("username", "MongoDB username").StringVar(&AddMongoDB.Username)
	AddMongoDBC.Flag("password", "MongoDB password").StringVar(&AddMongoDB.Password)
	AddMongoDBC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddMongoDB.AgentPassword)
	AddMongoDBC.Flag("credentials-source", "Credentials provider").StringVar(&AddMongoDB.CredentialsSource)

	querySources := []string{mongodbQuerySourceProfiler, mongodbQuerySourceNone} // TODO add "auto"
	querySourceHelp := fmt.Sprintf("Source of queries, one of: %s (default: %s)", strings.Join(querySources, ", "), querySources[0])
	AddMongoDBC.Flag("query-source", querySourceHelp).Default(querySources[0]).EnumVar(&AddMongoDB.QuerySource, querySources...)

	AddMongoDBC.Flag("environment", "Environment name").StringVar(&AddMongoDB.Environment)
	AddMongoDBC.Flag("cluster", "Cluster name").StringVar(&AddMongoDB.Cluster)
	AddMongoDBC.Flag("replication-set", "Replication set name").StringVar(&AddMongoDB.ReplicationSet)
	AddMongoDBC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddMongoDB.CustomLabels)

	AddMongoDBC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddMongoDB.SkipConnectionCheck)
	AddMongoDBC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddMongoDB.TLS)
	AddMongoDBC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddMongoDB.TLSSkipVerify)
	AddMongoDBC.Flag("tls-certificate-key-file", "Path to TLS certificate PEM file").StringVar(&AddMongoDB.TLSCertificateKeyFile)
	AddMongoDBC.Flag("tls-certificate-key-file-password", "Password for certificate").StringVar(&AddMongoDB.TLSCertificateKeyFilePassword)
	AddMongoDBC.Flag("tls-ca-file", "Path to certificate authority file").StringVar(&AddMongoDB.TLSCaFile)
	AddMongoDBC.Flag("authentication-mechanism", "Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates").
		StringVar(&AddMongoDB.AuthenticationMechanism)
	AddMongoDBC.Flag("authentication-database", "Authentication database. Default is empty. Use $external for ssl certificates").
		StringVar(&AddMongoDB.AuthenticationDatabase)

	AddMongoDBC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").
		Default("auto").
		EnumVar(&AddMongoDB.MetricsMode, metricsModes...)
	AddMongoDBC.Flag("enable-all-collectors", "Enable all collectors").BoolVar(&AddMongoDB.EnableAllCollectors)
	AddMongoDBC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(&AddMongoDB.DisableCollectors)
	addGlobalFlags(AddMongoDBC, true)
	AddMongoDBC.Flag("socket", "Path to socket").StringVar(&AddMongoDB.Socket)

	AddMongoDBC.Flag("stats-collections", "Collections for collstats & indexstats").StringVar(&AddMongoDB.StatsCollections)
	AddMongoDBC.Flag("max-collections-limit",
		"Disable collstats, dbstats, topmetrics and indexstats if there are more than <n> collections. 0: No limit. Default is -1, which let PMM automatically set this value.").
		Default("-1").Int32Var(&AddMongoDB.CollectionsLimit)
}
