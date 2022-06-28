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
	"strconv"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"
)

const (
	MysqlQuerySourceSlowLog    = "slowlog"
	MysqlQuerySourcePerfSchema = "perfschema"
	MysqlQuerySourceNone       = "none"
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

func (cmd *AddMySQLCmd) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *AddMySQLCmd) GetAddress() string {
	return cmd.Address
}

func (cmd *AddMySQLCmd) GetDefaultAddress() string {
	if cmd.DefaultsFile != "" {
		// address might be specified in defaults file
		return ""
	}
	return "127.0.0.1:3306"
}

func (cmd *AddMySQLCmd) GetDefaultUsername() string {
	return "root"
}

func (cmd *AddMySQLCmd) GetSocket() string {
	return cmd.Socket
}

func (cmd *AddMySQLCmd) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.AgentPassword = creds.AgentPassword
	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

func (cmd *AddMySQLCmd) RunCmd() (commands.Result, error) {
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

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)
	if err != nil {
		return nil, err
	}

	username := defaultsFileUsernameCheck(cmd)

	tablestatsGroupTableLimit := int32(cmd.DisableTablestatsLimit)
	if cmd.DisableTablestats {
		if tablestatsGroupTableLimit != 0 {
			return nil, errors.Errorf("both --disable-tablestats and --disable-tablestats-limit are passed")
		}

		tablestatsGroupTableLimit = -1
	}

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve credentials from %s", cmd.CredentialsSource)
		}
	}

	params := &mysql.AddMySQLParams{
		Body: mysql.AddMySQLBody{
			NodeID:         cmd.NodeID,
			ServiceName:    serviceName,
			Address:        host,
			Socket:         socket,
			Port:           int64(port),
			PMMAgentID:     cmd.PMMAgentID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			Username:       username,
			Password:       cmd.Password,
			AgentPassword:  cmd.AgentPassword,
			CustomLabels:   customLabels,

			QANMysqlSlowlog:    cmd.QuerySource == MysqlQuerySourceSlowLog,
			QANMysqlPerfschema: cmd.QuerySource == MysqlQuerySourcePerfSchema,

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

func defaultsFileUsernameCheck(cmd *AddMySQLCmd) string {
	// passed username has higher priority over defaults file
	if cmd.Username != "" {
		return cmd.Username
	}

	// username not specified, but can be in defaults file
	if cmd.Username == "" && cmd.DefaultsFile != "" {
		return ""
	}

	return cmd.GetDefaultUsername()
}
