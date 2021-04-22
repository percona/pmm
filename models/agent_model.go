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
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/percona/pmm/version"
	"gopkg.in/reform.v1"
)

//go:generate reform

// AgentType represents Agent type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and VictoriaMetrics.
type AgentType string

const (
	certificateKeyFilePlaceholder = "certificateKeyFilePlaceholder"
	caFilePlaceholder             = "caFilePlaceholder"
)

// Agent types (in the same order as in agents.proto).
const (
	PMMAgentType                        AgentType = "pmm-agent"
	NodeExporterType                    AgentType = "node_exporter"
	MySQLdExporterType                  AgentType = "mysqld_exporter"
	MongoDBExporterType                 AgentType = "mongodb_exporter"
	PostgresExporterType                AgentType = "postgres_exporter"
	ProxySQLExporterType                AgentType = "proxysql_exporter"
	RDSExporterType                     AgentType = "rds_exporter"
	AzureDatabaseExporterType           AgentType = "azure_database_exporter"
	QANMySQLPerfSchemaAgentType         AgentType = "qan-mysql-perfschema-agent"
	QANMySQLSlowlogAgentType            AgentType = "qan-mysql-slowlog-agent"
	QANMongoDBProfilerAgentType         AgentType = "qan-mongodb-profiler-agent"
	QANPostgreSQLPgStatementsAgentType  AgentType = "qan-postgresql-pgstatements-agent"
	QANPostgreSQLPgStatMonitorAgentType AgentType = "qan-postgresql-pgstatmonitor-agent"
	ExternalExporterType                AgentType = "external-exporter"
	VMAgentType                         AgentType = "vmagent"
)

// PMMServerAgentID is a special Agent ID representing pmm-agent on PMM Server.
const PMMServerAgentID string = "pmm-server" // no /agent_id/ prefix

// MongoDBOptions represents structure for special MongoDB options.
type MongoDBOptions struct {
	TLSCertificateKey             string `json:"tls_certificate_key"`
	TLSCertificateKeyFilePassword string `json:"tls_certificate_key_file_password"`
	TLSCa                         string `json:"tls_ca"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c MongoDBOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *MongoDBOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// AzureOptions represents structure for special Azure options.
type AzureOptions struct {
	SubscriptionID string `json:"subscription_id"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	TenantID       string `json:"tenant_id"`
	ResourceGroup  string `json:"resource_group"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c AzureOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *AzureOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// PMMAgentWithPushMetricsSupport - version of pmmAgent,
// that support vmagent and push metrics mode
// will be released with PMM Agent v2.12.
var PMMAgentWithPushMetricsSupport = version.MustParse("2.11.99")

// Agent represents Agent as stored in database.
//reform:agents
type Agent struct {
	AgentID      string    `reform:"agent_id,pk"`
	AgentType    AgentType `reform:"agent_type"`
	RunsOnNodeID *string   `reform:"runs_on_node_id"`
	ServiceID    *string   `reform:"service_id"`
	NodeID       *string   `reform:"node_id"`
	PMMAgentID   *string   `reform:"pmm_agent_id"`
	CustomLabels []byte    `reform:"custom_labels"`
	CreatedAt    time.Time `reform:"created_at"`
	UpdatedAt    time.Time `reform:"updated_at"`

	Disabled   bool    `reform:"disabled"`
	Status     string  `reform:"status"`
	ListenPort *uint16 `reform:"listen_port"`
	Version    *string `reform:"version"`

	Username      *string `reform:"username"`
	Password      *string `reform:"password"`
	TLS           bool    `reform:"tls"`
	TLSSkipVerify bool    `reform:"tls_skip_verify"`

	AWSAccessKey *string `reform:"aws_access_key"`
	AWSSecretKey *string `reform:"aws_secret_key"`

	AzureOptions *AzureOptions `reform:"azure_options"`

	// TableCount stores last known table count. NULL if unknown.
	TableCount *int32 `reform:"table_count"`

	// Tablestats group collectors are disabled if there are more than that number of tables.
	// 0 means tablestats group collectors are always enabled (no limit).
	// Negative value means tablestats group collectors are always disabled.
	// See IsMySQLTablestatsGroupEnabled method.
	TableCountTablestatsGroupLimit int32 `reform:"table_count_tablestats_group_limit"`

	QueryExamplesDisabled bool    `reform:"query_examples_disabled"`
	MaxQueryLogSize       int64   `reform:"max_query_log_size"`
	MetricsPath           *string `reform:"metrics_path"`
	MetricsScheme         *string `reform:"metrics_scheme"`

	RDSBasicMetricsDisabled    bool           `reform:"rds_basic_metrics_disabled"`
	RDSEnhancedMetricsDisabled bool           `reform:"rds_enhanced_metrics_disabled"`
	PushMetrics                bool           `reform:"push_metrics"`
	DisabledCollectors         pq.StringArray `reform:"disabled_collectors"`

	MongoDBOptions *MongoDBOptions `reform:"mongo_db_tls_options"`
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
	return getLabels(s.CustomLabels)
}

// SetCustomLabels encodes custom labels.
func (s *Agent) SetCustomLabels(m map[string]string) error {
	return setLabels(m, &s.CustomLabels)
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

// DBConfig contains values required to connect to DB.
type DBConfig struct {
	User     string
	Password string
	Address  string
	Port     int
	Socket   string
}

// Valid returns true if config is valid.
func (c DBConfig) Valid() bool {
	return c.User != "" && (c.Address != "" || c.Socket != "")
}

// DBConfig returns DBConfig for given Service with this agent.
func (s *Agent) DBConfig(service *Service) DBConfig {
	cfg := DBConfig{
		User:     pointer.GetString(s.Username),
		Password: pointer.GetString(s.Password),
		Address:  pointer.GetString(service.Address),
		Port:     int(pointer.GetUint16(service.Port)),
		Socket:   pointer.GetString(service.Socket),
	}
	return cfg
}

// DSN returns DSN string for accessing given Service with this Agent (and implicit driver).
func (s *Agent) DSN(service *Service, dialTimeout time.Duration, database string, tdp *DelimiterPair) string {
	host := pointer.GetString(service.Address)
	port := pointer.GetUint16(service.Port)
	socket := pointer.GetString(service.Socket)
	username := pointer.GetString(s.Username)
	password := pointer.GetString(s.Password)

	if tdp == nil {
		tdp = s.TemplateDelimiters(service)
	}

	switch s.AgentType {
	case MySQLdExporterType, ProxySQLExporterType:
		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = "unix"
		cfg.Addr = socket
		if socket == "" {
			cfg.Net = "tcp"
			cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}
		cfg.Timeout = dialTimeout
		cfg.DBName = database
		cfg.Params = make(map[string]string)
		if s.TLS {
			// TODO: how certs and other parameters are going to be specified? We need to implement calling RegisterTLSConfig
			// See https://godoc.org/github.com/go-sql-driver/mysql#RegisterTLSConfig
			if s.TLSSkipVerify {
				cfg.Params["tls"] = "skip-verify"
			} else {
				cfg.Params["tls"] = "true"
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		return cfg.FormatDSN()

	case QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = "unix"
		cfg.Addr = socket
		if socket == "" {
			cfg.Net = "tcp"
			cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}
		cfg.Timeout = dialTimeout
		cfg.DBName = database
		cfg.Params = make(map[string]string)
		if s.TLS {
			// TODO: how certs and other parameters are going to be specified? We need to implement calling RegisterTLSConfig
			// See https://godoc.org/github.com/go-sql-driver/mysql#RegisterTLSConfig
			if s.TLSSkipVerify {
				cfg.Params["tls"] = "skip-verify"
			} else {
				cfg.Params["tls"] = "true"
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		// QAN code in pmm-agent uses reform which requires those fields
		cfg.ClientFoundRows = true
		cfg.ParseTime = true

		return cfg.FormatDSN()

	case QANMongoDBProfilerAgentType, MongoDBExporterType:
		q := make(url.Values)
		if dialTimeout != 0 {
			q.Set("connectTimeoutMS", strconv.Itoa(int(dialTimeout/time.Millisecond)))
		}

		// https://docs.mongodb.com/manual/reference/connection-string/
		// > If the connection string does not specify a database/ you must specify a slash (/)
		// between the last host and the question mark (?) that begins the string of options.
		path := database
		if database == "" {
			path = "/"
		}

		if s.TLS {
			q.Add("ssl", "true")
			if s.TLSSkipVerify {
				q.Add("tlsInsecure", "true")
			}

			if s.MongoDBOptions != nil {
				if s.MongoDBOptions.TLSCertificateKey != "" {
					q.Add("tlsCertificateKeyFile", tdp.Left+".TextFiles."+certificateKeyFilePlaceholder+tdp.Right)
				}
				if s.MongoDBOptions.TLSCertificateKeyFilePassword != "" {
					q.Add("tlsCertificateKeyFilePassword", s.MongoDBOptions.TLSCertificateKeyFilePassword)
				}
				if s.MongoDBOptions.TLSCa != "" {
					q.Add("tlsCaFile", tdp.Left+".TextFiles."+caFilePlaceholder+tdp.Right)
				}
			}
		}

		address := socket
		if socket == "" {
			address = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}

		u := &url.URL{
			Scheme:   "mongodb",
			Host:     address,
			Path:     path,
			RawQuery: q.Encode(),
		}
		switch {
		case password != "":
			u.User = url.UserPassword(username, password)
		case username != "":
			u.User = url.User(username)
		}
		dsn := u.String()
		dsn = strings.ReplaceAll(dsn, url.QueryEscape(tdp.Left), tdp.Left)
		dsn = strings.ReplaceAll(dsn, url.QueryEscape(tdp.Right), tdp.Right)
		return dsn

	case PostgresExporterType, QANPostgreSQLPgStatementsAgentType, QANPostgreSQLPgStatMonitorAgentType:
		q := make(url.Values)

		sslmode := "disable"
		if s.TLS {
			if s.TLSSkipVerify {
				sslmode = "require"
			} else {
				sslmode = "verify-full"
			}
		}
		q.Set("sslmode", sslmode)

		if dialTimeout != 0 {
			q.Set("connect_timeout", strconv.Itoa(int(dialTimeout.Seconds())))
		}

		address := ""
		if socket == "" {
			address = net.JoinHostPort(host, strconv.Itoa(int(port)))
		} else {
			// Set socket dirrectory as host URI parameter.
			q.Set("host", socket)
			// In case of empty url.URL.Host we need to identify a start of a path (database name).
			database = "/" + database
		}

		u := &url.URL{
			Scheme:   "postgres",
			Host:     address,
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

// ExporterURL composes URL to an external exporter.
func (s *Agent) ExporterURL(q *reform.Querier) (string, error) {
	scheme := pointer.GetString(s.MetricsScheme)
	path := pointer.GetString(s.MetricsPath)
	listenPort := int(pointer.GetUint16(s.ListenPort))
	username := pointer.GetString(s.Username)
	password := pointer.GetString(s.Password)

	host := "127.0.0.1"
	if !s.PushMetrics {
		node, err := FindNodeByID(q, *s.RunsOnNodeID)
		if err != nil {
			return "", err
		}
		host = node.Address
	}

	address := net.JoinHostPort(host, strconv.Itoa(listenPort))
	u := &url.URL{
		Scheme: scheme,
		Host:   address,
		Path:   path,
	}
	switch {
	case password != "":
		u.User = url.UserPassword(username, password)
	case username != "":
		u.User = url.User(username)
	}
	return u.String(), nil
}

// IsMySQLTablestatsGroupEnabled returns true if mysqld_exporter tablestats group collectors should be enabled.
func (s *Agent) IsMySQLTablestatsGroupEnabled() bool {
	if s.AgentType != MySQLdExporterType {
		panic(fmt.Errorf("unhandled AgentType %q", s.AgentType))
	}

	switch {
	case s.TableCountTablestatsGroupLimit == 0: // no limit, always enable
		return true
	case s.TableCountTablestatsGroupLimit < 0: // always disable
		return false
	case s.TableCount == nil: // for compatibility with 2.0
		return true
	default:
		return *s.TableCount <= s.TableCountTablestatsGroupLimit
	}
}

// Files returns files map required to connect to DB.
func (s Agent) Files() map[string]string {
	switch s.AgentType {
	case MySQLdExporterType, ProxySQLExporterType:
		return nil
	case QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		return nil
	case QANMongoDBProfilerAgentType, MongoDBExporterType:
		if s.MongoDBOptions != nil {
			return map[string]string{
				caFilePlaceholder:             s.MongoDBOptions.TLSCa,
				certificateKeyFilePlaceholder: s.MongoDBOptions.TLSCertificateKey,
			}
		}
		return nil
	case PostgresExporterType, QANPostgreSQLPgStatementsAgentType, QANPostgreSQLPgStatMonitorAgentType:
		return nil
	default:
		panic(fmt.Errorf("unhandled AgentType %q", s.AgentType))
	}
}

// TemplateDelimiters returns a pair of safe template delimiters that are not present in agent parameters.
func (s Agent) TemplateDelimiters(svc *Service) *DelimiterPair {
	templateParams := []string{
		pointer.GetString(svc.Address),
		pointer.GetString(s.Username),
		pointer.GetString(s.Password),
		pointer.GetString(s.MetricsPath),
	}

	switch svc.ServiceType {
	case MySQLServiceType:
	case MongoDBServiceType:
		if s.MongoDBOptions != nil {
			templateParams = append(templateParams, s.MongoDBOptions.TLSCertificateKeyFilePassword)
		}
	case PostgreSQLServiceType:
	case ProxySQLServiceType:
	case HAProxyServiceType:
	case ExternalServiceType:
	}

	tdp := TemplateDelimsPair(
		templateParams...,
	)
	return &tdp
}

// check interfaces
var (
	_ reform.BeforeInserter = (*Agent)(nil)
	_ reform.BeforeUpdater  = (*Agent)(nil)
	_ reform.AfterFinder    = (*Agent)(nil)
)
