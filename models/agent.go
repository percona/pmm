// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package models

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:generate reform

const (
	// maximum time for connecting to the database
	sqlDialTimeout = 5 * time.Second
)

type AgentType string

// AgentType agent types for exporters and agents
const (
	MySQLdExporterAgentType   AgentType = "mysqld_exporter"
	PostgresExporterAgentType AgentType = "postgres_exporter"
	RDSExporterAgentType      AgentType = "rds_exporter"
	QanAgentAgentType         AgentType = "qan-agent"
)

// NameForSupervisor returns a name of agent for supervisor.
func NameForSupervisor(typ AgentType, listenPort uint16) string {
	return fmt.Sprintf("pmm-%s-%d", typ, listenPort)
}

//reform:agents
type Agent struct {
	ID           int32     `reform:"id,pk"`
	Type         AgentType `reform:"type"`
	RunsOnNodeID int32     `reform:"runs_on_node_id"`

	// TODO Does it really belong there? Remove when we have agent without one.
	ListenPort *uint16 `reform:"listen_port"`
}

//reform:agents
type MySQLdExporter struct {
	ID           int32     `reform:"id,pk"`
	Type         AgentType `reform:"type"`
	RunsOnNodeID int32     `reform:"runs_on_node_id"`

	ServiceUsername        *string `reform:"service_username"`
	ServicePassword        *string `reform:"service_password"`
	ListenPort             *uint16 `reform:"listen_port"`
	MySQLDisableTablestats *bool   `reform:"mysql_disable_tablestats"`
}

func (m *MySQLdExporter) DSN(service *MySQLService) string {
	cfg := mysql.NewConfig()
	cfg.User = *m.ServiceUsername
	cfg.Passwd = *m.ServicePassword

	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(*service.Address, strconv.Itoa(int(*service.Port)))

	cfg.Timeout = sqlDialTimeout

	// TODO TLSConfig: "true", https://jira.percona.com/browse/PMM-1727
	// TODO Other parameters?
	return cfg.FormatDSN()
}

// binary name is postgres_exporter, that's why PostgresExporter below is not PostgreSQLExporter

//reform:agents
// PostgresExporter exports PostgreSQL metrics.
type PostgresExporter struct {
	ID           int32     `reform:"id,pk"`
	Type         AgentType `reform:"type"`
	RunsOnNodeID int32     `reform:"runs_on_node_id"`

	ServiceUsername *string `reform:"service_username"`
	ServicePassword *string `reform:"service_password"`
	ListenPort      *uint16 `reform:"listen_port"`
}

// DSN returns DSN for PostgreSQL service.
func (p *PostgresExporter) DSN(service *PostgreSQLService) string {
	q := make(url.Values)
	q.Set("sslmode", "disable") // TODO https://jira.percona.com/browse/PMM-1727
	q.Set("connect_timeout", strconv.Itoa(int(sqlDialTimeout.Seconds())))

	address := net.JoinHostPort(*service.Address, strconv.Itoa(int(*service.Port)))
	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(*p.ServiceUsername, *p.ServicePassword),
		Host:     address,
		Path:     "postgres",
		RawQuery: q.Encode(),
	}
	return uri.String()
}

//reform:agents
type RDSExporter struct {
	ID           int32     `reform:"id,pk"`
	Type         AgentType `reform:"type"`
	RunsOnNodeID int32     `reform:"runs_on_node_id"`

	ListenPort *uint16 `reform:"listen_port"`
}

//reform:agents
type QanAgent struct {
	ID           int32     `reform:"id,pk"`
	Type         AgentType `reform:"type"`
	RunsOnNodeID int32     `reform:"runs_on_node_id"`

	ServiceUsername   *string `reform:"service_username"`
	ServicePassword   *string `reform:"service_password"`
	ListenPort        *uint16 `reform:"listen_port"`
	QANDBInstanceUUID *string `reform:"qan_db_instance_uuid"` // MySQL instance UUID in QAN
}

func (q *QanAgent) DSN(service *MySQLService) string {
	cfg := mysql.NewConfig()
	cfg.User = *q.ServiceUsername
	cfg.Passwd = *q.ServicePassword

	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(*service.Address, strconv.Itoa(int(*service.Port)))

	cfg.Timeout = sqlDialTimeout

	// TODO TLSConfig: "true", https://jira.percona.com/browse/PMM-1727
	// TODO Other parameters?
	return cfg.FormatDSN()
}
