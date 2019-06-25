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

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/reform.v1"
)

//go:generate reform

// AgentType represents Agent type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and Prometheus.
type AgentType string

// Agent types (in the same order as in agents.proto).
const (
	PMMAgentType                       AgentType = "pmm-agent"
	NodeExporterType                   AgentType = "node_exporter"
	MySQLdExporterType                 AgentType = "mysqld_exporter"
	MongoDBExporterType                AgentType = "mongodb_exporter"
	PostgresExporterType               AgentType = "postgres_exporter"
	QANMySQLPerfSchemaAgentType        AgentType = "qan-mysql-perfschema-agent"
	QANMySQLSlowlogAgentType           AgentType = "qan-mysql-slowlog-agent"
	QANMongoDBProfilerAgentType        AgentType = "qan-mongodb-profiler-agent"
	QANPostgreSQLPgStatementsAgentType AgentType = "qan-postgresql-pgstatements-agent"
	ProxySQLExporterType               AgentType = "proxysql_exporter"
)

// PMMServerAgentID is a special Agent ID representing pmm-agent on PMM Server.
const PMMServerAgentID string = "pmm-server" // no /agent_id/ prefix

// Agent represents Agent as stored in database.
//reform:agents
type Agent struct {
	AgentID      string    `reform:"agent_id,pk"`
	AgentType    AgentType `reform:"agent_type"`
	RunsOnNodeID *string   `reform:"runs_on_node_id"`
	PMMAgentID   *string   `reform:"pmm_agent_id"`
	CustomLabels []byte    `reform:"custom_labels"`
	CreatedAt    time.Time `reform:"created_at"`
	UpdatedAt    time.Time `reform:"updated_at"`

	Disabled   bool    `reform:"disabled"`
	Status     string  `reform:"status"`
	ListenPort *uint16 `reform:"listen_port"`
	Version    *string `reform:"version"`

	Username   *string `reform:"username"`
	Password   *string `reform:"password"`
	MetricsURL *string `reform:"metrics_url"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *Agent) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *Agent) BeforeUpdate() error {
	s.UpdatedAt = Now()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *Agent) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// GetCustomLabels decodes custom labels.
func (s *Agent) GetCustomLabels() (map[string]string, error) {
	return getCustomLabels(s.CustomLabels)
}

// SetCustomLabels encodes custom labels.
func (s *Agent) SetCustomLabels(m map[string]string) error {
	return setCustomLabels(m, &s.CustomLabels)
}

// UnifiedLabels returns combined standard and custom labels with empty labels removed.
func (s *Agent) UnifiedLabels() (map[string]string, error) {
	custom, err := s.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	res := map[string]string{
		"agent_id":   s.AgentID,
		"agent_type": string(s.AgentType),
	}
	for name, value := range custom {
		res[name] = value
	}

	if err = prepareLabels(res, true); err != nil {
		return nil, err
	}
	return res, nil
}

// DSN returns DSN string for accessing given Service with this Agent (and implicit driver).
func (s *Agent) DSN(service *Service, dialTimeout time.Duration, database string) string {
	host := pointer.GetString(service.Address)
	port := pointer.GetUint16(service.Port)
	username := pointer.GetString(s.Username)
	password := pointer.GetString(s.Password)

	switch s.AgentType {
	case MySQLdExporterType, ProxySQLExporterType:
		// TODO TLSConfig: "true", https://jira.percona.com/browse/PMM-1727

		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		cfg.Timeout = dialTimeout
		cfg.DBName = database

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		return cfg.FormatDSN()

	case QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		// TODO TLSConfig: "true", https://jira.percona.com/browse/PMM-1727

		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		cfg.Timeout = dialTimeout
		cfg.DBName = database

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		// QAN code in pmm-agent uses reform which requires those fields
		cfg.ClientFoundRows = true
		cfg.ParseTime = true

		return cfg.FormatDSN()

	case MongoDBExporterType:
		// TODO return MongoURI for the new driver
		fallthrough

	case QANMongoDBProfilerAgentType:
		u := &url.URL{
			Scheme: "mongodb",
			Host:   net.JoinHostPort(host, strconv.Itoa(int(port))),
		}
		switch {
		case password != "":
			u.User = url.UserPassword(username, password)
		case username != "":
			u.User = url.User(username)
		}
		return u.String()

	case PostgresExporterType, QANPostgreSQLPgStatementsAgentType:
		q := make(url.Values)
		q.Set("sslmode", "disable") // TODO: make it configurable
		if dialTimeout != 0 {
			q.Set("connect_timeout", strconv.Itoa(int(dialTimeout.Seconds())))
		}

		u := &url.URL{
			Scheme:   "postgres",
			Host:     net.JoinHostPort(host, strconv.Itoa(int(port))),
			Path:     database,
			RawQuery: q.Encode(),
		}
		switch {
		case password != "":
			u.User = url.UserPassword(username, password)
		case username != "":
			u.User = url.User(username)
		}
		return u.String()

	default:
		panic(fmt.Errorf("unhandled AgentType %q", s.AgentType))
	}
}

// check interfaces
var (
	_ reform.BeforeInserter = (*Agent)(nil)
	_ reform.BeforeUpdater  = (*Agent)(nil)
	_ reform.AfterFinder    = (*Agent)(nil)
)
