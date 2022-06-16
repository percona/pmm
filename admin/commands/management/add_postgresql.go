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
	postgresql "github.com/percona/pmm/api/managementpb/json/client/postgre_sql"
)

var addPostgreSQLResultT = commands.ParseTemplate(`
PostgreSQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addPostgreSQLResult struct {
	Service *postgresql.AddPostgreSQLOKBodyService `json:"service"`
}

func (res *addPostgreSQLResult) Result() {}

func (res *addPostgreSQLResult) String() string {
	return commands.RenderTemplate(addPostgreSQLResultT, res)
}

func (cmd *AddPostgreSqlCmd) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *AddPostgreSqlCmd) GetAddress() string {
	return cmd.Address
}

func (cmd *AddPostgreSqlCmd) GetDefaultAddress() string {
	return "127.0.0.1:5432"
}

func (cmd *AddPostgreSqlCmd) GetSocket() string {
	return cmd.Socket
}

func (cmd *AddPostgreSqlCmd) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.AgentPassword = creds.AgentPassword
	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

func (cmd *AddPostgreSqlCmd) RunCmd() (commands.Result, error) {
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

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)
	if err != nil {
		return nil, err
	}

	var usePgStatements bool

	var usePgStatMonitor bool

	switch cmd.QuerySource {
	case "pgstatements":
		usePgStatements = true
	case "pgstatmonitor":
		usePgStatMonitor = true
	case "none":
	}

	var tlsCa, tlsCert, tlsKey string
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCAFile)
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

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &postgresql.AddPostgreSQLParams{
		Body: postgresql.AddPostgreSQLBody{
			NodeID:      cmd.NodeID,
			ServiceName: serviceName,

			Address:             host,
			Port:                int64(port),
			Username:            cmd.Username,
			Password:            cmd.Password,
			Database:            cmd.Database,
			AgentPassword:       cmd.AgentPassword,
			Socket:              socket,
			SkipConnectionCheck: cmd.SkipConnectionCheck,

			PMMAgentID:     cmd.PMMAgentID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			CustomLabels:   customLabels,

			QANPostgresqlPgstatementsAgent:  usePgStatements,
			QANPostgresqlPgstatmonitorAgent: usePgStatMonitor,

			TLS:           cmd.TLS,
			TLSCa:         tlsCa,
			TLSCert:       tlsCert,
			TLSKey:        tlsKey,
			TLSSkipVerify: cmd.TLSSkipVerify,

			DisableQueryExamples: cmd.DisableQueryExamples,
			MetricsMode:          pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:    commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:             &cmd.AddLogLevel,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.PostgreSQL.AddPostgreSQL(params)
	if err != nil {
		return nil, err
	}

	return &addPostgreSQLResult{
		Service: resp.Payload.Service,
	}, nil
}
