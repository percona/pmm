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
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/crypto/bcrypt"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/envvars"
	"github.com/percona/pmm/utils/vault"
	"github.com/percona/pmm/version"
)

//go:generate go tool reform

// AgentType represents Agent type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and VictoriaMetrics.
type AgentType string

// Text file names carrying TLS material to pmm-agent. They are part of the agent wire
// protocol and are declared canonically in api/agent/v1, which models must not import;
// TestTLSFileNamesMatchAgentAPI in managed/services/agents keeps the two copies in step.
const (
	TLSCaFileName   = "tlsCa"
	TLSCertFileName = "tlsCert"
	TLSKeyFileName  = "tlsKey"
)

const (
	certificateFilePlaceholder    = "certificateFilePlaceholder"
	certificateKeyFilePlaceholder = "certificateKeyFilePlaceholder"
	caFilePlaceholder             = "caFilePlaceholder"
	// AgentStatusUnknown indicates we know nothing about agent because it is not connected.
	AgentStatusUnknown = "AGENT_STATUS_UNKNOWN"
	// AgentStatusDone indicates thay the agent has either been stopped or disabled.
	agentStatusDone    = "AGENT_STATUS_DONE"
	tcp                = "tcp"
	trueStr            = "true"
	unix               = "unix"
	skipVerify         = "skip-verify"
	defaultDialTimeout = 2 * time.Second
	valkeyDialTimeout  = 3 * time.Second
	agentIDLabel       = "agent_id"
	agentTypeLabel     = "agent_type"
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
	QANMongoDBMongologAgentType         AgentType = "qan-mongodb-mongolog-agent"
	QANPostgreSQLPgStatementsAgentType  AgentType = "qan-postgresql-pgstatements-agent"
	QANPostgreSQLPgStatMonitorAgentType AgentType = "qan-postgresql-pgstatmonitor-agent"
	ExternalExporterType                AgentType = "external-exporter"
	VMAgentType                         AgentType = "vmagent"
	NomadAgentType                      AgentType = "nomad-agent"
	ValkeyExporterType                  AgentType = "valkey_exporter"
	RTAMongoDBAgentType                 AgentType = "rta-mongodb-agent"
)

// GetRTAAgentTypes returns all Real-Time Analytics Agent types.
func GetRTAAgentTypes() []AgentType {
	return []AgentType{
		RTAMongoDBAgentType,
		// Add more types here once they are implemented.
	}
}

var v2_42 = version.MustParse("2.42.0-0")

// PMMServerAgentID is a special Agent ID representing pmm-agent on PMM Server.
// It takes the value of "pmm-server" in non-HA setups, while in HA setups it is set to the actual
// pmm-agent's Agent ID, which is a UUID.
var PMMServerAgentID = string("pmm-server")

// AgentConfigFilePath is the default path to pmm-agent config file; it changes to /srv in HA setups.
var AgentConfigFilePath = "/usr/local/percona/pmm/config/pmm-agent.yaml"

// ExporterOptions represents structure for special Exporter options.
type ExporterOptions struct {
	ExposeExporter     bool                `json:"expose_exporter"`
	PushMetrics        bool                `json:"push_metrics"`
	DisabledCollectors pq.StringArray      `json:"disabled_collectors"`
	MetricsResolutions *MetricsResolutions `json:"metrics_resolutions"`
	MetricsPath        string              `json:"metrics_path"`
	MetricsScheme      string              `json:"metrics_scheme"`
	// Connection timeout for exporter. Optional.
	ConnectionTimeout *time.Duration `json:"connection_timeout"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c ExporterOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *ExporterOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all ExporterOptions fields are unset or have zero values, otherwise returns false.
func (c ExporterOptions) IsEmpty() bool {
	return !c.ExposeExporter &&
		!c.PushMetrics &&
		len(c.DisabledCollectors) == 0 &&
		c.MetricsResolutions == nil &&
		c.MetricsPath == "" &&
		c.MetricsScheme == "" &&
		pointer.Get(c.ConnectionTimeout) == 0
}

// QANOptions represents structure for special QAN options.
type QANOptions struct {
	MaxQueryLength          int32 `json:"max_query_length"`
	MaxQueryLogSize         int64 `json:"max_query_log_size"`
	QueryExamplesDisabled   bool  `json:"query_examples_disabled"`
	CommentsParsingDisabled bool  `json:"comments_parsing_disabled"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c QANOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *QANOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all QANOptions fields are unset or have zero values, otherwise returns false.
func (c QANOptions) IsEmpty() bool {
	return c.MaxQueryLength == 0 &&
		c.MaxQueryLogSize == 0 &&
		!c.QueryExamplesDisabled &&
		!c.CommentsParsingDisabled
}

// AWSOptions represents structure for special AWS options.
type AWSOptions struct {
	AWSAccessKey               string `json:"aws_access_key"`
	AWSSecretKey               string `json:"aws_secret_key"`
	AWSRoleARN                 string `json:"aws_role_arn"`
	RDSBasicMetricsDisabled    bool   `json:"rds_basic_metrics_disabled"`
	RDSEnhancedMetricsDisabled bool   `json:"rds_enhanced_metrics_disabled"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c AWSOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *AWSOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all AWSOptions fields are unset or have zero values, otherwise returns false.
func (c AWSOptions) IsEmpty() bool {
	return c.AWSAccessKey == "" &&
		c.AWSSecretKey == "" &&
		c.AWSRoleARN == "" &&
		!c.RDSBasicMetricsDisabled &&
		!c.RDSEnhancedMetricsDisabled
}

// CredentialsKey returns a stable identifier for the AWS identity these options resolve to.
// For static keys it returns the access key unchanged, preserving existing group IDs. A role
// ARN is hashed because this value becomes a path component on the pmm-agent side.
func (c AWSOptions) CredentialsKey() string {
	if c.AWSAccessKey != "" {
		return c.AWSAccessKey
	}
	if c.AWSRoleARN != "" {
		sum := sha256.Sum256([]byte(c.AWSRoleARN))
		return "role-" + hex.EncodeToString(sum[:])[:16]
	}
	return ""
}

// Validate returns an error if the AWS options are mutually inconsistent.
func (c AWSOptions) Validate() error {
	if c.AWSRoleARN != "" && (c.AWSAccessKey != "" || c.AWSSecretKey != "") {
		return status.Error(codes.InvalidArgument, "Both AWS role ARN and AWS access key/secret key are set; they are mutually exclusive.")
	}

	return nil
}

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
func (c *AzureOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all AzureOptions fields are unset or have zero values, otherwise returns false.
func (c AzureOptions) IsEmpty() bool {
	return c.SubscriptionID == "" &&
		c.ClientID == "" &&
		c.ClientSecret == "" &&
		c.TenantID == "" &&
		c.ResourceGroup == ""
}

// MongoDBOptions represents structure for special MongoDB options.
type MongoDBOptions struct {
	TLSCertificateKey              string   `json:"tls_certificate_key"`
	TLSCertificateKeyFilePassword  string   `json:"tls_certificate_key_file_password"`
	TLSCa                          string   `json:"tls_ca"`
	AuthenticationMechanism        string   `json:"authentication_mechanism"`
	AuthenticationDatabase         string   `json:"authentication_database"`
	StatsCollections               []string `json:"stats_collections"`
	CollectionsLimit               int32    `json:"collections_limit"`
	EnableAllCollectors            bool     `json:"enable_all_collectors"`
	EnableDiagnosticDataHistograms bool     `json:"enable_diagnostic_data_histograms"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c MongoDBOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *MongoDBOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all MongoDBOptions fields are unset or have zero values, otherwise returns false.
func (c MongoDBOptions) IsEmpty() bool {
	return c.TLSCertificateKey == "" &&
		c.TLSCertificateKeyFilePassword == "" &&
		c.TLSCa == "" &&
		c.AuthenticationMechanism == "" &&
		c.AuthenticationDatabase == "" &&
		len(c.StatsCollections) == 0 &&
		c.CollectionsLimit == 0 &&
		!c.EnableAllCollectors &&
		!c.EnableDiagnosticDataHistograms
}

// MySQLOptions represents structure for special MySQL options.
type MySQLOptions struct {
	TLSCa   string `json:"tls_ca"`
	TLSCert string `json:"tls_cert"`
	TLSKey  string `json:"tls_key"`

	// TableCount stores last known table count. NULL if unknown.
	TableCount *int32 `json:"table_count"`

	// Tablestats group collectors are disabled if there are more than that number of tables.
	// 0 means tablestats group collectors are always enabled (no limit).
	// Negative value means tablestats group collectors are always disabled.
	// See IsMySQLTablestatsGroupEnabled method.
	TableCountTablestatsGroupLimit int32 `json:"table_count_tablestats_group_limit"`

	// Extra DSN query parameters for MySQL.
	ExtraDSNParams map[string]string `json:"extra_dsn_params"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c MySQLOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *MySQLOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all MySQLOptions fields are unset or have zero values, otherwise returns false.
func (c MySQLOptions) IsEmpty() bool {
	return c.TLSCa == "" &&
		c.TLSCert == "" &&
		c.TLSKey == "" &&
		c.TableCount == nil &&
		c.TableCountTablestatsGroupLimit == 0
}

// PostgreSQLOptions represents structure for special PostgreSQL options.
type PostgreSQLOptions struct {
	SSLCa                  string  `json:"ssl_ca"`
	SSLCert                string  `json:"ssl_cert"`
	SSLKey                 string  `json:"ssl_key"`
	AutoDiscoveryLimit     *int32  `json:"auto_discovery_limit"`
	DatabaseCount          int32   `json:"database_count"`
	PGSMVersion            *string `json:"pgsm_version"`
	MaxExporterConnections int32   `json:"max_exporter_connections"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c PostgreSQLOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *PostgreSQLOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all PostgreSQLOptions fields are unset or have zero values, otherwise returns false.
func (c PostgreSQLOptions) IsEmpty() bool {
	return c.SSLCa == "" &&
		c.SSLCert == "" &&
		c.SSLKey == "" &&
		c.AutoDiscoveryLimit == nil &&
		c.DatabaseCount == 0 &&
		c.PGSMVersion == nil &&
		c.MaxExporterConnections == 0
}

// ValkeyOptions represents a structure for special Valkey options.
type ValkeyOptions struct {
	TLS     bool   `json:"tls"`
	SSLCa   string `json:"ssl_ca"`
	SSLCert string `json:"ssl_cert"`
	SSLKey  string `json:"ssl_key"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c ValkeyOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *ValkeyOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all ValkeyOptions fields are unset or have zero values, otherwise returns false.
func (c ValkeyOptions) IsEmpty() bool {
	return c.SSLCa == "" &&
		c.SSLCert == "" &&
		c.SSLKey == ""
}

// clientKeyPairIncomplete reports whether only one half of the TLS client key pair is set.
func (c ValkeyOptions) clientKeyPairIncomplete() bool {
	return (c.SSLCert == "") != (c.SSLKey == "")
}

// RTAOptions represents structure for Real-Time Analytics options.
type RTAOptions struct {
	// Queries collection interval for this agent.
	CollectInterval *time.Duration `json:"collect_interval,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c RTAOptions) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *RTAOptions) Scan(src any) error { return jsonScan(c, src) }

// IsEmpty returns true if all RTAOptions fields are unset or have zero values, otherwise returns false.
func (c RTAOptions) IsEmpty() bool {
	return c.CollectInterval == nil
}

// Merge merges non-zero values from opts into c.
func (c *RTAOptions) Merge(opts *RTAOptions) {
	if pointer.Get(opts).IsEmpty() {
		return
	}

	if opts.CollectInterval != nil {
		c.CollectInterval = opts.CollectInterval
	}
	// Add new fields here.
}

// Agent represents Agent as stored in database.
//
//reform:agents
type Agent struct {
	AgentID              string    `reform:"agent_id,pk"`
	AgentType            AgentType `reform:"agent_type"`
	RunsOnNodeID         *string   `reform:"runs_on_node_id"`
	ServiceID            *string   `reform:"service_id"`
	NodeID               *string   `reform:"node_id"`
	PMMAgentID           *string   `reform:"pmm_agent_id"`
	CustomLabels         []byte    `reform:"custom_labels"`
	EnvironmentVariables []byte    `reform:"environment_variables"`
	CreatedAt            time.Time `reform:"created_at"`
	UpdatedAt            time.Time `reform:"updated_at"`

	Disabled        bool    `reform:"disabled"`
	Status          string  `reform:"status"`
	ListenPort      *uint16 `reform:"listen_port"`
	Version         *string `reform:"version"`
	ProcessExecPath *string `reform:"process_exec_path"`
	IsConnected     bool    `reform:"is_connected"`

	Username      *string `reform:"username"`
	Password      *string `reform:"password"`
	AgentPassword *string `reform:"agent_password"`
	TLS           bool    `reform:"tls"`
	TLSSkipVerify bool    `reform:"tls_skip_verify"`
	// VaultPath is the Vault secret holding the database credentials, which then never reach PMM Server.
	// Any TLS option may also hold a vault:<path>#<key> reference instead of the value, see tlsFields.
	VaultPath *string `reform:"vault_path"`

	LogLevel *string `reform:"log_level"`

	ExporterOptions ExporterOptions `reform:"exporter_options"`
	QANOptions      QANOptions      `reform:"qan_options"`
	RTAOptions      RTAOptions      `reform:"rta_options"`

	AWSOptions        AWSOptions        `reform:"aws_options"`
	AzureOptions      AzureOptions      `reform:"azure_options"`
	MongoDBOptions    MongoDBOptions    `reform:"mongo_options"`
	MySQLOptions      MySQLOptions      `reform:"mysql_options"`
	PostgreSQLOptions PostgreSQLOptions `reform:"postgresql_options"`
	ValkeyOptions     ValkeyOptions     `reform:"valkey_options"`

	// undecrypted holds the stored ciphertext of the secret fields this node's key could not
	// decrypt, by field name, so that writing the Agent back keeps them intact.
	undecrypted map[string]string

	// checkUsername and checkPassword stand in for those in Vault, see WithCheckSecrets.
	checkUsername *string
	checkPassword *string
}

// WithCheckSecrets returns a copy of the Agent that connects with the given secrets, by the key of its
// Vault secret, instead of the ones in Vault. It serves to check new secrets before they are stored in
// Vault, where they replace the ones every Agent reading the secret uses. The copy is never to be stored.
func (a Agent) WithCheckSecrets(secrets map[string]string) *Agent {
	if v, ok := secrets[VaultUsernameKey]; ok {
		a.checkUsername = &v
	}
	if v, ok := secrets[VaultPasswordKey]; ok {
		a.checkPassword = &v
	}

	// a is a copy, options included, so their fields can be set.
	path := strings.Trim(pointer.GetString(a.VaultPath), "/")
	for _, f := range a.tlsFields() {
		v, ok := secrets[f.key]
		if ok && *f.value == VaultRef(path, f.key) {
			*f.value = v
		}
	}
	return &a
}

// secretField is a field of an Agent that holds a secret, or a vault:<path>#<key> reference to one.
type secretField struct {
	// key is the key of the Agent's Vault secret that the field is stored under when it is moved there.
	key   string
	value *string
}

// tlsFields returns the TLS options, the client key with the certificates that go with it, by the key of
// the Vault secret they are stored under. An Agent has only the options of its type set, so the keys of the
// others do not clash with them. Any option may be nil.
func tlsFields(mysql *MySQLOptions, mongo *MongoDBOptions, postgres *PostgreSQLOptions, valkey *ValkeyOptions) []secretField {
	var res []secretField
	if mysql != nil {
		res = append(res,
			secretField{VaultTLSCaKey, &mysql.TLSCa},
			secretField{VaultTLSCertKey, &mysql.TLSCert},
			secretField{VaultTLSKeyKey, &mysql.TLSKey},
		)
	}
	if mongo != nil {
		res = append(res,
			secretField{VaultTLSCaKey, &mongo.TLSCa},
			secretField{VaultTLSCertificateKeyKey, &mongo.TLSCertificateKey},
			secretField{VaultTLSCertificateKeyFilePasswordKey, &mongo.TLSCertificateKeyFilePassword},
		)
	}
	if postgres != nil {
		res = append(res,
			secretField{VaultTLSCaKey, &postgres.SSLCa},
			secretField{VaultTLSCertKey, &postgres.SSLCert},
			secretField{VaultTLSKeyKey, &postgres.SSLKey},
		)
	}
	if valkey != nil {
		res = append(res,
			secretField{VaultTLSCaKey, &valkey.SSLCa},
			secretField{VaultTLSCertKey, &valkey.SSLCert},
			secretField{VaultTLSKeyKey, &valkey.SSLKey},
		)
	}
	return res
}

func (a *Agent) tlsFields() []secretField {
	return tlsFields(&a.MySQLOptions, &a.MongoDBOptions, &a.PostgreSQLOptions, &a.ValkeyOptions)
}

// VaultRef returns the vault:<path>#<key> reference to key of the Vault secret at path.
func VaultRef(path, key string) string {
	return vault.RefPrefix + strings.Trim(path, "/") + "#" + key
}

// passwordUsedByServer reports whether PMM Server uses the Agent's password itself, to scrape the exporter,
// rather than pmm-agent to connect to a database.
func (a *Agent) passwordUsedByServer() bool {
	return a.AgentType == ExternalExporterType || a.AgentType == VMAgentType
}

// agentFields returns the fields of the Agent that pmm-agent reads from Vault when they hold a reference:
// the database password and the TLS options.
func (a *Agent) agentFields() []secretField {
	res := a.tlsFields()
	if a.Password != nil && !a.passwordUsedByServer() {
		res = append(res, secretField{VaultPasswordKey, a.Password})
	}
	return res
}

// serverFields returns the fields of the Agent that PMM Server reads from Vault when they hold a reference,
// see WithServerSecrets: those it uses itself, and those it sends to pmm-agent as they are.
func (a *Agent) serverFields() []secretField {
	var res []secretField
	if a.AgentPassword != nil {
		res = append(res, secretField{"agent_password", a.AgentPassword})
	}
	if a.Password != nil && a.passwordUsedByServer() {
		res = append(res, secretField{VaultPasswordKey, a.Password})
	}
	return append(res,
		secretField{"aws_access_key", &a.AWSOptions.AWSAccessKey},
		secretField{"aws_secret_key", &a.AWSOptions.AWSSecretKey},
		secretField{"azure_client_id", &a.AzureOptions.ClientID},
		secretField{"azure_client_secret", &a.AzureOptions.ClientSecret},
		secretField{"azure_tenant_id", &a.AzureOptions.TenantID},
		secretField{"azure_subscription_id", &a.AzureOptions.SubscriptionID},
	)
}

// UsesVault reports whether pmm-agent reads any of the Agent's secrets from Vault.
func (a *Agent) UsesVault() bool {
	if a.VaultPath != nil {
		return true
	}
	for _, f := range a.agentFields() {
		if vault.IsRef(*f.value) {
			return true
		}
	}
	return false
}

// HasServerSecrets reports whether PMM Server reads any of the Agent's secrets from Vault.
func (a *Agent) HasServerSecrets() bool {
	for _, f := range a.serverFields() {
		if vault.IsRef(*f.value) {
			return true
		}
	}
	return false
}

// WithServerSecrets returns a copy of the Agent with the references in the fields that PMM Server reads
// from Vault replaced with the secrets they refer to. The copy is never to be stored.
func (a Agent) WithServerSecrets(ctx context.Context) (*Agent, error) {
	// a is a copy, options included, but the strings these fields point to are shared with the caller.
	if a.AgentPassword != nil {
		a.AgentPassword = new(*a.AgentPassword)
	}
	if a.Password != nil {
		a.Password = new(*a.Password)
	}

	for _, f := range a.serverFields() {
		v, err := secrets.Resolve(ctx, *f.value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.key, err)
		}
		*f.value = v
	}
	return &a, nil
}

// ServerSecretsFingerprint returns a fingerprint of the values of the fields that PMM Server reads from Vault,
// for telling whether they changed there, see WithServerSecrets.
func (a *Agent) ServerSecretsFingerprint() string {
	h := sha256.New()
	for _, f := range a.serverFields() {
		h.Write([]byte(f.key + "\x00" + *f.value + "\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// validateVaultRefs returns an error for a field of the Agent that holds an invalid Vault reference, or a
// reference to a secret that PMM Server cannot read although it is the one to read it.
func (a *Agent) validateVaultRefs(ctx context.Context) error {
	for _, f := range slices.Concat(a.agentFields(), a.serverFields()) {
		if !vault.IsRef(*f.value) {
			continue
		}
		_, err := vault.ParseRef(*f.value)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid %s: %s.", f.key, err)
		}
	}

	_, err := a.WithServerSecrets(ctx)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "Failed to read the %s.", err)
	}
	return nil
}

// BeforeInsert implements reform.BeforeInserter interface.
func (a *Agent) BeforeInsert() error {
	now := Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if len(a.CustomLabels) == 0 {
		a.CustomLabels = nil
	}
	if len(a.EnvironmentVariables) == 0 {
		a.EnvironmentVariables = nil
	}
	if a.Status == "" && a.AgentType != PMMAgentType {
		a.Status = AgentStatusUnknown
	}
	if a.Disabled {
		a.Status = agentStatusDone
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (a *Agent) BeforeUpdate() error {
	a.UpdatedAt = Now()
	if len(a.CustomLabels) == 0 {
		a.CustomLabels = nil
	}
	if len(a.EnvironmentVariables) == 0 {
		a.EnvironmentVariables = nil
	}
	if a.Disabled {
		a.Status = agentStatusDone
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (a *Agent) AfterFind() error {
	a.CreatedAt = a.CreatedAt.UTC()
	a.UpdatedAt = a.UpdatedAt.UTC()
	if len(a.CustomLabels) == 0 {
		a.CustomLabels = nil
	}
	if len(a.EnvironmentVariables) == 0 {
		a.EnvironmentVariables = nil
	}
	return nil
}

// GetCustomLabels decodes custom labels.
func (a *Agent) GetCustomLabels() (map[string]string, error) {
	return getLabels(a.CustomLabels)
}

// SetCustomLabels encodes custom labels.
func (a *Agent) SetCustomLabels(m map[string]string) error {
	return setLabels(m, &a.CustomLabels)
}

// GetEnvironmentVariableNames decodes shared environment variable names.
func (a *Agent) GetEnvironmentVariableNames() ([]string, error) {
	if a.EnvironmentVariables == nil {
		return nil, nil
	}

	var names []string
	err := json.Unmarshal(a.EnvironmentVariables, &names)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shared environment variable names: %w", err)
	}
	return names, nil
}

// SetEnvironmentVariableNames encodes shared environment variable names. Names already stored
// (e.g. from before this validation existed, or under since-tightened rules) are carried forward
// as-is: this is a full-replace field, so a caller resending an existing name alongside a new one
// must not be rejected because of a name it did not intend to change.
func (a *Agent) SetEnvironmentVariableNames(names []string) error {
	if len(names) == 0 {
		a.EnvironmentVariables = nil
		return nil
	}

	names, err := envvars.NormalizeNamesAllowing(names, a.GrandfatheredEnvironmentVariableNames())
	if err != nil {
		return err
	}

	b, err := json.Marshal(names)
	if err != nil {
		return fmt.Errorf("failed to marshal shared environment variable names: %w", err)
	}
	a.EnvironmentVariables = b
	return nil
}

// GrandfatheredEnvironmentVariableNames returns the agent's currently-stored environment variable
// names as the set the grandfathering checks compare against, normalized the same way
// NormalizeNamesAllowing normalizes its input. Callers that use it to decide whether an update is
// allowed must read the agent within the same transaction as that update, so the names it
// grandfathers cannot go stale before the update applies them.
//
// Two stored values yield no grandfathering rather than an error. A column that cannot be decoded
// holds nothing worth carrying forward, and failing here would make the row permanently unwritable:
// every non-empty update would be rejected, contradicting the repair path ToAPIAgent documents. An
// empty or whitespace-only entry is skipped for a related reason — grandfathering it would let the
// empty string be written back for good, and pmm-agent can only skip it and warn on every state
// update. Neither can be fixed by the caller, so neither is reported to them.
func (a *Agent) GrandfatheredEnvironmentVariableNames() map[string]struct{} {
	existing, err := a.GetEnvironmentVariableNames()
	if err != nil {
		return nil
	}

	grandfathered := make(map[string]struct{}, len(existing))
	for _, name := range existing {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		grandfathered[name] = struct{}{}
	}

	return grandfathered
}

// GetAgentPassword returns agent password, if it is empty then agent ID.
func (a *Agent) GetAgentPassword() string {
	password := a.AgentID
	if pointer.GetString(a.AgentPassword) != "" {
		password = *a.AgentPassword
	}

	return password
}

// UnifiedLabels returns combined standard and custom labels with empty labels removed.
func (a *Agent) UnifiedLabels() (map[string]string, error) {
	custom, err := a.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	res := map[string]string{
		agentIDLabel:   a.AgentID,
		agentTypeLabel: string(a.AgentType),
	}
	maps.Copy(res, custom)

	err = prepareLabels(res, true)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Keys of the Vault secret at VaultPath. Those of the TLS options are named after their API fields.
const (
	VaultUsernameKey                      = "username"
	VaultPasswordKey                      = "password"
	VaultTLSCaKey                         = "tls_ca"
	VaultTLSCertKey                       = "tls_cert"
	VaultTLSKeyKey                        = "tls_key"
	VaultTLSCertificateKeyKey             = "tls_certificate_key"
	VaultTLSCertificateKeyFilePasswordKey = "tls_certificate_key_file_password"
)

// DBCredentials returns the username and password the Agent connects to its database with.
// For credentials stored in Vault, they are tokens that pmm-agent replaces with the values of the secret,
// escaped as enc. The username of the secret wins over the Agent's own, which stands in when the secret
// has none: the secret is what Vault rotates or issues anew, as it does with dynamic database credentials.
func (a *Agent) DBCredentials(enc vault.Encoding) (string, string) {
	username := pointer.GetString(a.Username)
	path := pointer.GetString(a.VaultPath)
	if path == "" {
		return username, vault.TokenForRef(pointer.GetString(a.Password), enc)
	}

	vaultUsername := vault.TokenWithFallback(path, VaultUsernameKey, enc, username)
	if a.checkUsername != nil {
		vaultUsername = *a.checkUsername
	}
	vaultPassword := vault.Token(path, VaultPasswordKey, enc)
	if a.checkPassword != nil {
		vaultPassword = *a.checkPassword
	}
	return vaultUsername, vaultPassword
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
func (a *Agent) DBConfig(service *Service) *DBConfig {
	username, password := a.DBCredentials(vault.EncodingRaw)
	return &DBConfig{
		User:     username,
		Password: password,
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
func (a *Agent) DSN(service *Service, dsnParams DSNParams, tdp *DelimiterPair, pmmAgentVersion *version.Parsed) string { //nolint:gocognit,cyclop,maintidx
	host := pointer.GetString(service.Address)
	port := pointer.GetUint16(service.Port)
	socket := pointer.GetString(service.Socket)
	username, password := a.DBCredentials(vault.EncodingRaw)

	if tdp == nil {
		tdp = a.TemplateDelimiters(service)
	}

	switch a.AgentType {
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
		if a.TLS {
			// It is mandatory to have "custom" as the first case.
			// Except case for backward compatibility.
			// Skip verify for "custom" is handled on pmm-agent side.
			switch {
			// Backward compatibility
			case a.TLSSkipVerify && (pmmAgentVersion == nil || pmmAgentVersion.Less(v2_42)):
				cfg.Params["tls"] = skipVerify
			case len(a.Files()) != 0:
				cfg.Params["tls"] = "custom"
			case a.TLSSkipVerify:
				cfg.Params["tls"] = skipVerify
			default:
				cfg.Params["tls"] = trueStr
			}
		}

		if a.MySQLOptions.ExtraDSNParams != nil {
			// Add extra DSN parameters if they are set.
			maps.Copy(cfg.Params, a.MySQLOptions.ExtraDSNParams)
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
		if a.TLS {
			// It is mandatory to have "custom" as the first case.
			// Except case for backward compatibility.
			// Skip verify for "custom" is handled on pmm-agent side.
			switch {
			// Backward compatibility
			case pmmAgentVersion != nil && a.TLSSkipVerify && pmmAgentVersion.Less(v2_42):
				cfg.Params["tls"] = skipVerify
			case len(a.Files()) != 0:
				cfg.Params["tls"] = "custom"
			case a.TLSSkipVerify:
				cfg.Params["tls"] = skipVerify
			default:
				cfg.Params["tls"] = trueStr
			}
		}

		if a.MySQLOptions.ExtraDSNParams != nil {
			// Add extra DSN parameters if they are set.
			maps.Copy(cfg.Params, a.MySQLOptions.ExtraDSNParams)
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
		if a.TLS {
			if a.TLSSkipVerify {
				cfg.Params["tls"] = "skip-verify"
			} else {
				cfg.Params["tls"] = trueStr
			}
		}

		// MultiStatements must not be used as it enables SQL injections (in particular, in pmm-agent's Actions)
		cfg.MultiStatements = false

		return cfg.FormatDSN()

	case QANMongoDBProfilerAgentType, QANMongoDBMongologAgentType, MongoDBExporterType, RTAMongoDBAgentType:
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

		if a.TLS {
			q.Add("ssl", trueStr)
			if a.TLSSkipVerify {
				q.Add("tlsInsecure", trueStr)
			}
		}

		if a.MongoDBOptions.TLSCertificateKey != "" {
			q.Add("tlsCertificateKeyFile", tdp.Left+".TextFiles."+certificateKeyFilePlaceholder+tdp.Right)
		}
		if a.MongoDBOptions.TLSCertificateKeyFilePassword != "" {
			q.Add("tlsCertificateKeyFilePassword", vault.TokenForRef(a.MongoDBOptions.TLSCertificateKeyFilePassword, vault.EncodingQuery))
		}
		if a.MongoDBOptions.TLSCa != "" {
			q.Add("tlsCaFile", tdp.Left+".TextFiles."+caFilePlaceholder+tdp.Right)
		}
		if a.MongoDBOptions.AuthenticationMechanism != "" {
			q.Add("authMechanism", a.MongoDBOptions.AuthenticationMechanism)
		}
		if a.MongoDBOptions.AuthenticationDatabase != "" {
			q.Add("authSource", a.MongoDBOptions.AuthenticationDatabase)
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
		username, password = a.DBCredentials(vault.EncodingUserinfo)
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
		if a.TLS {
			if a.TLSSkipVerify {
				sslmode = RequireSSLMode
			} else {
				sslmode = VerifyCaSSLMode
			}
			if dsnParams.PostgreSQLSupportsSSLSNI {
				q.Set("sslsni", "0")
			}
		}
		q.Set("sslmode", sslmode)

		if files := a.Files(); len(files) != 0 {
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
		username, password = a.DBCredentials(vault.EncodingUserinfo)
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

	case ValkeyExporterType:
		urlScheme := "redis"
		if a.TLS {
			urlScheme += "s"
		}
		address := ""
		if socket == "" {
			address = net.JoinHostPort(host, strconv.Itoa(int(port)))
		} else {
			// Set socket directory as host URI parameter.
			address = socket
		}

		u := &url.URL{
			Scheme: urlScheme,
			Host:   address,
		}
		username, password = a.DBCredentials(vault.EncodingUserinfo)
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
		panic(fmt.Errorf("unhandled AgentType %q", a.AgentType))
	}
}

// EffectiveDialTimeout returns the database connection timeout for DSN generation.
// Returns ExporterOptions.ConnectionTimeout if set, otherwise default based on agent type.
//
// Defaults: MySQL, MongoDB, ProxySQL, and PostgreSQL use 2s; Valkey uses 3s.
//
// Exporters without DB connection (node, rds, azure, external) don't use this.
func (a *Agent) EffectiveDialTimeout() time.Duration {
	if a.ExporterOptions.ConnectionTimeout != nil {
		return *a.ExporterOptions.ConnectionTimeout
	}

	if a.AgentType == ValkeyExporterType {
		return valkeyDialTimeout
	}

	return defaultDialTimeout
}

// ExporterURL composes URL to an external exporter.
func (a *Agent) ExporterURL(q *reform.Querier) (string, error) {
	scheme := a.ExporterOptions.MetricsScheme
	path := a.ExporterOptions.MetricsPath
	listenPort := int(pointer.GetUint16(a.ListenPort))
	username := pointer.GetString(a.Username)
	password := pointer.GetString(a.Password)

	host := LocalhostAddr
	if !a.ExporterOptions.PushMetrics {
		node, err := FindNodeByID(q, *a.RunsOnNodeID)
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
func (a *Agent) IsMySQLTablestatsGroupEnabled() bool {
	if a.AgentType != MySQLdExporterType {
		panic(fmt.Errorf("unhandled AgentType %q", a.AgentType))
	}

	switch {
	case a.MySQLOptions.TableCountTablestatsGroupLimit == 0: // server defined
		return true
	case a.MySQLOptions.TableCountTablestatsGroupLimit < 0: // always disabled
		return false
	case a.MySQLOptions.TableCount == nil: // for compatibility with 2.0
		return true
	default:
		return *a.MySQLOptions.TableCount <= a.MySQLOptions.TableCountTablestatsGroupLimit
	}
}

// Files returns files map required to connect to DB. The content of a file stored in Vault is a token,
// which pmm-agent replaces with it.
func (a Agent) Files() map[string]string { //nolint:gocognit
	switch a.AgentType {
	case MySQLdExporterType, QANMySQLPerfSchemaAgentType, QANMySQLSlowlogAgentType:
		files := make(map[string]string)
		if a.MySQLOptions.TLSCa != "" {
			files[TLSCaFileName] = vault.TokenForRef(a.MySQLOptions.TLSCa, vault.EncodingRaw)
		}
		if a.MySQLOptions.TLSCert != "" {
			files[TLSCertFileName] = vault.TokenForRef(a.MySQLOptions.TLSCert, vault.EncodingRaw)
		}
		if a.MySQLOptions.TLSKey != "" {
			files[TLSKeyFileName] = vault.TokenForRef(a.MySQLOptions.TLSKey, vault.EncodingRaw)
		}

		if len(files) != 0 {
			return files
		}

		return nil
	case ProxySQLExporterType:
		return nil
	case QANMongoDBProfilerAgentType, QANMongoDBMongologAgentType, MongoDBExporterType, RTAMongoDBAgentType:
		files := make(map[string]string)
		if a.MongoDBOptions.TLSCa != "" {
			files[caFilePlaceholder] = vault.TokenForRef(a.MongoDBOptions.TLSCa, vault.EncodingRaw)
		}
		if a.MongoDBOptions.TLSCertificateKey != "" {
			files[certificateKeyFilePlaceholder] = vault.TokenForRef(a.MongoDBOptions.TLSCertificateKey, vault.EncodingRaw)
		}

		if len(files) != 0 {
			return files
		}

		return nil
	case PostgresExporterType, QANPostgreSQLPgStatementsAgentType, QANPostgreSQLPgStatMonitorAgentType:
		files := make(map[string]string)

		if a.PostgreSQLOptions.SSLCa != "" {
			files[caFilePlaceholder] = vault.TokenForRef(a.PostgreSQLOptions.SSLCa, vault.EncodingRaw)
		}
		if a.PostgreSQLOptions.SSLCert != "" {
			files[certificateFilePlaceholder] = vault.TokenForRef(a.PostgreSQLOptions.SSLCert, vault.EncodingRaw)
		}
		if a.PostgreSQLOptions.SSLKey != "" {
			files[certificateKeyFilePlaceholder] = vault.TokenForRef(a.PostgreSQLOptions.SSLKey, vault.EncodingRaw)
		}

		if len(files) != 0 {
			return files
		}

		return nil
	case ValkeyExporterType:
		// Nothing consumes the material over a plaintext link, so keep the private key off
		// the agent host in that case.
		if !a.TLS {
			return nil
		}

		files := make(map[string]string)

		if a.ValkeyOptions.SSLCa != "" {
			files[TLSCaFileName] = vault.TokenForRef(a.ValkeyOptions.SSLCa, vault.EncodingRaw)
		}
		// valkey_exporter calls log.Fatal on half a client key pair and the connection
		// check cannot use one either, so the pair only ships as a unit. Registration rejects
		// a half pair, so this only guards rows written before that validation existed.
		if a.ValkeyOptions.SSLCert != "" && a.ValkeyOptions.SSLKey != "" {
			files[TLSCertFileName] = vault.TokenForRef(a.ValkeyOptions.SSLCert, vault.EncodingRaw)
			files[TLSKeyFileName] = vault.TokenForRef(a.ValkeyOptions.SSLKey, vault.EncodingRaw)
		}

		if len(files) != 0 {
			return files
		}

		return nil
	default:
		panic(fmt.Errorf("unhandled AgentType %q", a.AgentType))
	}
}

// TemplateDelimiters returns a pair of safe template delimiters that are not present in agent parameters.
func (a Agent) TemplateDelimiters(svc *Service) *DelimiterPair {
	templateParams := []string{
		pointer.GetString(svc.Address),
		pointer.GetString(a.Username),
		pointer.GetString(a.Password),
		pointer.GetString(a.checkUsername),
		pointer.GetString(a.checkPassword),
		a.ExporterOptions.MetricsPath,
	}

	switch svc.ServiceType {
	case MySQLServiceType:
		if a.MySQLOptions.TLSKey != "" {
			templateParams = append(templateParams, a.MySQLOptions.TLSKey)
		}
	case MongoDBServiceType:
		if a.MongoDBOptions.TLSCertificateKeyFilePassword != "" {
			templateParams = append(templateParams, a.MongoDBOptions.TLSCertificateKeyFilePassword)
		}
	case PostgreSQLServiceType:
		if a.PostgreSQLOptions.SSLKey != "" {
			templateParams = append(templateParams, a.PostgreSQLOptions.SSLKey)
		}
	case ValkeyServiceType:
		// pmm-agent renders every text file's content as a template, so all three
		// certificates have to be considered, not just the private key.
		if a.ValkeyOptions.SSLCa != "" {
			templateParams = append(templateParams, a.ValkeyOptions.SSLCa)
		}
		if a.ValkeyOptions.SSLCert != "" {
			templateParams = append(templateParams, a.ValkeyOptions.SSLCert)
		}
		if a.ValkeyOptions.SSLKey != "" {
			templateParams = append(templateParams, a.ValkeyOptions.SSLKey)
		}
	case ProxySQLServiceType:
	case HAProxyServiceType:
	case ExternalServiceType:
	}

	return new(TemplateDelimsPair(
		templateParams...,
	))
}

// HashPassword func to calculate password hash. Public and overridable for testing purposes.
var HashPassword = func(password, salt string) (string, error) {
	buf, err := bcrypt.GenerateFromPasswordAndSalt([]byte(password), bcrypt.DefaultCost, []byte(salt))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md
const webConfigTemplate = `basic_auth_users:
    pmm: {{ . }}
`

// BuildWebConfigFile builds prometheus-compatible basic auth configuration.
func (a *Agent) BuildWebConfigFile() (string, error) {
	// If not provided by the user, it is the `agent_id`.
	password := a.GetAgentPassword()
	salt := getPasswordSalt(a)

	hashedPassword, err := HashPassword(password, salt)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	var configBuffer bytes.Buffer
	tmpl, err := template.New("webConfig").Parse(webConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse webconfig template: %w", err)
	}
	err = tmpl.Execute(&configBuffer, hashedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to execute webconfig template: %w", err)
	}

	config := configBuffer.String()

	return config, nil
}

func getPasswordSalt(a *Agent) string {
	if a.AgentID != "" && len(a.AgentID) >= bcrypt.MaxSaltSize {
		return a.AgentID[len(a.AgentID)-bcrypt.MaxSaltSize:]
	}

	return "pmm-salt-magic--"
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Agent)(nil)
	_ reform.BeforeUpdater  = (*Agent)(nil)
	_ reform.AfterFinder    = (*Agent)(nil)
)
