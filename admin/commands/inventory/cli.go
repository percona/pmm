package inventory

import "github.com/alecthomas/units"

type InventoryCmd struct {
	List   ListCmd   `cmd:"" hidden:"" help:"List inventory commands"`
	Add    AddCmd    `cmd:"" hidden:"" help:"Add to inventory commands"`
	Remove RemoveCmd `cmd:"" hidden:"" help:"Remove from inventory commands"`
}

type ListCmd struct {
	Services ListServicesCmd `cmd:"" hidden:"" help:"Show services in inventory"`
	Nodes    ListNodesCmd    `cmd:"" hidden:"" help:"Show nodes in inventory"`
	Agents   ListAgentsCmd   `cmd:"" hidden:"" help:"Show agents in inventory"`
}

type ListServicesCmd struct {
	NodeID        string `help:"Filter by Node identifier"`
	ServiceType   string `help:"Filter by Service type"`
	ExternalGroup string `help:"Filter by external group"`
}

type ListNodesCmd struct {
	NodeType string `help:"Filter by Node type"`
}

type ListAgentsCmd struct {
	PMMAgentId string `help:"Filter by pmm-agent identifier"`
	ServiceID  string `help:"Filter by Service identifier"`
	NodeID     string `help:"Filter by Node identifier"`
	AgentType  string `help:"Filter by Agent type"`
}

type AddCmd struct {
	Service AddServiceCmd `cmd:"" hidden:"" help:"Add service to inventory"`
	Node    AddNodeCmd    `cmd:"" hidden:"" help:"Add node to inventory"`
	Agent   AddAgentCmd   `cmd:"" hidden:"" help:"Add agent to inventory"`
}

type AddServiceCmd struct {
	ProxySQL   AddServiceProxySQLCmd   `cmd:"" hidden:"" name:"proxysql" help:"Add ProxySQL service to inventory"`
	PostgreSQL AddServicePostgreSQLCmd `cmd:"" hidden:"" name:"postgresql" help:"Add PostgreSQL service to inventory"`
	MySQL      AddServiceMySQLCmd      `cmd:"" hidden:"" name:"mysql" help:"Add MySQL service to inventory"`
	MongoDB    AddServiceMongoDBCmd    `cmd:"" hidden:"" name:"mongodb" help:"Add MongoDB service to inventory"`
	HAProxy    AddServiceHAProxyCmd    `cmd:"" hidden:"" name:"haproxy" help:"Add HAProxy service to inventory"`
	External   AddServiceExternalCmd   `cmd:"" hidden:"" name:"haproxy" help:"Add an external service to inventory"`
}

type AddServiceProxySQLCmd struct {
	ServiceName    string `arg:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" help:"Node ID"`
	Address        string `arg:"" help:"Address"`
	Port           int64  `arg:"" help:"Port"`
	Socket         string `help:"Path to ProxySQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServicePostgreSQLCmd struct {
	ServiceName    string `arg:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" help:"Node ID"`
	Address        string `arg:"" help:"Address"`
	Port           int64  `arg:"" help:"Port"`
	Socket         string `help:"Path to PostgreSQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceMySQLCmd struct {
	ServiceName    string `arg:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" help:"Node ID"`
	Address        string `arg:"" help:"Address"`
	Port           int64  `arg:"" help:"Port"`
	Socket         string `help:"Path to MySQL socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceMongoDBCmd struct {
	ServiceName    string `arg:"" name:"name" help:"Service name"`
	NodeID         string `arg:"" help:"Node ID"`
	Address        string `arg:"" help:"Address"`
	Port           int64  `arg:"" help:"Port"`
	Socket         string `help:"Path to socket"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
}

type AddServiceHAProxyCmd struct {
	ServiceName    string `arg:"" name:"name" help:"HAProxy service name"`
	NodeID         string `arg:"" help:"HAProxy service node ID"`
	Environment    string `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster        string `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet string `placeholder:"rs1" help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels. Example: region=east,app=app1"`
}

type AddServiceExternalCmd struct {
	ServiceName    string `name:"name" required:"" help:"External service name. Required"`
	NodeID         string `required:"" help:"External service node ID. Required"`
	Environment    string `help:"Environment name"`
	Cluster        string `help:"Cluster name"`
	ReplicationSet string `help:"Replication set name"`
	CustomLabels   string `help:"Custom user-assigned labels"`
	Group          string `help:"Group name of external service"`
}

type AddNodeCmd struct {
	Remote    AddNodeRemoteCmd    `cmd:"" hidden:"" help:"Add Remote node to inventory"`
	RemoteRDS AddNodeRemoteRDSCmd `cmd:"" hidden:"" help:"Add Remote RDS node to inventory"`
	Generic   AddNodeGenericCmd   `cmd:"" hidden:"" help:"Add generic node to inventory"`
	Container AddNodeContainerCmd `cmd:"" hidden:"" help:"Add container node to inventory"`
}

type AddNodeRemoteCmd struct {
	NodeName     string `arg:"" name:"name" help:"Node name"`
	Address      string `help:"Address"`
	CustomLabels string `help:"Custom user-assigned labels"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
}

type AddNodeRemoteRDSCmd struct {
	NodeName     string `arg:"" name:"name" help:"Node name"`
	Address      string `help:"Address"`
	NodeModel    string `name:"name" help:"Node mddel"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
	CustomLabels string `help:"Custom user-assigned labels"`
}

type AddNodeGenericCmd struct {
	NodeName     string `arg:"" name:"name" help:"Node name"`
	MachineID    string `help:"Linux machine-id"`
	Distro       string `help:"Linux distribution (if any)"`
	Address      string `help:"Address"`
	CustomLabels string `help:"Custom user-assigned labels"`
	Region       string `help:"Node region"`
	Az           string `help:"Node availability zone"`
	NodeModel    string `name:"name" help:"Node mddel"`
}

type AddNodeContainerCmd struct {
	NodeName      string `arg:"" name:"name" help:"Node name"`
	MachineID     string `help:"Linux machine-id"`
	ContainerID   string `help:"Container identifier; if specified, must be a unique Docker container identifier"`
	ContainerName string `help:"Container name"`
	Address       string `help:"Address"`
	CustomLabels  string `help:"Custom user-assigned labels"`
	Region        string `help:"Node region"`
	Az            string `help:"Node availability zone"`
	NodeModel     string `name:"name" help:"Node mddel"`
}

type AddAgentCmd struct {
	RDSExporter AddAgentRDSExporterCmd `cmd:"" hidden:"" help:"Add rds_exporter to inventory"`

	QANPostgreSQLPgStatMonitorAgent AddQANPostgreSQLPgStatMonitorAgentCmd `cmd:"" hidden:"" name:"qan-postgresql-pgstatmonitor-agent" help:"Add QAN PostgreSQL Stat Monitor Agent to inventory"`
	QANPostgreSQLPgStatementsAgent  AddQANPostgreSQLPgStatementsAgentCmd  `cmd:"" hidden:"" name:"qan-postgresql-pgstatements-agent" help:"Add QAN PostgreSQL Stat Statements Agent to inventory"`
	QANMySQLSlowlogAgent            AddQANMySQLSlowlogAgentCmd            `cmd:"" hidden:"" name:"qan-mysql-slowlog-agent" help:"Add QAN MySQL slowlog agent to inventory"`
	QANMySQLPerfSchemaAgent         AddQANMySQLPerfSchemaAgentCmd         `cmd:"" hidden:"" name:"qan-mysql-perfschema-agent" help:"Add QAN MySQL perf schema agent to inventory"`
	QANMongoDBProfilerAgent         AddQANMongoDBProfilerAgentCmd         `cmd:"" hidden:"" name:"qan-mongodb-profiler-agent" help:"Add QAN MongoDB profiler agent to inventory"`

	PostgresExporter PostgresExporterCmd `cmd:"" hidden:"" help:"Add postgres_exporter to inventory"`
	PMMAgent         PMMAgentCmd         `cmd:"" hidden:"" help:"Add PMM agent to inventory"`
	NodeExporter     NodeExporterCmd     `cmd:"" hidden:"" help:"Add Node exporter to inventory"`
	MysqldExporter   MysqldExporterCmd   `cmd:"" hidden:"" help:"Add mysqld_exporter to inventory"`
	MongodbExporter  MongoDBExporterCmd  `cmd:"" hidden:"" help:"Add mongodb_exporter to inventory"`
	ExternalExporter ExternalExporterCmd `cmd:"" hidden:"" help:"Add external exporter to inventory"`
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

type AddQANPostgreSQLPgStatMonitorAgentCmd struct {
	PMMAgentID            string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID             string `arg:"" help:"Service identifier"`
	Username              string `arg:"" help:"PostgreSQL username for QAN agent"`
	Password              string `help:"PostgreSQL password for QAN agent"`
	CustomLabels          string `help:"Custom user-assigned labels"`
	SkipConnectionCheck   bool   `help:"Skip connection check"`
	QueryExamplesDisabled bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                   bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify         bool   `help:"Skip TLS certificates validation"`
	TLSCAFile             string `help:"TLS CA certificate file"`
	TLSCertFile           string `help:"TLS certificate file"`
	TLSKeyFile            string `help:"TLS certificate key file"`
}

type AddQANPostgreSQLPgStatementsAgentCmd struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" help:"PostgreSQL username for QAN agent"`
	Password            string `help:"PostgreSQL password for QAN agent"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	TLSCAFile           string `help:"TLS CA certificate file"`
	TLSCertFile         string `help:"TLS certificate file"`
	TLSKeyFile          string `help:"TLS certificate key file"`
}

type AddQANMySQLSlowlogAgentCmd struct {
	PMMAgentID           string           `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID            string           `arg:"" help:"Service identifier"`
	Username             string           `arg:"" help:"MySQL username for scraping metrics"`
	Password             string           `help:"MySQL password for scraping metrics"`
	CustomLabels         string           `help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool             `help:"Skip connection check"`
	DisableQueryExamples bool             `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MaxSlowlogFileSize   units.Base2Bytes `name:"size-slow-logs" help:"Rotate slow log file at this size (default: 0; 0 or negative value disables rotation). Ex.: 1GiB"`
	TLS                  bool             `help:"Use TLS to connect to the database"`
	TLSSkipVerify        bool             `help:"Skip TLS certificates validation"`
	TLSCAFile            string           `help:"TLS CA certificate file"`
	TLSCertFile          string           `help:"TLS certificate file"`
	TLSKeyFile           string           `help:"TLS certificate key file"`
}

type AddQANMySQLPerfSchemaAgentCmd struct {
	PMMAgentID           string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID            string `arg:"" help:"Service identifier"`
	Username             string `arg:"" help:"MySQL username for scraping metrics"`
	Password             string `help:"MySQL password for scraping metrics"`
	CustomLabels         string `help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool   `help:"Skip connection check"`
	DisableQueryExamples bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                  bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify        bool   `help:"Skip TLS certificates validation"`
	TLSCAFile            string `help:"TLS CA certificate file"`
	TLSCertFile          string `help:"TLS certificate file"`
	TLSKeyFile           string `help:"TLS certificate key file"`
}

type AddQANMongoDBProfilerAgentCmd struct {
	PMMAgentID                    string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string `arg:"" help:"Service identifier"`
	Username                      string `arg:"" help:"MongoDB username for scraping metrics"`
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

type ProxysqlExporterCmd struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" help:"ProxySQL username for scraping metrics"`
	Password            string `help:"ProxySQL password for scraping metrics"`
	AgentPassword       string `help:"Custom password for /metrics endpoint"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	PushMetrics         bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors   string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type PostgresExporterCmd struct {
	PMMAgentID          string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string `arg:"" help:"Service identifier"`
	Username            string `arg:"" help:"PostgreSQL username for scraping metrics"`
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

type PMMAgentCmd struct {
	RunsOnNodeID string `arg:"" help:"Node identifier where this instance runs"`
	CustomLabels string `help:"Custom user-assigned labels"`
}

type NodeExporterCmd struct {
	PMMAgentID        string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	CustomLabels      string `help:"Custom user-assigned labels"`
	PushMetrics       bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type MysqldExporterCmd struct {
	PMMAgentID                string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                 string `arg:"" help:"Service identifier"`
	Username                  string `arg:"" help:"MySQL username for scraping metrics"`
	Password                  string `help:"MySQL password for scraping metrics"`
	AgentPassword             string `help:"Custom password for /metrics endpoint"`
	CustomLabels              string `help:"Custom user-assigned labels"`
	SkipConnectionCheck       bool   `help:"Skip connection check"`
	TLS                       bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify             bool   `help:"Skip TLS certificates validation"`
	TLSCAFile                 string `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile               string `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile                string `name:"tls-key" help:"Path to client key file"`
	TablestatsGroupTableLimit int32  `help:"Tablestats group collectors will be disabled if there are more than that number of tables (default: 0 - always enabled; negative value - always disabled)"`
	PushMetrics               bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors         string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type MongoDBExporterCmd struct {
	PMMAgentID                    string `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string `arg:"" help:"Service identifier"`
	Username                      string `arg:"" help:"MongoDB username for scraping metrics"`
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
	CollectionsLimit              int32  `name:"max-collections-limit" help:"Disable collstats & indexstats if there are more than <n> collections"`
}

type ExternalExporterCmd struct {
	RunsOnNodeID string `required:"" help:"Node identifier where this instance runs"`
	ServiceID    string `required:"" help:"Service identifier"`
	Username     string `help:"HTTP Basic auth username for scraping metrics"`
	Password     string `help:"HTTP Basic auth password for scraping metrics"`
	Scheme       string `help:"Scheme to generate URI to exporter metrics endpoints (http, https)"`
	MetricsPath  string `help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort   int64  `required:"" help:"Listen port for scraping metrics"`
	CustomLabels string `help:"Custom user-assigned labels"`
	PushMetrics  bool   `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
}

type RemoveCmd struct {
	Service RemoveServiceCmd `cmd:"" hidden:"" help:"Remove service from inventory"`
	Node    RemoveNodeCmd    `cmd:"" hidden:"" help:"Remove node from inventory"`
	Agent   RemoveAgentCmd   `cmd:"" hidden:"" help:"Remove agent from inventory"`
}

type RemoveServiceCmd struct {
	ServiceID string `help:"Service ID"`
	Force     bool   `help:"Remove service with all dependencies"`
}

type RemoveNodeCmd struct {
	NodeID string `help:"Node ID"`
	Force  bool   `help:"Remove node with all dependencies"`
}

type RemoveAgentCmd struct {
	AgentID string `help:"Agent ID"`
	Force   bool   `help:"Remove agent with all dependencies"`
}
