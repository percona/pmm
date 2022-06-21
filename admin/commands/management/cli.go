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

import "github.com/alecthomas/units"

type UnregisterCommand struct {
	Force    bool   `help:"Remove this node with all dependencies"`
	NodeName string `help:"Node name (autodetected default: ${hostname})"`
}

type RemoveCommand struct {
	ServiceType string `arg:"" enum:"${serviceTypesEnum}" help:"Service type, one of: ${serviceTypesEnum}"`
	ServiceName string `arg:"" default:"" help:"Service name"`
	ServiceID   string `help:"Service ID"`
}

type RegisterCommand struct {
	Address           string `name:"node-address" arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string `arg:"" enum:"generic,container" default:"generic" help:"Node type, one of: generic, container (default: generic)"`
	NodeName          string `arg:"" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	MachineID         string `default:"${defaultMachineID}" help:"Node machine-id (default is autodetected)"`
	Distro            string `default:"${distro}" help:"Node OS distribution (default is autodetected)"`
	ContainerID       string `help:"Container ID"`
	ContainerName     string `help:"Container name"`
	NodeModel         string `help:"Node model"`
	Region            string `help:"Node region"`
	Az                string `help:"Node availability zone"`
	CustomLabels      string `help:"Custom user-assigned labels"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	Force             bool   `help:"Re-register Node"`
	MetricsMode       string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors string `help:"Comma-separated list of collector names to exclude from exporter"`
}

type AddCommand struct {
	External           AddExternalCmd           `cmd:"" help:"Add External source of data (like a custom exporter running on a port) to the monitoring"`
	ExternalServerless AddExternalServerlessCmd `cmd:"" help:"Add External Service on Remote node to monitoring."`
	HAProxy            AddHAProxyCmd            `cmd:"" name:"haproxy" help:"Add HAProxy to monitoring"`
	MongoDB            AddMongoDBCmd            `cmd:"" name:"mongodb" help:"Add MongoDB to monitoring"`
	MySQL              AddMySQLCmd              `cmd:"" name:"mysql" help:"Add MySQL to monitoring"`
	PostgreSQL         AddPostgreSQLCmd         `cmd:"" name:"postgresql" help:"Add PostgreSQL to monitoring"`
	ProxySQL           AddProxySQLCmd           `cmd:"" name:"proxysql" help:"Add ProxySQL to monitoring"`
}

type AddCommonFlags struct {
	AddServiceNameFlag string `name:"service-name" placeholder:"NAME" help:"Service name (overrides positional argument)"`
	AddHostFlag        string `name:"host" placeholder:"HOST" help:"Service hostname or IP address (overrides positional argument)"`
	AddPortFlag        uint16 `name:"port" placeholder:"PORT" help:"Service port number (overrides positional argument)"`
	AddLogLevel        string `name:"log-level" enum:"debug,info,warn,error,fatal" default:"warn" help:"Service logging level"`
}

type AddProxySQLCmd struct {
	ServiceName         string `name:"name" arg:"" default:"${hostname}-proxysql" help:"Service name (autodetected default: ${hostname}-proxysql)"`
	Address             string `arg:"" default:"127.0.0.1:6032" help:"ProxySQL address and port (default: 127.0.0.1:6032)"`
	Socket              string `help:"Path to ProxySQL socket"`
	NodeID              string `help:"Node ID (default is autodetected)"`
	PMMAgentID          string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username            string `default:"admin" help:"ProxySQL username"`
	Password            string `default:"admin" help:"ProxySQL password"`
	AgentPassword       string `help:"Custom password for /metrics endpoint"`
	CredentialsSource   string `type:"existingfile" help:"Credentials provider"`
	Environment         string `help:"Environment name"`
	Cluster             string `help:"Cluster name"`
	ReplicationSet      string `help:"Replication set name"`
	CustomLabels        string `help:"Custom user-assigned labels"`
	SkipConnectionCheck bool   `help:"Skip connection check"`
	TLS                 bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool   `help:"Skip TLS certificates validation"`
	MetricsMode         string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors   string `help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddPostgreSQLCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-postgresql" help:"Service name (autodetected default: ${hostname}-postgresql)"`
	Address           string `arg:"" default:"127.0.0.1:5432" help:"PostgreSQL address and port (default: 127.0.0.1:5432)"`
	Socket            string `help:"Path to socket"`
	Username          string `default:"postgres" help:"PostgreSQL username"`
	Password          string `help:"PostgreSQL password"`
	Database          string `help:"PostgreSQL database"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	CredentialsSource string `type:"existingfile" help:"Credentials provider"`
	NodeID            string `help:"Node ID (default is autodetected)"`
	PMMAgentID        string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	// TODO add "auto"
	QuerySource          string `default:"pgstatements" help:"Source of SQL queries, one of: pgstatements, pgstatmonitor, none (default: pgstatements)"`
	Environment          string `help:"Environment name"`
	Cluster              string `help:"Cluster name"`
	ReplicationSet       string `help:"Replication set name"`
	CustomLabels         string `help:"Custom user-assigned labels"`
	SkipConnectionCheck  bool   `help:"Skip connection check"`
	TLS                  bool   `help:"Use TLS to connect to the database"`
	TLSCAFile            string `name:"tls-ca-file" help:"TLS CA certificate file"`
	TLSCertFile          string `help:"TLS certificate file"`
	TLSKeyFile           string `help:"TLS certificate key file"`
	TLSSkipVerify        bool   `help:"Skip TLS certificates validation"`
	DisableQueryExamples bool   `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MetricsMode          string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors    string `help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddMySQLCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-mysql" help:"Service name (autodetected default: ${hostname}-mysql)"`
	Address           string `arg:"" default:"127.0.0.1:3306" help:"MySQL address and port (default: 127.0.0.1:3306)"`
	Socket            string `help:"Path to MySQL socket"`
	NodeID            string `help:"Node ID (default is autodetected)"`
	PMMAgentID        string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username          string `help:"MySQL username"`
	Password          string `help:"MySQL password"`
	DefaultsFile      string `help:"Path to defaults file"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	CredentialsSource string `type:"existingfile" help:"Credentials provider"`
	// TODO add "auto", make it default
	QuerySource            string           `default:"${mysqlQuerySourceDefault}" enum:"${mysqlQuerySourcesEnum}" help:"Source of SQL queries, one of: ${mysqlQuerySourcesEnum} (default: ${mysqlQuerySourceDefault})"`
	DisableQueryExamples   bool             `name:"disable-queryexamples" help:"Disable collection of query examples"`
	MaxSlowlogFileSize     units.Base2Bytes `name:"size-slow-logs" placeholder:"size" help:"Rotate slow log file at this size (default: server-defined; negative value disables rotation). Ex.: 1GiB"`
	DisableTablestats      bool             `help:"Disable table statistics collection"`
	DisableTablestatsLimit uint16           `help:"Table statistics collection will be disabled if there are more than specified number of tables (default: server-defined)"`
	Environment            string           `help:"Environment name"`
	Cluster                string           `help:"Cluster name"`
	ReplicationSet         string           `help:"Replication set name"`
	CustomLabels           string           `help:"Custom user-assigned labels"`
	SkipConnectionCheck    bool             `help:"Skip connection check"`
	TLS                    bool             `help:"Use TLS to connect to the database"`
	TLSSkipVerify          bool             `help:"Skip TLS certificates validation"`
	TLSCaFile              string           `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile            string           `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile             string           `name:"tls-key" help:"Path to client key file"`
	CreateUser             bool             `hidden:"" help:"Create pmm user"`
	MetricsMode            string           `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors      string           `help:"Comma-separated list of collector names to exclude from exporter"`

	AddCommonFlags
}

type AddMongoDBCmd struct {
	ServiceName       string `name:"name" arg:"" default:"${hostname}-mongodb" help:"Service name (autodetected default: ${hostname}-mongodb)"`
	Address           string `arg:"" default:"127.0.0.1:27017" help:"MongoDB address and port (default: 127.0.0.1:27017)"`
	Socket            string `help:"Path to socket"`
	NodeID            string `help:"Node ID (default is autodetected)"`
	PMMAgentID        string `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username          string `help:"MongoDB username"`
	Password          string `help:"MongoDB password"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	CredentialsSource string `type:"existingfile" help:"Credentials provider"`
	// TODO add "auto"
	QuerySource                   string `default:"${mongoDbQuerySourceDefault}" enum:"${mongoDbQuerySourcesEnum}" help:"Source of queries, one of: ${mongoDbQuerySourcesEnum} (default: ${mongoDbQuerySourceDefault})"`
	Environment                   string `help:"Environment name"`
	Cluster                       string `help:"Cluster name"`
	ReplicationSet                string `help:"Replication set name"`
	CustomLabels                  string `help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool   `help:"Skip connection check"`
	TLS                           bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool   `help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string `help:"Password for certificate"`
	TLSCaFile                     string `help:"Path to certificate authority file"`
	AuthenticationMechanism       string `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	AuthenticationDatabase        string `help:"Authentication database. Default is empty. Use $external for ssl certificates"`
	MetricsMode                   string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	EnableAllCollectors           bool   `help:"Enable all collectors"`
	DisableCollectors             string `help:"Comma-separated list of collector names to exclude from exporter"`
	StatsCollections              string `help:"Collections for collstats & indexstats"`
	CollectionsLimit              int32  `name:"max-collections-limit" default:"-1" help:"Disable collstats, dbstats, topmetrics and indexstats if there are more than <n> collections. 0: No limit. Default is -1, which let PMM automatically set this value."`

	AddCommonFlags
}

type AddHAProxyCmd struct {
	ServiceName         string `name:"name" arg:"" default:"${hostname}-haproxy" help:"Service name (autodetected default: ${hostname}-haproxy)"`
	Username            string `help:"HAProxy username"`
	Password            string `help:"HAProxy password"`
	CredentialsSource   string `type:"existingfile" help:"Credentials provider"`
	Scheme              string `placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16 `placeholder:"port" required:"" help:"Listen port of haproxy exposing the metrics for scraping metrics (Required)"`
	NodeID              string `help:"Node ID (default is autodetected)"`
	Environment         string `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `help:"Custom user-assigned labels. Example: region=east,app=app1"`
	MetricsMode         string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	SkipConnectionCheck bool   `help:"Skip connection check"`
}

type AddExternalCmd struct {
	ServiceName         string `default:"${hostname}${externalDefaultServiceName}" help:"Service name (autodetected default: ${hostname}${externalDefaultServiceName})"`
	RunsOnNodeID        string `name:"agent-node-id" help:"Node ID where agent runs (default is autodetected)"`
	Username            string `help:"External username"`
	Password            string `help:"External password"`
	CredentialsSource   string `type:"existingfile" help:"Credentials provider"`
	Scheme              string `placeholder:"http or https" help:"Scheme to generate URI to exporter metrics endpoints"`
	MetricsPath         string `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URI"`
	ListenPort          uint16 `placeholder:"port" required:"" help:"Listen port of external exporter for scraping metrics. (Required)"`
	NodeID              string `name:"service-node-id" help:"Node ID where service runs (default is autodetected)"`
	Environment         string `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster             string `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet      string `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `help:"Custom user-assigned labels. Example: region=east,app=app1"`
	MetricsMode         string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	Group               string `default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	SkipConnectionCheck bool   `help:"Skip exporter connection checks"`
}

type AddExternalServerlessCmd struct {
	Name                string `name:"external-name" help:"Service name"`
	URL                 string `help:"Full URL to exporter metrics endpoints"`
	Scheme              string `placeholder:"https" help:"Scheme to generate URI to exporter metrics endpoints"`
	Username            string `help:"External username"`
	Password            string `help:"External password"`
	CredentialsSource   string `type:"existingfile" help:"Credentials provider"`
	Address             string `placeholder:"1.2.3.4:9000" help:"External exporter address and port"`
	Host                string `placeholder:"1.2.3.4" help:"External exporters hostname or IP address"`
	ListenPort          uint16 `placeholder:"9999" help:"Listen port of external exporter for scraping metrics."`
	MetricsPath         string `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URL."`
	Environment         string `placeholder:"testing" help:"Environment name"`
	Cluster             string `help:"Cluster name"`
	ReplicationSet      string `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        string `placeholder:"'app=myapp,region=s1'" help:"Custom user-assigned labels"`
	Group               string `default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	MachineID           string `help:"Node machine-id"`
	Distro              string `help:"Node OS distribution"`
	ContainerID         string `help:"Container ID"`
	ContainerName       string `help:"Container name"`
	NodeModel           string `help:"Node model"`
	Region              string `help:"Node region"`
	Az                  string `help:"Node availability zone"`
	SkipConnectionCheck bool   `help:"Skip exporter connection checks"`
}
