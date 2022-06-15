package commands

import "time"

type StatusCmd struct {
	Timeout time.Duration `name:"wait" help:"Time to wait for a successful response from pmm-agent"`
}

type SummaryCmd struct {
	Filename   string `name:"filename" help:"Summary archive filename"`
	SkipServer bool   `name:"skip-server" help:"Skip fetching logs.zip from PMM Server"`
	Pprof      bool   `name:"pprof" help:"Include performance profiling data"`
}

type ListCmd struct {
	NodeID string `name:"node-id" help:"Node ID (default is autodetected)"`
}

type ConfigCmd struct {
	NodeAddress       string `name:"node-address" arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string `name:"node-type" enum:"generic,container" default:"${nodeTypeDefault}" help:"Node type, one of: generic, container (default: ${nodeTypeDefault})"`
	NodeName          string `name:"node-name" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	NodeModel         string `name:"node-model" help:"Node model"`
	Region            string `name:"region" help:"Node region"`
	Az                string `name:"az" help:"Node availability zone"`
	AgentPassword     string `name:"agent-password" help:"Custom password for /metrics endpoint"`
	Force             bool   `name:"force" help:"Remove Node with that name with all dependent Services and Agents if one exist"`
	MetricsMode       string `name:"metrics-mode" enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode for agents node-exporter, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors string `name:"disable-collectors" help:"Comma-separated list of collector names to exclude from exporter"`
	CustomLabels      string `name:"custom-labels" help:"Custom user-assigned labels"`
	BasePath          string `name:"paths-base" help:"Base path where all binaries, tools and collectors of PMM client are located"`
	LogLevel          string `name:"log-level" enum:"debug,info,warn,error,fatal" default:"warn" help:"Logging level"`
}

type AnnotateCmd struct {
	Text        string `name:"text" arg:"" help:"Text of annotation"`
	Tags        string `name:"tags" help:"Tags to filter annotations. Multiple tags are separated by a comma"`
	Node        bool   `name:"node" help:"Annotate current node"`
	NodeName    string `name:"node-name" help:"Name of node to annotate"`
	Service     bool   `name:"service" help:"Annotate services of current node"`
	ServiceName string `name:"service-name" help:"Name of service to annotate"`
}
