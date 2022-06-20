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

package commands

import "time"

type StatusCmd struct {
	Timeout time.Duration `name:"wait" help:"Time to wait for a successful response from pmm-agent"`
}

type SummaryCmd struct {
	Filename   string `help:"Summary archive filename"`
	SkipServer bool   `help:"Skip fetching logs.zip from PMM Server"`
	Pprof      bool   `name:"pprof" help:"Include performance profiling data"`
}

type ListCmd struct {
	NodeID string `help:"Node ID (default is autodetected)"`
}

type ConfigCmd struct {
	NodeAddress       string `arg:"" default:"${nodeIp}" help:"Node address (autodetected default: ${nodeIp})"`
	NodeType          string `arg:"" enum:"generic,container" default:"${nodeTypeDefault}" help:"Node type, one of: generic, container (default: ${nodeTypeDefault})"`
	NodeName          string `arg:"" default:"${hostname}" help:"Node name (autodetected default: ${hostname})"`
	NodeModel         string `help:"Node model"`
	Region            string `help:"Node region"`
	Az                string `help:"Node availability zone"`
	AgentPassword     string `help:"Custom password for /metrics endpoint"`
	Force             bool   `help:"Remove Node with that name with all dependent Services and Agents if one exist"`
	MetricsMode       string `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode for agents node-exporter, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server."`
	DisableCollectors string `help:"Comma-separated list of collector names to exclude from exporter"`
	CustomLabels      string `help:"Custom user-assigned labels"`
	BasePath          string `name:"paths-base" help:"Base path where all binaries, tools and collectors of PMM client are located"`
	LogLevel          string `enum:"debug,info,warn,error,fatal" default:"warn" help:"Logging level"`
}

type AnnotateCmd struct {
	Text        string `arg:"" help:"Text of annotation"`
	Tags        string `help:"Tags to filter annotations. Multiple tags are separated by a comma"`
	Node        bool   `help:"Annotate current node"`
	NodeName    string `help:"Name of node to annotate"`
	Service     bool   `help:"Annotate services of current node"`
	ServiceName string `help:"Name of service to annotate"`
}

type VersionCmd struct{}
