// Copyright (C) 2023 Percona LLC
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
	"bytes"
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/version"
)

//go:generate ../../bin/reform

// AgentType represents Agent type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and VictoriaMetrics.
type AgentType string

const (
	certificateFilePlaceholder    = "certificateFilePlaceholder"
	certificateKeyFilePlaceholder = "certificateKeyFilePlaceholder"
	caFilePlaceholder             = "caFilePlaceholder"
	// AgentStatusUnknown indicates we know nothing about agent because it is not connected.
	AgentStatusUnknown = "AGENT_STATUS_UNKNOWN"
	tcp                = "tcp"
	trueStr            = "true"
	unix               = "unix"
	skipVerify         = "skip-verify"
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

var v2_42 = version.MustParse("2.42.0-0")

// PMMServerAgentID is a special Agent ID representing pmm-agent on PMM Server.
const PMMServerAgentID = string("pmm-server") // a special ID, reserved for PMM Server

// ExporterOptions represents structure for special Exporter options.
type ExporterOptions struct {
	ExposeExporter     bool                `reform:"expose_exporter"`
	PushMetrics        bool                `reform:"push_metrics"`
	DisabledCollectors pq.StringArray      `reform:"disabled_collectors"`
	MetricsResolutions *MetricsResolutions `reform:"metrics_resolutions"`
	MetricsPath        *string             `reform:"metrics_path"`
	MetricsScheme      *string             `reform:"metrics_scheme"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c ExporterOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *ExporterOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// QANOptions represents structure for special QAN options.
type QANOptions struct {
	MaxQueryLength          int32 `reform:"max_query_length"`
	MaxQueryLogSize         int64 `reform:"max_query_log_size"`
	QueryExamplesDisabled   bool  `reform:"query_examples_disabled"`
	CommentsParsingDisabled bool  `reform:"comments_parsing_disabled"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c QANOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *QANOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// AWSOptions represents structure for special AWS options.
type AWSOptions struct {
	AWSAccessKey               string `reform:"aws_access_key"`
	AWSSecretKey               string `reform:"aws_secret_key"`
	RDSBasicMetricsDisabled    bool   `reform:"rds_basic_metrics_disabled"`
	RDSEnhancedMetricsDisabled bool   `reform:"rds_enhanced_metrics_disabled"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c AWSOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *AWSOptions) Scan(src interface{}) error { return jsonScan(c, src) }

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

// MongoDBOptions represents structure for special MongoDB options.
type MongoDBOptions struct {
	TLSCertificateKey             string   `json:"tls_certificate_key"`
	TLSCertificateKeyFilePassword string   `json:"tls_certificate_key_file_password"`
	TLSCa                         string   `json:"tls_ca"`
	AuthenticationMechanism       string   `json:"authentication_mechanism"`
	AuthenticationDatabase        string   `json:"authentication_database"`
	StatsCollections              []string `json:"stats_collections"`
	CollectionsLimit              int32    `json:"collections_limit"`
	EnableAllCollectors           bool     `json:"enable_all_collectors"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c MongoDBOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *MongoDBOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// MySQLOptions represents structure for special MySQL options.
type MySQLOptions struct {
	TLSCa   string `json:"tls_ca"`
	TLSCert string `json:"tls_cert"`
	TLSKey  string `json:"tls_key"`

	// TableCount stores last known table count. NULL if unknown.
	TableCount *int32 `reform:"table_count"`

	// Tablestats group collectors are disabled if there are more than that number of tables.
	// 0 means tablestats group collectors are always enabled (no limit).
	// Negative value means tablestats group collectors are always disabled.
	// See IsMySQLTablestatsGroupEnabled method.
	TableCountTablestatsGroupLimit int32 `reform:"table_count_tablestats_group_limit"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c MySQLOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *MySQLOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// PostgreSQLOptions represents structure for special PostgreSQL options.
type PostgreSQLOptions struct {
	SSLCa                  string  `json:"ssl_ca"`
	SSLCert                string  `json:"ssl_cert"`
	SSLKey                 string  `json:"ssl_key"`
	AutoDiscoveryLimit     int32   `json:"auto_discovery_limit"`
	DatabaseCount          int32   `json:"database_count"`
	PGSMVersion            *string `json:"pgsm_version"`
	MaxExporterConnections int32   `json:"max_exporter_connections"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c PostgreSQLOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *PostgreSQLOptions) Scan(src interface{}) error { return jsonScan(c, src) }

// PMMAgentWithPushMetricsSupport - version of pmmAgent,
// that support vmagent and push metrics mode
// will be released with PMM Agent v2.12.
var PMMAgentWithPushMetricsSupport = version.MustParse("2.11.99")

// Agent represents Agent as stored in database.
//
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

	Disabled        bool    `reform:"disabled"`
	Status          string  `reform:"status"`
	ListenPort      *uint16 `reform:"listen_port"`
	Version         *string `reform:"version"`
	ProcessExecPath *string `reform:"process_exec_path"`

	Username      *string `reform:"username"`
	Password      *string `reform:"password"`
	AgentPassword *string `reform:"agent_password"`
	TLS           bool    `reform:"tls"`
	TLSSkipVerify bool    `reform:"tls_skip_verify"`

	LogLevel *string `reform:"log_level"`

	ExporterOptions *ExporterOptions `reform:"exporter_options"`
	QANOptions      *QANOptions      `reform:"qan_options"`

	AWSOptions        *AWSOptions        `reform:"aws_options"`
	AzureOptions      *AzureOptions      `reform:"azure_options"`
	MongoDBOptions    *MongoDBOptions    `reform:"mongo_options"`
	MySQLOptions      *MySQLOptions      `reform:"mysql_options"`
	PostgreSQLOptions *PostgreSQLOptions `reform:"postgresql_options"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *Agent) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	if s.Status == "" && s.AgentType != ExternalExporterType && s.AgentType != PMMAgentType {
		s.Status = AgentStatusUnknown
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

// GetAgentPassword returns agent password, if it is empty then agent ID.
func (s *Agent) GetAgentPassword() string {
	password := s.AgentID
	if pointer.GetString(s.AgentPassword) != "" {
		password = *s.AgentPassword
	}

	return password
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
func (c *DBConfig) Valid() bool {
	return c.Address != "" || c.Socket != ""
}

// DBConfig returns DBConfig for given Service with this agent.
func (s *Agent) DBConfig(service *Service) *DBConfig {
	return &DBConfig{
		User:     pointer.GetString(s.Username),
		Password: pointer.GetString(s.Password),
		Address:  pointer.GetString(service.Address),
		Port:     int(pointer.GetUint16(service.Port)),
		Socket:   pointer.GetString(service.Socket),
	}
}

// DSNParams represents the parameters for configuring a Data Source Name (DSN).
type DSNParams struct {
	DialTimeout time.Duration
	Database    string

	PostgreSQLSupportsSSLSNI bool
}

// DSN returns a DSN string for accessing a given Service with this Agent (and an implicit driver).
func (s *Agent) DSN(service *Service, dsnParams DSNParams, tdp *DelimiterPair, pmmAgentVersion *version.Parsed) string { //nolint:cyclop,maintidx
	host := pointer.GetString(service.Address)
	port := pointer.GetUint16(service.Port)
	socket := pointer.GetString(service.Socket)
	username := pointer.GetString(s.Username)
	password := pointer.GetString(s.Password)

	if tdp == nil {
		tdp = s.TemplateDelimiters(service)
	}

	switch s.AgentType {
	case MySQLdExporterType:
		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = unix
		cfg.Addr = socket
		if socket == "" {
			cfg.Net = tcp
			cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}
		cfg.Timeout = dsnParams.DialTimeout
		cfg.DBName = dsnParams.Database
		cfg.Params = make(map[string]string)
		if s.TLS {
			// It is mandatory to have "custom" as the first case.
			// Except case for backward compatibility.
			// Skip verify for "custom" is handled on pmm-agent side.
			switch {
			// Backward compatibility
			case s.TLSSkipVerify && (pmmAgentVersion == nil || pmmAgentVersion.Less(v2_42)):
				cfg.Params["tls"] = skipVerify
			case len(s.Files()) != 0:
				cfg.Params["tls"] = "custom"
			case s.TLSSkipVerify:
				cfg.Params["tls"] = skipVerify
			default:
				cfg.Params["tls"] = trueStr
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		return cfg.FormatDSN()

	case QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = unix
		cfg.Addr = socket
		if socket == "" {
			cfg.Net = tcp
			cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}
		cfg.Timeout = dsnParams.DialTimeout
		cfg.DBName = dsnParams.Database
		cfg.Params = make(map[string]string)
		if s.TLS {
			// It is mandatory to have "custom" as the first case.
			// Except case for backward compatibility.
			// Skip verify for "custom" is handled on pmm-agent side.
			switch {
			// Backward compatibility
			case pmmAgentVersion != nil && s.TLSSkipVerify && pmmAgentVersion.Less(v2_42):
				cfg.Params["tls"] = skipVerify
			case len(s.Files()) != 0:
				cfg.Params["tls"] = "custom"
			case s.TLSSkipVerify:
				cfg.Params["tls"] = skipVerify
			default:
				cfg.Params["tls"] = trueStr
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		// QAN code in pmm-agent uses reform which requires those fields
		cfg.ClientFoundRows = true
		cfg.ParseTime = true

		return cfg.FormatDSN()

	case ProxySQLExporterType:
		cfg := mysql.NewConfig()
		cfg.User = username
		cfg.Passwd = password
		cfg.Net = unix
		cfg.Addr = socket
		if socket == "" {
			cfg.Net = tcp
			cfg.Addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		}
		cfg.Timeout = dsnParams.DialTimeout
		cfg.DBName = dsnParams.Database
		cfg.Params = make(map[string]string)
		if s.TLS {
			if s.TLSSkipVerify {
				cfg.Params["tls"] = "skip-verify"
			} else {
				cfg.Params["tls"] = trueStr
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		return cfg.FormatDSN()

	case QANMongoDBProfilerAgentType, MongoDBExporterType:
		q := make(url.Values)
		if dsnParams.DialTimeout != 0 {
			q.Set("connectTimeoutMS", strconv.Itoa(int(dsnParams.DialTimeout/time.Millisecond)))
			q.Set("serverSelectionTimeoutMS", strconv.Itoa(int(dsnParams.DialTimeout/time.Millisecond)))
		}

		// https://docs.mongodb.com/manual/reference/connection-string/
		// > If the connection string does not specify a database/ you must specify a slash (/)
		// between the last host and the question mark (?) that begins the string of options.
		path := dsnParams.Database
		if path == "" {
			path = "/"
		}

		// Force direct connections: https://www.mongodb.com/docs/drivers/go/current/fundamentals/connection/#direct-connection
		// It's needed for Actions, we need to execute queries exactly on the node specified in DSN. This parameter
		// prevents driver from switching to Primary node.
		q.Add("directConnection", trueStr)

		if s.TLS {
			q.Add("ssl", trueStr)
			if s.TLSSkipVerify {
				q.Add("tlsInsecure", trueStr)
			}
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
			if s.MongoDBOptions.AuthenticationMechanism != "" {
				q.Add("authMechanism", s.MongoDBOptions.AuthenticationMechanism)
			}
			if s.MongoDBOptions.AuthenticationDatabase != "" {
				q.Add("authSource", s.MongoDBOptions.AuthenticationDatabase)
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

		sslmode := DisableSSLMode
		if s.TLS {
			if s.TLSSkipVerify {
				sslmode = RequireSSLMode
			} else {
				sslmode = VerifyCaSSLMode
			}
			if dsnParams.PostgreSQLSupportsSSLSNI {
				q.Set("sslsni", "0")
			}
		}
		q.Set("sslmode", sslmode)

		if files := s.Files(); len(files) != 0 {
			for key := range files {
				switch key {
				case caFilePlaceholder:
					q.Add("sslrootcert", tdp.Left+".TextFiles."+caFilePlaceholder+tdp.Right)
				case certificateFilePlaceholder:
					q.Add("sslcert", tdp.Left+".TextFiles."+certificateFilePlaceholder+tdp.Right)
				case certificateKeyFilePlaceholder:
					q.Add("sslkey", tdp.Left+".TextFiles."+certificateKeyFilePlaceholder+tdp.Right)
				}
			}
		}

		if dsnParams.DialTimeout != 0 {
			q.Set("connect_timeout", strconv.Itoa(int(dsnParams.DialTimeout.Seconds())))
		}

		address := ""
		database := dsnParams.Database
		if socket == "" {
			address = net.JoinHostPort(host, strconv.Itoa(int(port)))
		} else {
			// Set socket directory as host URI parameter.
			q.Set("host", socket)
			// In case of empty url.URL.Host we need to identify the start of the path (database name).
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

		dsn := u.String()
		dsn = strings.ReplaceAll(dsn, url.QueryEscape(tdp.Left), tdp.Left)
		dsn = strings.ReplaceAll(dsn, url.QueryEscape(tdp.Right), tdp.Right)

		return dsn
	default:
		panic(fmt.Errorf("unhandled AgentType %q", s.AgentType))
	}
}

// ExporterURL composes URL to an external exporter.
func (s *Agent) ExporterURL(q *reform.Querier) (string, error) {
	if s.ExporterOptions == nil {
		s.ExporterOptions = &ExporterOptions{}
	}

	scheme := pointer.GetString(s.ExporterOptions.MetricsScheme)
	path := pointer.GetString(s.ExporterOptions.MetricsPath)
	listenPort := int(pointer.GetUint16(s.ListenPort))
	username := pointer.GetString(s.Username)
	password := pointer.GetString(s.Password)

	host := "127.0.0.1"
	if !s.ExporterOptions.PushMetrics {
		node, err := FindNodeByID(q, *s.RunsOnNodeID)
		if err != nil {
			return "", err
		}
		host = node.Address
	}

	address := net.JoinHostPort(host, strconv.Itoa(listenPort))
	// We have to split MetricsPath into the path and the query because it may contain both.
	// Example: "/metrics?format=prometheus&output=json"
	p := strings.Split(path, "?")

	u := &url.URL{
		Scheme: scheme,
		Host:   address,
		Path:   p[0],
	}

	if len(p) > 1 {
		u.RawQuery = p[1]
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

	if s.MySQLOptions == nil {
		s.MySQLOptions = &MySQLOptions{}
	}

	switch {
	case s.MySQLOptions.TableCountTablestatsGroupLimit == 0: // server defined
		return true
	case s.MySQLOptions.TableCountTablestatsGroupLimit < 0: // always disabled
		return false
	case s.MySQLOptions.TableCount == nil: // for compatibility with 2.0
		return true
	default:
		return *s.MySQLOptions.TableCount <= s.MySQLOptions.TableCountTablestatsGroupLimit
	}
}

// Files returns files map required to connect to DB.
func (s Agent) Files() map[string]string {
	switch s.AgentType {
	case MySQLdExporterType, QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		if s.MySQLOptions != nil {
			files := make(map[string]string)
			if s.MySQLOptions.TLSCa != "" {
				files["tlsCa"] = s.MySQLOptions.TLSCa
			}
			if s.MySQLOptions.TLSCert != "" {
				files["tlsCert"] = s.MySQLOptions.TLSCert
			}
			if s.MySQLOptions.TLSKey != "" {
				files["tlsKey"] = s.MySQLOptions.TLSKey
			}
			return files
		}
		return nil
	case ProxySQLExporterType:
		return nil
	case QANMongoDBProfilerAgentType, MongoDBExporterType:
		if s.MongoDBOptions != nil {
			files := make(map[string]string)
			if s.MongoDBOptions.TLSCa != "" {
				files[caFilePlaceholder] = s.MongoDBOptions.TLSCa
			}
			if s.MongoDBOptions.TLSCertificateKey != "" {
				files[certificateKeyFilePlaceholder] = s.MongoDBOptions.TLSCertificateKey
			}
			return files
		}
		return nil
	case PostgresExporterType, QANPostgreSQLPgStatementsAgentType, QANPostgreSQLPgStatMonitorAgentType:
		if s.PostgreSQLOptions != nil {
			files := make(map[string]string)

			if s.PostgreSQLOptions.SSLCa != "" {
				files[caFilePlaceholder] = s.PostgreSQLOptions.SSLCa
			}
			if s.PostgreSQLOptions.SSLCert != "" {
				files[certificateFilePlaceholder] = s.PostgreSQLOptions.SSLCert
			}
			if s.PostgreSQLOptions.SSLKey != "" {
				files[certificateKeyFilePlaceholder] = s.PostgreSQLOptions.SSLKey
			}
			return files
		}
		return nil
	default:
		panic(fmt.Errorf("unhandled AgentType %q", s.AgentType))
	}
}

// TemplateDelimiters returns a pair of safe template delimiters that are not present in agent parameters.
func (s Agent) TemplateDelimiters(svc *Service) *DelimiterPair {
	if s.ExporterOptions == nil {
		s.ExporterOptions = &ExporterOptions{}
	}

	templateParams := []string{
		pointer.GetString(svc.Address),
		pointer.GetString(s.Username),
		pointer.GetString(s.Password),
		pointer.GetString(s.ExporterOptions.MetricsPath),
	}

	switch svc.ServiceType {
	case MySQLServiceType:
		if s.MySQLOptions != nil {
			templateParams = append(templateParams, s.MySQLOptions.TLSKey)
		}
	case MongoDBServiceType:
		if s.MongoDBOptions != nil {
			templateParams = append(templateParams, s.MongoDBOptions.TLSCertificateKeyFilePassword)
		}
	case PostgreSQLServiceType:
		if s.PostgreSQLOptions != nil {
			templateParams = append(templateParams, s.PostgreSQLOptions.SSLKey)
		}
	case ProxySQLServiceType:
	case HAProxyServiceType:
	case ExternalServiceType:
	}

	tdp := TemplateDelimsPair(
		templateParams...,
	)
	return &tdp
}

// HashPassword func to calculate password hash. Public and overridable for testing purposes.
var HashPassword = func(password, salt string) (string, error) {
	buf, err := bcrypt.GenerateFromPasswordAndSalt([]byte(password), bcrypt.DefaultCost, []byte(salt))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

const webConfigTemplate = `basic_auth_users:
    pmm: {{ . }}
`

// BuildWebConfigFile builds prometheus-compatible basic auth configuration.
func (s *Agent) BuildWebConfigFile() (string, error) {
	password := s.GetAgentPassword()
	salt := getPasswordSalt(s)

	hashedPassword, err := HashPassword(password, salt)
	if err != nil {
		return "", errors.Wrap(err, "Failed to hash password")
	}

	var configBuffer bytes.Buffer
	if tmpl, err := template.New("webConfig").Parse(webConfigTemplate); err != nil {
		return "", errors.Wrap(err, "Failed to parse webconfig template")
	} else if err = tmpl.Execute(&configBuffer, hashedPassword); err != nil {
		return "", errors.Wrap(err, "Failed to execute webconfig template")
	}

	config := configBuffer.String()

	return config, nil
}

func getPasswordSalt(s *Agent) string {
	if s.AgentID != "" && len(s.AgentID) >= bcrypt.MaxSaltSize {
		return s.AgentID[len(s.AgentID)-bcrypt.MaxSaltSize:]
	}

	return "pmm-salt-magic--"
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Agent)(nil)
	_ reform.BeforeUpdater  = (*Agent)(nil)
	_ reform.AfterFinder    = (*Agent)(nil)
)
