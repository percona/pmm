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
	"strconv"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/alecthomas/units"
	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"
)

const (
	// MysqlQuerySourceSlowLog defines available source name for profiler.
	MysqlQuerySourceSlowLog = "slowlog"
	// MysqlQuerySourcePerfSchema defines available source name for profiler.
	MysqlQuerySourcePerfSchema = "perfschema"
	// MysqlQuerySourceNone defines available source name for profiler.
	MysqlQuerySourceNone = "none"
)

var addMySQLResultT = commands.ParseTemplate(`
MySQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}

{{ .TablestatStatus }}
`)

type addMySQLResult struct {
	Service        *mysql.AddMySQLOKBodyService        `json:"service"`
	MysqldExporter *mysql.AddMySQLOKBodyMysqldExporter `json:"mysqld_exporter,omitempty"`
	TableCount     int32                               `json:"table_count,omitempty"`
}

func (res *addMySQLResult) Result() {}

func (res *addMySQLResult) String() string {
	return commands.RenderTemplate(addMySQLResultT, res)
}

func (res *addMySQLResult) TablestatStatus() string {
	if res.MysqldExporter == nil {
		return ""
	}

	status := "enabled"
	if res.MysqldExporter.TablestatsGroupDisabled {
		status = "disabled"
	}

	s := "Table statistics collection " + status

	switch {
	case res.MysqldExporter.TablestatsGroupTableLimit == 0: // server defined
		s += " (the table count limit is not set)."
	case res.MysqldExporter.TablestatsGroupTableLimit < 0: // always disabled
		s += " (always)."
	default:
		count := "unknown"
		if res.TableCount > 0 {
			count = strconv.Itoa(int(res.TableCount))
		}

		s += fmt.Sprintf(" (the limit is %d, the actual table count is %s).", res.MysqldExporter.TablestatsGroupTableLimit, count)
	}

	return s
}

// AddMySQLCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddMySQLCommand struct {
	ServiceName   string `name:"name" arg:"" default:"${hostname}-mysql" help:"Service name (autodetected default: ${hostname}-mysql)"`
	Address       string `arg:"" optional:"" help:"MySQL address and port (default: 127.0.0.1:3306)"`
	Socket        string `help:"Path to MySQL socket"`
	NodeID        string `help:"Node ID (default is autodetected)"`
	PMMAgentID    string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username      string `default:"root" help:"MySQL username"`
	Password      string `help:"MySQL password"`
	AgentPassword string `help:"Custom password for /metrics endpoint"`
	// TODO add "auto", make it default
	QuerySource            string            `default:"${mysqlQuerySourceDefault}" enum:"${mysqlQuerySourcesEnum}" help:"Source of SQL queries, one of: ${mysqlQuerySourcesEnum} (default: ${mysqlQuerySourceDefault})"`
	MaxQueryLength         int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples   bool              `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MaxSlowlogFileSize     units.Base2Bytes  `name:"size-slow-logs" placeholder:"size" help:"Rotate slow log file at this size (default: server-defined; negative value disables rotation). Ex.: 1GiB"`
	DisableTablestats      bool              `help:"Disable table statistics collection"`
	DisableTablestatsLimit uint16            `placeholder:"NUMBER" help:"Table statistics collection will be disabled if there are more than specified number of tables (default: server-defined)"`
	Environment            string            `help:"Environment name"`
	Cluster                string            `help:"Cluster name"`
	ReplicationSet         string            `help:"Replication set name"`
	CustomLabels           map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool              `help:"Skip connection check"`
	CommentsParsing        string            `enum:"on,off" default:"off" help:"Enable/disable parsing comments from queries. One of: [on, off]"`
	TLS                    bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify          bool              `help:"Skip TLS certificates validation"`
	TLSCaFile              string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile            string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile             string            `name:"tls-key" help:"Path to client key file"`
	CreateUser             bool              `hidden:"" help:"Create pmm user"`
	MetricsMode            string            `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"`
	DisableCollectors      []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	ExposeExporter         bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`

	AddCommonFlags
	AddLogLevelNoFatalFlags
}

// GetServiceName returns the service name for AddMySQLCommand.
func (cmd *AddMySQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

// GetAddress returns the address for AddMySQLCommand.
func (cmd *AddMySQLCommand) GetAddress() string {
	return cmd.Address
}

// GetDefaultAddress returns the default address for AddMySQLCommand.
func (cmd *AddMySQLCommand) GetDefaultAddress() string {
	return "127.0.0.1:3306"
}

// GetSocket returns the socket for AddMySQLCommand.
func (cmd *AddMySQLCommand) GetSocket() string {
	return cmd.Socket
}

// RunCmd runs the command for AddMySQLCommand.
func (cmd *AddMySQLCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	if cmd.CreateUser {
		return nil, errors.New("Unrecognized option. To create a user, see " +
			"'https://docs.percona.com/percona-monitoring-and-management/setting-up/client/mysql.html#create-a-database-account-for-pmm'")
	}

	var (
		err                    error
		tlsCa, tlsCert, tlsKey string
	)
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCaFile)
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

	tablestatsGroupTableLimit := int32(cmd.DisableTablestatsLimit)
	if cmd.DisableTablestats {
		if tablestatsGroupTableLimit != 0 {
			return nil, errors.Errorf("both --disable-tablestats and --disable-tablestats-limit are passed")
		}

		tablestatsGroupTableLimit = -1
	}

	params := &mysql.AddMySQLParams{
		Body: mysql.AddMySQLBody{
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
			CustomLabels:   customLabels,

			QANMysqlSlowlog:    cmd.QuerySource == MysqlQuerySourceSlowLog,
			QANMysqlPerfschema: cmd.QuerySource == MysqlQuerySourcePerfSchema,

			SkipConnectionCheck:    cmd.SkipConnectionCheck,
			DisableCommentsParsing: disableCommentsParsing,
			MaxQueryLength:         cmd.MaxQueryLength,
			DisableQueryExamples:   cmd.DisableQueryExamples,

			MaxSlowlogFileSize:        strconv.FormatInt(int64(cmd.MaxSlowlogFileSize), 10),
			TLS:                       cmd.TLS,
			TLSSkipVerify:             cmd.TLSSkipVerify,
			TLSCa:                     tlsCa,
			TLSCert:                   tlsCert,
			TLSKey:                    tlsKey,
			TablestatsGroupTableLimit: tablestatsGroupTableLimit,
			MetricsMode:               pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:         commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:                  &cmd.AddLogLevel,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.MySQL.AddMySQL(params)
	if err != nil {
		return nil, err
	}

	return &addMySQLResult{
		Service:        resp.Payload.Service,
		MysqldExporter: resp.Payload.MysqldExporter,
		TableCount:     resp.Payload.TableCount,
	}, nil
}
