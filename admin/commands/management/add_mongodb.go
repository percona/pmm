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
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"
)

const (
	MongodbQuerySourceProfiler = "profiler"
	MongodbQuerySourceNone     = "none"
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
			Port:           int64(port),
			Socket:         socket,
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
