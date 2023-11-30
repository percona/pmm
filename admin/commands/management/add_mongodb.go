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
	"github.com/percona/pmm/api/managementpb/json/client"
	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"
)

const (
	// MongodbQuerySourceProfiler defines available source name for profiler.
	MongodbQuerySourceProfiler = "profiler"
	// MongodbQuerySourceNone defines available source name for profiler.
	MongodbQuerySourceNone = "none"
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

// AddMongoDBCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddMongoDBCommand struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-mongodb" help:"Service name (autodetected default: ${hostname}-mongodb)"`
	Address           string `arg:"" optional:"" help:"MongoDB address and port (default: 127.0.0.1:27017)"`
	Socket            string `help:"Path to socket"`
	NodeID            string `help:"Node ID (default is autodetected)"`
	PMMAgentID        string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username          string `help:"MongoDB username"`
	Password          string `help:"MongoDB password"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	CredentialsSource string `type:"existingfile" help:"Credentials provider"`
	// TODO add "auto"
	QuerySource                   string            `default:"${mongoDbQuerySourceDefault}" enum:"${mongoDbQuerySourcesEnum}" help:"Source of queries, one of: ${mongoDbQuerySourcesEnum} (default: ${mongoDbQuerySourceDefault})"`
	Environment                   string            `help:"Environment name"`
	Cluster                       string            `help:"Cluster name"`
	ReplicationSet                string            `help:"Replication set name"`
	CustomLabels                  map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool              `help:"Skip connection check"`
	MaxQueryLength                int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	TLS                           bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool              `help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string            `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string            `help:"Password for certificate"`
	TLSCaFile                     string            `help:"Path to certificate authority file"`
	AuthenticationMechanism       string            `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	AuthenticationDatabase        string            `help:"Authentication database. Default is empty. Use $external for ssl certificates"`
	MetricsMode                   string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	EnableAllCollectors           bool              `help:"Enable all collectors"`
	DisableCollectors             []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	StatsCollections              []string          `help:"Collections for collstats & indexstats"`
	CollectionsLimit              int32             `name:"max-collections-limit" default:"-1" help:"Disable collstats, dbstats, topmetrics and indexstats if there are more than <n> collections. 0: No limit. Default is -1, which let PMM automatically set this value"`
	ExposeExporter                bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`

	AddCommonFlags
	AddLogLevelFatalFlags
}

func (cmd *AddMongoDBCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *AddMongoDBCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *AddMongoDBCommand) GetDefaultAddress() string {
	return "127.0.0.1:27017"
}

func (cmd *AddMongoDBCommand) GetSocket() string {
	return cmd.Socket
}

func (cmd *AddMongoDBCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.AgentPassword = creds.AgentPassword
	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

func (cmd *AddMongoDBCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

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

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)
	if err != nil {
		return nil, err
	}

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &mongodb.AddMongoDBParams{
		Body: mongodb.AddMongoDBBody{
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

			QANMongodbProfiler: cmd.QuerySource == MongodbQuerySourceProfiler,

			CustomLabels:                  customLabels,
			SkipConnectionCheck:           cmd.SkipConnectionCheck,
			MaxQueryLength:                cmd.MaxQueryLength,
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
			LogLevel:            &cmd.AddLogLevel,
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
