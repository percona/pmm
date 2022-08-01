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

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/external"
)

const (
	defaultGroupExternalExporter = "external"
	defaultServiceNameSuffix     = "-external"
)

var addExternalResultT = commands.ParseTemplate(`
External Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
Group       : {{ .Service.Group }}
`)

type addExternalResult struct {
	Service *external.AddExternalOKBodyService `json:"service"`
}

func (res *addExternalResult) Result() {}

func (res *addExternalResult) String() string {
	return commands.RenderTemplate(addExternalResultT, res)
}

type addExternalCommand struct {
	RunsOnNodeID        string
	ServiceName         string
	Username            string
	Password            string
	CredentialsSource   string
	Scheme              string
	MetricsPath         string
	ListenPort          uint16
	NodeID              string
	Environment         string
	Cluster             string
	ReplicationSet      string
	CustomLabels        string
	MetricsMode         string
	Group               string
	SkipConnectionCheck bool
}

func (cmd *addExternalCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.RunsOnNodeID == "" || cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.RunsOnNodeID == "" {
			cmd.RunsOnNodeID = status.NodeID
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	hostname, _ := os.Hostname()
	defaultServiceName := hostname + defaultServiceNameSuffix

	if cmd.Group != defaultGroupExternalExporter && cmd.ServiceName == defaultServiceName {
		cmd.ServiceName = fmt.Sprintf("%s-%s", strings.TrimSuffix(cmd.ServiceName, defaultServiceNameSuffix), cmd.Group)
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &external.AddExternalParams{
		Body: external.AddExternalBody{
			RunsOnNodeID:        cmd.RunsOnNodeID,
			ServiceName:         cmd.ServiceName,
			Username:            cmd.Username,
			Password:            cmd.Password,
			Scheme:              cmd.Scheme,
			MetricsPath:         cmd.MetricsPath,
			ListenPort:          int64(cmd.ListenPort),
			NodeID:              cmd.NodeID,
			Environment:         cmd.Environment,
			Cluster:             cmd.Cluster,
			ReplicationSet:      cmd.ReplicationSet,
			CustomLabels:        customLabels,
			MetricsMode:         pointer.ToString(strings.ToUpper(cmd.MetricsMode)),
			Group:               cmd.Group,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			CredentialsSource:   cmd.CredentialsSource,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.External.AddExternal(params)
	if err != nil {
		return nil, err
	}

	return &addExternalResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command.
var (
	AddExternal  addExternalCommand
	AddExternalC = AddC.Command("external", "Add External source of data (like a custom exporter running on a port) to the monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	defaultServiceName := hostname + defaultServiceNameSuffix
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", defaultServiceName)
	AddExternalC.Flag("service-name", serviceNameHelp).Default(defaultServiceName).StringVar(&AddExternal.ServiceName)

	AddExternalC.Flag("agent-node-id", "Node ID where agent runs (default is autodetected)").StringVar(&AddExternal.RunsOnNodeID)

	AddExternalC.Flag("username", "External username").StringVar(&AddExternal.Username)
	AddExternalC.Flag("password", "External password").StringVar(&AddExternal.Password)
	AddExternalC.Flag("credentials-source", "Credentials provider").StringVar(&AddExternal.CredentialsSource)

	AddExternalC.Flag("scheme", "Scheme to generate URI to exporter metrics endpoints").
		PlaceHolder("http or https").StringVar(&AddExternal.Scheme)
	AddExternalC.Flag("metrics-path", "Path under which metrics are exposed, used to generate URI.").
		PlaceHolder("/metrics").StringVar(&AddExternal.MetricsPath)
	AddExternalC.Flag("listen-port", "Listen port of external exporter for scraping metrics. (Required)").Required().Uint16Var(&AddExternal.ListenPort)

	AddExternalC.Flag("service-node-id", "Node ID where service runs (default is autodetected)").StringVar(&AddExternal.NodeID)
	AddExternalC.Flag("environment", "Environment name like 'production' or 'qa'").
		PlaceHolder("prod").StringVar(&AddExternal.Environment)
	AddExternalC.Flag("cluster", "Cluster name").PlaceHolder("east-cluster").StringVar(&AddExternal.Cluster)
	AddExternalC.Flag("replication-set", "Replication set name").
		PlaceHolder("rs1").StringVar(&AddExternal.ReplicationSet)
	AddExternalC.Flag("custom-labels", "Custom user-assigned labels. Example: region=east,app=app1").StringVar(&AddExternal.CustomLabels)
	AddExternalC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server.").
		Default("auto").
		EnumVar(&AddExternal.MetricsMode, metricsModes...)

	groupHelp := fmt.Sprintf("Group name of external service (default: %s)", defaultGroupExternalExporter)
	AddExternalC.Flag("group", groupHelp).Default(defaultGroupExternalExporter).StringVar(&AddExternal.Group)
	AddExternalC.Flag("skip-connection-check", "Skip exporter connection checks").BoolVar(&AddExternal.SkipConnectionCheck)
}
