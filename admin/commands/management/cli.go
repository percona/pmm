package management

import "github.com/alecthomas/units"

type UnregisterCmd struct {
	Force    bool   `name:"force" help:"Remove this node with all dependencies"`
	NodeName string `name:"node-name" help:"Node name (autodetected default: ${hostname})"`
}

type RemoveCmd struct {
	ServiceType string `name:"service-type" arg:"" enum:"${serviceTypesEnum}" help:"Service type, one of: ${serviceTypesEnum}"`
	ServiceName string `name:"service-name" arg:"" default:"" help:"Service name"`
	ServiceID   string `name:"service-id" help:"Service ID"`
}

type RegisterCmd struct {
	Address           string `name:"node-address" arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string `name:"node-type" arg:"" enum:"generic,container" default:"generic" help:"Node type, one of: generic, container (default: generic)"`
	NodeName          string `name:"node-name" arg:"" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	MachineID         string `name:"machine-id" default:"${defaultMachineID}" help:"Node machine-id (default is autodetected)"`
	Distro            string `name:"distro" default:"${distro}" help:"Node OS distribution (default is autodetected)"`
	ContainerID       string `name:"container-id" help:"Container ID"`
	ContainerName     string `name:"container-name" help:"Container name"`
	NodeModel         string `name:"node-model" help:"Node model"`
	Region            string `name:"region" help:"Node region"`
	Az                string `name:"az" help:"Node availability zone"`
	CustomLabels      string `name:"custom-labels" help:"Custom user-assigned labels"`
	AgentPassword     string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	Force             bool   `name:"force" help:"Re-register Node"`
	MetricsMode       string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors string `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`
}

type AddCmd struct {
	External           AddExternalCmd           `cmd:"" name:"external" help:"Add External source of data (like a custom exporter running on a port) to the monitoring"`
	ExternalServerless AddExternalServerlessCmd `cmd:"" name:"external-serverless" help:"Add External Service on Remote node to monitoring."`
	HAProxy            AddHAProxyCmd            `cmd:"" name:"haproxy" help:"Add HAProxy to monitoring"`
	MongoDb            AddMongoDbCmd            `cmd:"" name:"mongodb" help:"Add MongoDB to monitoring"`
	MySql              AddMySqlCmd              `cmd:"" name:"mysql" help:"Add MySQL to monitoring"`
	PostgreSql         AddPostgreSqlCmd         `cmd:"" name:"postgresql" help:"Add PostgreSQL to monitoring"`
	ProxySql           AddProxySqlCmd           `cmd:"" name:"proxysql" help:"Add ProxySQL to monitoring"`
}

type AddCommonFlags struct {
	AddServiceNameFlag string `name:"service-name" placeholder:"NAME" help:"Service name (overrides positional argument)"`
	AddHostFlag        string `name:"host" placeholder:"HOST" help:"Service hostname or IP address (overrides positional argument)"`
	AddPortFlag        uint16 `name:"port" placeholder:"PORT" help:"Service port number (overrides positional argument)"`
	AddLogLevel        string `name:"log-level" enum:"debug,info,warn,error,fatal" default:"warn" help:"Service logging level"`
}

type AddProxySqlCmd struct {
	ServiceName         string `name:"name" arg:"" default:"${hostname}-proxysql" help:"Service name (autodetected default: ${hostname}-proxysql)"`
	Address             string `name:"address" arg:"" default:"127.0.0.1:6032" help:"ProxySQL address and port (default: 127.0.0.1:6032)"`
	Socket              string `name:"socket" help:"Path to ProxySQL socket"`
	NodeID              string `name:"node-id" help:"Node ID (default is autodetected)"`
	PMMAgentID          string `name:"pmm-agent-id" help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username            string `name:"username" default:"admin" help:"ProxySQL username"`
	Password            string `name:"password" default:"admin" help:"ProxySQL password"`
	AgentPassword       string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	CredentialsSource   string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	Environment         string `name:"environment" help:"Environment name"`
	Cluster             string `name:"cluster" help:"Cluster name"`
	ReplicationSet      string `name:"replication-set" help:"Replication set name"`
	CustomLabels        string `name:"custom-labels" help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `name:"skip-connection-check" help:"Skip connection check"`
	TLS                 bool   `name:"tls" help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `name:"tls-skip-verify" help:"Skip TLS certificates validation"`
	MetricsMode         string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors   string `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddPostgreSqlCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-postgresql" help:"Service name (autodetected default: ${hostname}-postgresql)"`
	Address           string `name:"address" arg:"" default:"127.0.0.1:5432" help:"PostgreSQL address and port (default: 127.0.0.1:5432)"`
	Socket            string `name:"socket" help:"Path to socket"`
	Username          string `name:"username" default:"postgres" help:"PostgreSQL username"`
	Password          string `name:"password" help:"PostgreSQL password"`
	Database          string `name:"database" help:"PostgreSQL database"`
	AgentPassword     string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	CredentialsSource string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	NodeID            string `name:"node-id" help:"Node ID (default is autodetected)"`
	PMMAgentID        string `name:"pmm-agent-id" help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	// TODO add "auto"
	QuerySource          string `name:"query-source" default:"pgstatements" help:"Source of SQL queries, one of: pgstatements, pgstatmonitor, none (default: pgstatements)"`
	Environment          string `name:"environment" help:"Environment name"`
	Cluster              string `name:"cluster" help:"Cluster name"`
	ReplicationSet       string `name:"replication-set" help:"Replication set name"`
	CustomLabels         string `name:"custom-labels" help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool   `name:"skip-connection-check" help:"Skip connection check"`
	TLS                  bool   `name:"tls" help:"Use TLS to connect to the database"`
	TLSCAFile            string `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile          string `name:"tls-cert-file" help:"TLS certificate file"`
	TLSKeyFile           string `name:"tls-key-file" help:"TLS certificate key file"`
	TLSSkipVerify        bool   `name:"tls-skip-verify" help:"Skip TLS certificates validation"`
	DisableQueryExamples bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MetricsMode          string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors    string `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddMySqlCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-mysql" help:"Service name (autodetected default: ${hostname}-mysql)"`
	Address           string `name:"address" arg:"" default:"127.0.0.1:3306" help:"MySQL address and port (default: 127.0.0.1:3306)"`
	Socket            string `name:"socket" help:"Path to MySQL socket"`
	NodeID            string `name:"node-id" help:"Node ID (default is autodetected)"`
	PMMAgentID        string `name:"pmm-agent-id" help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username          string `name:"username" help:"MySQL username"`
	Password          string `name:"password" help:"MySQL password"`
	DefaultsFile      string `name:"defaults-file" help:"Path to defaults file"`
	AgentPassword     string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	CredentialsSource string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	// TODO add "auto", make it default
	QuerySource          string `name:"query-source" default:"${mysqlQuerySourceDefault}" enum:"${mysqlQuerySourcesEnum}" help:"Source of SQL queries, one of: ${mysqlQuerySourcesEnum} (default: ${mysqlQuerySourceDefault})"`
	DisableQueryExamples bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	// TODO check if works
	MaxSlowlogFileSize     units.Base2Bytes `name:"size-slow-logs" placeholder:"size" help:"Rotate slow log file at this size (default: server-defined; negative value disables rotation). Ex.: 1GiB"`
	DisableTablestats      bool             `name:"disable-tablestats" help:"Disable table statistics collection"`
	DisableTablestatsLimit uint16           `name:"disable-tablestats-limit" help:"Table statistics collection will be disabled if there are more than specified number of tables (default: server-defined)"`
	Environment            string           `name:"environment" help:"Environment name"`
	Cluster                string           `name:"cluster" help:"Cluster name"`
	ReplicationSet         string           `name:"replication-set" help:"Replication set name"`
	CustomLabels           string           `name:"custom-labels" help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool             `name:"skip-connection-check" help:"Skip connection check"`
	TLS                    bool             `name:"tls" help:"Use TLS to connect to the database"`
	TLSSkipVerify          bool             `name:"tls-skip-verify" help:"Skip TLS certificates validation"`
	TLSCaFile              string           `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile            string           `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile             string           `name:"tls-key" help:"Path to client key file"`
	CreateUser             bool             `name:"create-user" hidden:"" help:"Create pmm user"`
	MetricsMode            string           `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors      string           `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddMongoDbCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-mongodb" help:"Service name (autodetected default: ${hostname}-mongodb)"`
	Address           string `name:"address" arg:"" default:"127.0.0.1:27017" help:"MongoDB address and port (default: 127.0.0.1:27017)"`
	Socket            string `name:"socket" help:"Path to socket"`
	NodeID            string `name:"node-id" help:"Node ID (default is autodetected)"`
	PMMAgentID        string `name:"pmm-agent-id" help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username          string `name:"username" help:"MongoDB username"`
	Password          string `name:"password" help:"MongoDB password"`
	AgentPassword     string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	CredentialsSource string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	// TODO add "auto"
	QuerySource                   string `name:"query-source" default:"${mongoDbQuerySourceDefault}" enum:"${mongoDbQuerySourcesEnum}" help:"Source of queries, one of: ${mongoDbQuerySourcesEnum} (default: ${mongoDbQuerySourceDefault})"`
	Environment                   string `name:"environment" help:"Environment name"`
	Cluster                       string `name:"cluster" help:"Cluster name"`
	ReplicationSet                string `name:"replication-set" help:"Replication set name"`
	CustomLabels                  string `name:"custom-labels" help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool   `name:"skip-connection-check" help:"Skip connection check"`
	TLS                           bool   `name:"tls" help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool   `name:"tls-skip-verify" help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string `name:"tls-certificate-key-file" help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string `name:"tls-certificate-key-file-password" help:"Password for certificate"`
	TLSCaFile                     string `name:"tls-ca-file" help:"Path to certificate authority file"`
	AuthenticationMechanism       string `name:"authentication-mechanism" help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	AuthenticationDatabase        string `name:"authentication-database" help:"Authentication database. Default is empty. Use $external for ssl certificates"`
	MetricsMode                   string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	EnableAllCollectors           bool   `name:"enable-all-collectors" help:"Enable all collectors"`
	DisableCollectors             string `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`
	StatsCollections              string `name:"stats-collections" help:"Collections for collstats & indexstats"`
	CollectionsLimit              int32  `name:"max-collections-limit" default:"-1" help:"Disable collstats, dbstats, topmetrics and indexstats if there are more than <n> collections. 0: No limit. Default is -1, which let PMM automatically set this value."`

	AddCommonFlags
}

type AddHAProxyCmd struct {
	ServiceName         string `name:"name" arg:"" default:"${hostname}-haproxy" help:"Service name (autodetected default: ${hostname}-haproxy)"`
	Username            string `name:"username" help:"HAProxy username"`
	Password            string `name:"password" help:"HAProxy password"`
	CredentialsSource   string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	Scheme              string `name:"scheme" placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string `name:"metrics-path" placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16 `name:"listen-port" placeholder:"port" required:"" help:"Listen port of haproxy exposing the metrics for scraping metrics (Required)"`
	NodeID              string `name:"node-id" help:"Node ID (default is autodetected)"`
	Environment         string `name:"environment" placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string `name:"cluster" placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string `name:"replication-set" placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `name:"custom-labels" help:"Custom user-assigned labels. Example: region=east,app=app1"`
	MetricsMode         string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	SkipConnectionCheck bool   `name:"skip-connection-check" help:"Skip connection check"`
}

type AddExternalCmd struct {
	ServiceName         string `name:"service-name" default:"${hostname}${externalDefaultServiceName}" help:"Service name (autodetected default: ${hostname}${externalDefaultServiceName})"`
	RunsOnNodeID        string `name:"agent-node-id" help:"Node ID where agent runs (default is autodetected)"`
	Username            string `name:"username" help:"External username"`
	Password            string `name:"password" help:"External password"`
	CredentialsSource   string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	Scheme              string `name:"scheme" placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string `name:"metrics-path" placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16 `name:"listen-port" placeholder:"port" required:"" help:"Listen port of external exporter for scraping metrics. (Required)"`
	NodeID              string `name:"service-node-id" help:"Node ID where service runs (default is autodetected)"`
	Environment         string `name:"environment" placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string `name:"cluster" placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string `name:"replication-set" placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `name:"custom-labels" help:"Custom user-assigned labels. Example: region=east,app=app1"`
	MetricsMode         string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	Group               string `name:"group" default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	SkipConnectionCheck bool   `name:"skip-connection-check" help:"Skip exporter connection checks"`
}

type AddExternalServerlessCmd struct {
	Name                string `name:"external-name" help:"Service name"`
	URL                 string `name:"url" help:"Full URL to exporter metrics endpoints"`
	Scheme              string `name:"scheme" placeholder:"https" help:"Scheme to generate URI to exporter metrics endpoints"`
	Username            string `name:"username" help:"External username"`
	Password            string `name:"password" help:"External password"`
	CredentialsSource   string `name:"credentials-source" type:"existingfile" help:"Credentials provider"`
	Address             string `name:"address" placeholder:"1.2.3.4:9000" help:"External exporter address and port"`
	Host                string `name:"host" placeholder:"1.2.3.4" help:"External exporters hostname or IP address"`
	ListenPort          uint16 `name:"listen-port" placeholder:"9999" help:"Listen port of external exporter for scraping metrics."`
	MetricsPath         string `name:"metrics-path" placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URL."`
	Environment         string `name:"environment" placeholder:"testing" help:"Environment name"`
	Cluster             string `name:"cluster" help:"Cluster name"`
	ReplicationSet      string `name:"replication-set" placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `name:"custom-labels" placeholder:"'app=myapp,region=s1'" help:"Custom user-assigned labels"`
	Group               string `name:"group" default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	MachineID           string `name:"machine-id" help:"Node machine-id"`
	Distro              string `name:"distro" help:"Node OS distribution"`
	ContainerID         string `name:"container-id" help:"Container ID"`
	ContainerName       string `name:"container-name" help:"Container name"`
	NodeModel           string `name:"node-model" help:"Node model"`
	Region              string `name:"region" help:"Node region"`
	Az                  string `name:"az" help:"Node availability zone"`
	SkipConnectionCheck bool   `name:"skip-connection-check" help:"Skip exporter connection checks"`
}
