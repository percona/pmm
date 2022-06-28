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

package inventory

import "github.com/alecthomas/units"

type InventoryCommand struct {
	List   ListCommand   `cmd:"" hidden:"" help:"List inventory commands"`
	Add    AddCommand    `cmd:"" hidden:"" help:"Add to inventory commands"`
	Remove RemoveCommand `cmd:"" hidden:"" help:"Remove from inventory commands"`
}

type ListCommand struct {
	Agents   ListAgentsCommand   `cmd:"" help:"Show agents in inventory"`
	Nodes    ListNodesCommand    `cmd:"" help:"Show nodes in inventory"`
	Services ListServicesCommand `cmd:"" help:"Show services in inventory"`
}

type ListServicesCommand struct {
	NodeID        string `help:"Filter by Node identifier"`
	ServiceType   string `help:"Filter by Service type"`
	ExternalGroup string `help:"Filter by external group"`
}

type ListNodesCommand struct {
	NodeType string `help:"Filter by Node type"`
}

type ListAgentsCommand struct {
	PMMAgentID string `help:"Filter by pmm-agent identifier"`
	ServiceID  string `help:"Filter by Service identifier"`
	NodeID     string `help:"Filter by Node identifier"`
	AgentType  string `help:"Filter by Agent type"`
}

type AddCommand struct {
	Agent   AddAgentCommand   `cmd:"" help:"Add agent to inventory"`
	Node    AddNodeCommand    `cmd:"" help:"Add node to inventory"`
	Service AddServiceCommand `cmd:"" help:"Add service to inventory"`
}

type AddServiceCommand struct {
	External   AddServiceExternalCommand   `cmd:"" help:"Add an external service to inventory"`
	HAProxy    AddServiceHAProxyCommand    `cmd:"" name:"haproxy" help:"Add HAProxy service to inventory"`
	MongoDB    AddServiceMongoDBCommand    `cmd:"" name:"mongodb" help:"Add MongoDB service to inventory"`
	MySQL      AddServiceMySQLCommand      `cmd:"" name:"mysql" help:"Add MySQL service to inventory"`
	PostgreSQL AddServicePostgreSQLCommand `cmd:"" name:"postgresql" help:"Add PostgreSQL service to inventory"`
	ProxySQL   AddServiceProxySQLCommand   `cmd:"" name:"proxysql" help:"Add ProxySQL service to inventory"`
}

type AddServiceProxySQLCommand struct {
	ServiceName    string `arg:"" optional:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" optional:"" help:"Node ID"`
	Address        string `arg:"" optional:"" help:"Address"`
	Port           int64  `arg:"" optional:"" help:"Port"`
	Socket         string `help:"Path to ProxySQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServicePostgreSQLCommand struct {
	ServiceName    string `arg:"" optional:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" optional:"" help:"Node ID"`
	Address        string `arg:"" optional:"" help:"Address"`
	Port           int64  `arg:"" optional:"" help:"Port"`
	Socket         string `help:"Path to PostgreSQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceMySQLCommand struct {
	ServiceName    string `arg:"" optional:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" optional:"" help:"Node ID"`
	Address        string `arg:"" optional:"" help:"Address"`
	Port           int64  `arg:"" optional:"" help:"Port"`
	Socket         string `help:"Path to MySQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceMongoDBCommand struct {
	ServiceName    string `arg:"" optional:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" optional:"" help:"Node ID"`
	Address        string `arg:"" optional:"" help:"Address"`
	Port           int64  `arg:"" optional:"" help:"Port"`
	Socket         string `help:"Path to socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceHAProxyCommand struct {
	ServiceName    string `arg:"" optional:"" name:"name" help:"HAProxy service name"`
	NodeID         string `arg:"" optional:"" help:"HAProxy service node ID"`
	Environment    string `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster        string `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet string `placeholder:"rs1" help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels. Example: region=east,app=app1"`
}

type AddServiceExternalCommand struct {
	ServiceName    string `name:"name" required:"" help:"External service name. Required"`
	NodeID         string `required:"" help:"External service node ID. Required"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
	Group          string `help:"Group name of external service"`
}

type AddNodeCommand struct {
	Container AddNodeContainerCommand `cmd:"" help:"Add container node to inventory"`
	Generic   AddNodeGenericCommand   `cmd:"" help:"Add generic node to inventory"`
	Remote    AddNodeRemoteCommand    `cmd:"" help:"Add Remote node to inventory"`
	RemoteRDS AddNodeRemoteRDSCommand `cmd:"" help:"Add Remote RDS node to inventory"`
}

type AddNodeRemoteCommand struct {
	NodeName     string `arg:"" optional:"" name:"name" help:"Node name"`
	Address      string `help:"Address"`
	CustomLabels string `help:"Custom user-assigned labels"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
}

type AddNodeRemoteRDSCommand struct {
	NodeName     string `arg:"" optional:"" name:"name" help:"Node name"`
	Address      string `help:"Address"`
	NodeModel    string `help:"Node mddel"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
	CustomLabels string `help:"Custom user-assigned labels"`
}

type AddNodeGenericCommand struct {
	NodeName     string `arg:"" optional:"" name:"name" help:"Node name"`
	MachineID    string `help:"Linux machine-id"`
	Distro       string `help:"Linux distribution (if any)"`
	Address      string `help:"Address"`
	CustomLabels string `help:"Custom user-assigned labels"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
	NodeModel    string `help:"Node mddel"`
}

type AddNodeContainerCommand struct {
	NodeName      string `arg:"" optional:"" name:"name" help:"Node name"`
	MachineID     string `help:"Linux machine-id"`
	ContainerID   string `help:"Container identifier; if specified, must be a unique Docker container identifier"`
	ContainerName string `help:"Container name"`
	Address       string `help:"Address"`
	CustomLabels  string `help:"Custom user-assigned labels"`
	Region        string `help:"Node region"`
	Az            string `help:"Node availability zone"`
	NodeModel     string `help:"Node model"`
}

type AddAgentCommand struct {
	ExternalExporter ExternalExporterCommand `cmd:"" name:"external" help:"Add external exporter to inventory"`
	MongodbExporter  MongoDBExporterCommand  `cmd:"" help:"Add mongodb_exporter to inventory"`
	MysqldExporter   MysqldExporterCommand   `cmd:"" help:"Add mysqld_exporter to inventory"`
	NodeExporter     NodeExporterCommand     `cmd:"" help:"Add Node exporter to inventory"`
	PMMAgent         PMMAgentCommand         `cmd:"" help:"Add PMM agent to inventory"`
	PostgresExporter PostgresExporterCommand `cmd:"" help:"Add postgres_exporter to inventory"`
	ProxysqlExporter ProxysqlExporterCommand `cmd:"" help:"Add proxysql_exporter to inventory"`

	QANMongoDBProfilerAgent         AddQANMongoDBProfilerAgentCommand         `cmd:"" name:"qan-mongodb-profiler-agent" help:"Add QAN MongoDB profiler agent to inventory"`
	QANMySQLPerfSchemaAgent         AddQANMySQLPerfSchemaAgentCommand         `cmd:"" name:"qan-mysql-perfschema-agent" help:"Add QAN MySQL perf schema agent to inventory"`
	QANMySQLSlowlogAgent            AddQANMySQLSlowlogAgentCommand            `cmd:"" name:"qan-mysql-slowlog-agent" help:"Add QAN MySQL slowlog agent to inventory"`
	QANPostgreSQLPgStatementsAgent  AddQANPostgreSQLPgStatementsAgentCommand  `cmd:"" name:"qan-postgresql-pgstatements-agent" help:"Add QAN PostgreSQL Stat Statements Agent to inventory"`
	QANPostgreSQLPgStatMonitorAgent AddQANPostgreSQLPgStatMonitorAgentCommand `cmd:"" name:"qan-postgresql-pgstatmonitor-agent" help:"Add QAN PostgreSQL Stat Monitor Agent to inventory"`

	RDSExporter AddAgentRDSExporterCmd `cmd:"" help:"Add rds_exporter to inventory"`
}

type AddAgentRDSExporterCmd struct {
	PMMAgentID             string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	NodeID                 string `arg:"" help:"Node identifier"`
	AWSAccessKey           string `help:"AWS Access Key ID"`
	AWSSecretKey           string `help:"AWS Secret Access Key"`
	CustomLabels           string `help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool   `help:"Skip connection check"`
	DisableBasicMetrics    bool   `help:"Disable basic metrics"`
	DisableEnhancedMetrics bool   `help:"Disable enhanced metrics"`
	PushMetrics            bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
}

type AddQANPostgreSQLPgStatMonitorAgentCommand struct {
	PMMAgentID            string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID             string `arg:"" help:"Service identifier"`
	Username              string `arg:"" optional:"" help:"PostgreSQL username for QAN agent"`
	Password              string `help:"PostgreSQL password for QAN agent"`
	CustomLabels          string `help:"Custom user-assigned labels"`
	SkipConnectionCheck   bool   `help:"Skip connection check"`
	QueryExamplesDisabled bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                   bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify         bool   `help:"Skip TLS certificates validation"`
	TLSCAFile             string `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile           string `help:"TLS certificate file"`
	TLSKeyFile            string `help:"TLS certificate key file"`
}

type AddQANPostgreSQLPgStatementsAgentCommand struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" optional:"" help:"PostgreSQL username for QAN agent"`
	Password            string `help:"PostgreSQL password for QAN agent"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	TLSCAFile           string `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile         string `help:"TLS certificate file"`
	TLSKeyFile          string `help:"TLS certificate key file"`
}

type AddQANMySQLSlowlogAgentCommand struct {
	PMMAgentID           string           `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID            string           `arg:"" help:"Service identifier"`
	Username             string           `arg:"" optional:"" help:"MySQL username for scraping metrics"`
	Password             string           `help:"MySQL password for scraping metrics"`
	CustomLabels         string           `help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool             `help:"Skip connection check"`
	DisableQueryExamples bool             `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MaxSlowlogFileSize   units.Base2Bytes `name:"size-slow-logs" placeholder:"size" help:"Rotate slow log file at this size (default: 0; 0 or negative value disables rotation). Ex.: 1GiB"`
	TLS                  bool             `help:"Use TLS to connect to the database"`
	TLSSkipVerify        bool             `help:"Skip TLS certificates validation"`
	TLSCAFile            string           `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile          string           `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile           string           `name:"tls-key" help:"Path to client key file"`
}

type AddQANMySQLPerfSchemaAgentCommand struct {
	PMMAgentID           string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID            string `arg:"" help:"Service identifier"`
	Username             string `arg:"" optional:"" help:"MySQL username for scraping metrics"`
	Password             string `help:"MySQL password for scraping metrics"`
	CustomLabels         string `help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool   `help:"Skip connection check"`
	DisableQueryExamples bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                  bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify        bool   `help:"Skip TLS certificates validation"`
	TLSCAFile            string `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile          string `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile           string `name:"tls-key" help:"Path to client key file"`
}

type AddQANMongoDBProfilerAgentCommand struct {
	PMMAgentID                    string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string `arg:"" help:"Service identifier"`
	Username                      string `arg:"" optional:"" help:"MongoDB username for scraping metrics"`
	Password                      string `help:"MongoDB password for scraping metrics"`
	CustomLabels                  string `help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool   `help:"Skip connection check"`
	DisableQueryExamples          bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                           bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool   `help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string `help:"Password for certificate"`
	TLSCaFile                     string `help:"Path to certificate authority file"`
	AuthenticationMechanism       string `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
}

type ProxysqlExporterCommand struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" optional:"" help:"ProxySQL username for scraping metrics"`
	Password            string `help:"ProxySQL password for scraping metrics"`
	AgentPassword       string `help:"Custom password for /metrics endpoint"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	PushMetrics         bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors   string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type PostgresExporterCommand struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" optional:"" help:"PostgreSQL username for scraping metrics"`
	Password            string `help:"PostgreSQL password for scraping metrics"`
	AgentPassword       string `help:"Custom password for /metrics endpoint"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	PushMetrics         bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors   string `help:"Comma-separated list of collector names to exclude from exporter"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	TLSCAFile           string `help:"TLS CA certificate file"`
	TLSCertFile         string `help:"TLS certificate file"`
	TLSKeyFile          string `help:"TLS certificate key file"`
}

type PMMAgentCommand struct {
	RunsOnNodeID string `arg:"" help:"Node identifier where this instance runs"`
	CustomLabels string `help:"Custom user-assigned labels"`
}

type NodeExporterCommand struct {
	PMMAgentID        string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	CustomLabels      string `help:"Custom user-assigned labels"`
	PushMetrics       bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type MysqldExporterCommand struct {
	PMMAgentID                string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                 string `arg:"" help:"Service identifier"`
	Username                  string `arg:"" optional:"" help:"MySQL username for scraping metrics"`
	Password                  string `help:"MySQL password for scraping metrics"`
	AgentPassword             string `help:"Custom password for /metrics endpoint"`
	CustomLabels              string `help:"Custom user-assigned labels"`
	SkipConnectionCheck       bool   `help:"Skip connection check"`
	TLS                       bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify             bool   `help:"Skip TLS certificates validation"`
	TLSCAFile                 string `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile               string `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile                string `name:"tls-key" help:"Path to client key file"`
	TablestatsGroupTableLimit int32  `placeholder:"number" help:"Tablestats group collectors will be disabled if there are more than that number of tables (default: 0 - always enabled; negative value - always disabled)"`
	PushMetrics               bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors         string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type MongoDBExporterCommand struct {
	PMMAgentID                    string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string `arg:"" help:"Service identifier"`
	Username                      string `arg:"" optional:"" help:"MongoDB username for scraping metrics"`
	Password                      string `help:"MongoDB password for scraping metrics"`
	AgentPassword                 string `help:"Custom password for /metrics endpoint"`
	CustomLabels                  string `help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool   `help:"Skip connection check"`
	TLS                           bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool   `help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string `help:"Password for certificate"`
	TLSCaFile                     string `help:"Path to certificate authority file"`
	AuthenticationMechanism       string `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	PushMetrics                   bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors             string `help:"Comma-separated list of collector names to exclude from exporter"`
	StatsCollections              string `help:"Collections for collstats & indexstats"`
	CollectionsLimit              int32  `name:"max-collections-limit" placeholder:"number" help:"Disable collstats & indexstats if there are more than <n> collections"`
}

type ExternalExporterCommand struct {
	RunsOnNodeID string `required:"" help:"Node identifier where this instance runs"`
	ServiceID    string `required:"" help:"Service identifier"`
	Username     string `help:"HTTP Basic auth username for scraping metrics"`
	Password     string `help:"HTTP Basic auth password for scraping metrics"`
	Scheme       string `help:"Scheme to generate URI to exporter metrics endpoints (http, https)"`
	MetricsPath  string `help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort   int64  `required:"" placeholder:"port" help:"Listen port for scraping metrics"`
	CustomLabels string `help:"Custom user-assigned labels"`
	PushMetrics  bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
}

type RemoveCommand struct {
	Agent   RemoveAgentCommand   `cmd:"" help:"Remove agent from inventory"`
	Node    RemoveNodeCommand    `cmd:"" help:"Remove node from inventory"`
	Service RemoveServiceCommand `cmd:"" help:"Remove service from inventory"`
}

type RemoveServiceCommand struct {
	ServiceID string `arg:"" optional:"" help:"Service ID"`
	Force     bool   `help:"Remove service with all dependencies"`
}

type RemoveNodeCommand struct {
	NodeID string `arg:"" optional:"" help:"Node ID"`
	Force  bool   `help:"Remove node with all dependencies"`
}

type RemoveAgentCommand struct {
	AgentID string `arg:"" optional:"" help:"Agent ID"`
	Force   bool   `help:"Remove agent with all dependencies"`
}
