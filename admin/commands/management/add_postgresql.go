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
	postgresql "github.com/percona/pmm/api/managementpb/json/client/postgre_sql"
)

var addPostgreSQLResultT = commands.ParseTemplate(`
PostgreSQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
{{ if .Warning }}
Warning: {{ .Warning }}
{{- end -}}
`)

type addPostgreSQLResult struct {
	Service *postgresql.AddPostgreSQLOKBodyService `json:"service"`
	Warning string                                 `json:"warning"`
}

func (res *addPostgreSQLResult) Result() {}

func (res *addPostgreSQLResult) String() string {
	return commands.RenderTemplate(addPostgreSQLResultT, res)
}

// AddPostgreSQLCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddPostgreSQLCommand struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-postgresql" help:"Service name (autodetected default: ${hostname}-postgresql)"`
	Address           string `arg:"" optional:"" help:"PostgreSQL address and port (default: 127.0.0.1:5432)"`
	Socket            string `help:"Path to socket"`
	Username          string `default:"postgres" help:"PostgreSQL username"`
	Password          string `help:"PostgreSQL password"`
	Database          string `help:"PostgreSQL database"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	CredentialsSource string `type:"existingfile" help:"Credentials provider"`
	NodeID            string `help:"Node ID (default is autodetected)"`
	PMMAgentID        string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	// TODO add "auto"
	QuerySource            string            `default:"pgstatmonitor" help:"Source of SQL queries, one of: pgstatements, pgstatmonitor, none (default: pgstatmonitor)"`
	Environment            string            `help:"Environment name"`
	Cluster                string            `help:"Cluster name"`
	ReplicationSet         string            `help:"Replication set name"`
	CustomLabels           map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool              `help:"Skip connection check"`
	CommentsParsing        string            `enum:"on,off" default:"off" help:"Enable/disable parsing comments from queries. One of: [on, off]"`
	TLS                    bool              `help:"Use TLS to connect to the database"`
	TLSCAFile              string            `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile            string            `help:"TLS certificate file"`
	TLSKeyFile             string            `help:"TLS certificate key file"`
	TLSSkipVerify          bool              `help:"Skip TLS certificates validation"`
	MaxQueryLength         int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples   bool              `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MetricsMode            string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	DisableCollectors      []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	ExposeExporter         bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`
	AutoDiscoveryLimit     int32             `placeholder:"NUMBER" help:"Auto-discovery will be disabled if there are more than that number of databases (default: server-defined, -1: always disabled)"`
	MaxExporterConnections int32             `placeholder:"NUMBER" help:"Maximum number of connections to PostgreSQL instance that exporter can use (default: server-defined)"`

	AddCommonFlags
	AddLogLevelNoFatalFlags
}

// GetServiceName returns the service name for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

// GetAddress returns the address for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) GetAddress() string {
	return cmd.Address
}

// GetDefaultAddress returns the default address for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) GetDefaultAddress() string {
	return "127.0.0.1:5432"
}

// GetSocket returns the socket for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) GetSocket() string {
	return cmd.Socket
}

// GetCredentials returns the credentials for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.AgentPassword = creds.AgentPassword
	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

// RunCmd runs the command for AddPostgreSQLCommand.
func (cmd *AddPostgreSQLCommand) RunCmd() (commands.Result, error) {
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

	disableCommentsParsing := true
	if cmd.CommentsParsing == "on" {
		disableCommentsParsing = false
	}

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &postgresql.AddPostgreSQLParams{
		Body: postgresql.AddPostgreSQLBody{
			NodeID:                 cmd.NodeID,
			ServiceName:            serviceName,
			Address:                host,
			Socket:                 socket,
			Port:                   int64(port),
			ExposeExporter:         cmd.ExposeExporter,
			Username:               cmd.Username,
			Password:               cmd.Password,
			Database:               cmd.Database,
			AgentPassword:          cmd.AgentPassword,
			SkipConnectionCheck:    cmd.SkipConnectionCheck,
			DisableCommentsParsing: disableCommentsParsing,

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

			MaxQueryLength:         cmd.MaxQueryLength,
			DisableQueryExamples:   cmd.DisableQueryExamples,
			MetricsMode:            pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:      commands.ParseDisableCollectors(cmd.DisableCollectors),
			AutoDiscoveryLimit:     cmd.AutoDiscoveryLimit,
			MaxExporterConnections: cmd.MaxExporterConnections,
			LogLevel:               &cmd.AddLogLevel,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.PostgreSQL.AddPostgreSQL(params)
	if err != nil {
		return nil, err
	}

	return &addPostgreSQLResult{
		Service: resp.Payload.Service,
		Warning: resp.Payload.Warning,
	}, nil
}
