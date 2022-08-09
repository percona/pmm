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
	mysqlQuerySourceSlowLog    = "slowlog"
	mysqlQuerySourcePerfSchema = "perfschema"
	mysqlQuerySourceNone       = "none"
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
	case res.MysqldExporter.TablestatsGroupTableLimit == 0: // no limit
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

type addMySQLCommand struct {
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

	SkipConnectionCheck    bool
	DisableQueryExamples   bool
	MaxSlowlogFileSize     units.Base2Bytes
	TLS                    bool
	TLSSkipVerify          bool
	TLSCaFile              string
	TLSCertFile            string
	TLSKeyFile             string
	DisableTablestats      bool
	DisableTablestatsLimit uint16
	CreateUser             bool
}

func (cmd *addMySQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *addMySQLCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *addMySQLCommand) GetDefaultAddress() string {
	if cmd.CredentialsSource != "" {
		// address might be specified in credentials source file
		return ""
	}

	return "127.0.0.1:3306"
}

func (cmd *addMySQLCommand) GetSocket() string {
	return cmd.Socket
}

func (cmd *addMySQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.CreateUser {
		return nil, errors.New("Unrecognized option. To create a user, see " +
			"'https://www.percona.com/doc/percona-monitoring-and-management/2.x/concepts/services-mysql.html#pmm-conf-mysql-user-account-creating'")
	}

	var tlsCa, tlsCert, tlsKey string
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

	tablestatsGroupTableLimit := int32(cmd.DisableTablestatsLimit)
	if cmd.DisableTablestats {
		if tablestatsGroupTableLimit != 0 {
			return nil, errors.Errorf("both --disable-tablestats and --disable-tablestats-limit are passed")
		}

		tablestatsGroupTableLimit = -1
	}

	params := &mysql.AddMySQLParams{
		Body: mysql.AddMySQLBody{
			NodeID:            cmd.NodeID,
			ServiceName:       serviceName,
			Address:           host,
			Socket:            socket,
			Port:              int64(port),
			PMMAgentID:        cmd.PMMAgentID,
			Environment:       cmd.Environment,
			Cluster:           cmd.Cluster,
			ReplicationSet:    cmd.ReplicationSet,
			Username:          cmd.Username,
			Password:          cmd.Password,
			AgentPassword:     cmd.AgentPassword,
			CredentialsSource: cmd.CredentialsSource,
			CustomLabels:      customLabels,

			QANMysqlSlowlog:    cmd.QuerySource == mysqlQuerySourceSlowLog,
			QANMysqlPerfschema: cmd.QuerySource == mysqlQuerySourcePerfSchema,

			SkipConnectionCheck:       cmd.SkipConnectionCheck,
			DisableQueryExamples:      cmd.DisableQueryExamples,
			MaxSlowlogFileSize:        strconv.FormatInt(int64(cmd.MaxSlowlogFileSize), 10),
			TLS:                       cmd.TLS,
			TLSSkipVerify:             cmd.TLSSkipVerify,
			TLSCa:                     tlsCa,
			TLSCert:                   tlsCert,
			TLSKey:                    tlsKey,
			TablestatsGroupTableLimit: tablestatsGroupTableLimit,
			MetricsMode:               pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			DisableCollectors:         commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:                  &addLogLevel,
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

// register command
var (
	AddMySQL  addMySQLCommand
	AddMySQLC = AddC.Command("mysql", "Add MySQL to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	serviceName := hostname + "-mysql"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddMySQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddMySQL.ServiceName)

	AddMySQLC.Arg("address", "MySQL address and port (default: 127.0.0.1:3306)").StringVar(&AddMySQL.Address)
	AddMySQLC.Flag("socket", "Path to MySQL socket").StringVar(&AddMySQL.Socket)

	AddMySQLC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddMySQL.NodeID)
	AddMySQLC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddMySQL.PMMAgentID)

	AddMySQLC.Flag("username", "MySQL username").Default("root").StringVar(&AddMySQL.Username)
	AddMySQLC.Flag("password", "MySQL password").StringVar(&AddMySQL.Password)
	AddMySQLC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddMySQL.AgentPassword)
	AddMySQLC.Flag("credentials-source", "Credentials provider").StringVar(&AddMySQL.CredentialsSource)

	querySources := []string{mysqlQuerySourceSlowLog, mysqlQuerySourcePerfSchema, mysqlQuerySourceNone} // TODO add "auto", make it default
	querySourceHelp := fmt.Sprintf("Source of SQL queries, one of: %s (default: %s)", strings.Join(querySources, ", "), querySources[0])
	AddMySQLC.Flag("query-source", querySourceHelp).Default(querySources[0]).EnumVar(&AddMySQL.QuerySource, querySources...)
	AddMySQLC.Flag("disable-queryexamples", "Disable collection of query examples").BoolVar(&AddMySQL.DisableQueryExamples)
	AddMySQLC.Flag("size-slow-logs", `Rotate slow log file at this size (default: server-defined; negative value disables rotation). Ex.: 1GiB`).
		BytesVar(&AddMySQL.MaxSlowlogFileSize)
	AddMySQLC.Flag("disable-tablestats", "Disable table statistics collection").BoolVar(&AddMySQL.DisableTablestats)
	AddMySQLC.Flag("disable-tablestats-limit", "Table statistics collection will be disabled if there are more than specified number of tables (default: server-defined)").
		Uint16Var(&AddMySQL.DisableTablestatsLimit)

	AddMySQLC.Flag("environment", "Environment name").StringVar(&AddMySQL.Environment)
	AddMySQLC.Flag("cluster", "Cluster name").StringVar(&AddMySQL.Cluster)
	AddMySQLC.Flag("replication-set", "Replication set name").StringVar(&AddMySQL.ReplicationSet)
	AddMySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddMySQL.CustomLabels)

	AddMySQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddMySQL.SkipConnectionCheck)
	AddMySQLC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddMySQL.TLS)
	AddMySQLC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddMySQL.TLSSkipVerify)
	AddMySQLC.Flag("tls-ca", "Path to certificate authority certificate file").StringVar(&AddMySQL.TLSCaFile)
	AddMySQLC.Flag("tls-cert", "Path to client certificate file").StringVar(&AddMySQL.TLSCertFile)
	AddMySQLC.Flag("tls-key", "Path to client key file").StringVar(&AddMySQL.TLSKeyFile)
	AddMySQLC.Flag("create-user", "Create pmm user").Hidden().BoolVar(&AddMySQL.CreateUser)
	AddMySQLC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").
		Default("auto").
		EnumVar(&AddMySQL.MetricsMode, metricsModes...)
	AddMySQLC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(&AddMySQL.DisableCollectors)
	addGlobalFlags(AddMySQLC)
}
