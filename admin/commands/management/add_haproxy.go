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
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/ha_proxy"
)

var addHAProxyResultT = commands.ParseTemplate(`
HAProxy Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addHAProxyResult struct {
	Service *ha_proxy.AddHAProxyOKBodyService `json:"service"`
}

func (res *addHAProxyResult) Result() {}

func (res *addHAProxyResult) String() string {
	return commands.RenderTemplate(addHAProxyResultT, res)
}

type addHAProxyCommand struct {
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
	SkipConnectionCheck bool
}

func (cmd *addHAProxyCommand) Run() (commands.Result, error) {
	isSupported, err := helpers.IsHAProxySupported()
	if !isSupported {
		return nil, err
	}

	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &ha_proxy.AddHAProxyParams{
		Body: ha_proxy.AddHAProxyBody{
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
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			CredentialsSource:   cmd.CredentialsSource,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.HAProxy.AddHAProxy(params)
	if err != nil {
		return nil, err
	}

	return &addHAProxyResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command
var (
	AddHAProxy  addHAProxyCommand
	AddHAProxyC = AddC.Command("haproxy", "Add HAProxy to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	defaultServiceName := hostname + "-haproxy"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", defaultServiceName)
	AddHAProxyC.Arg("name", serviceNameHelp).Default(defaultServiceName).StringVar(&AddHAProxy.ServiceName)

	AddHAProxyC.Flag("username", "HAProxy username").StringVar(&AddHAProxy.Username)
	AddHAProxyC.Flag("password", "HAProxy password").StringVar(&AddHAProxy.Password)
	AddHAProxyC.Flag("credentials-source", "Credentials provider").StringVar(&AddHAProxy.CredentialsSource)

	AddHAProxyC.Flag("scheme", "Scheme to generate URI to exporter metrics endpoints").
		PlaceHolder("http or https").StringVar(&AddHAProxy.Scheme)
	AddHAProxyC.Flag("metrics-path", "Path under which metrics are exposed, used to generate URI").
		PlaceHolder("/metrics").StringVar(&AddHAProxy.MetricsPath)
	AddHAProxyC.Flag("listen-port", "Listen port of haproxy exposing the metrics for scraping metrics (Required)").Required().Uint16Var(&AddHAProxy.ListenPort)

	AddHAProxyC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddHAProxy.NodeID)
	AddHAProxyC.Flag("environment", "Environment name like 'production' or 'qa'").
		PlaceHolder("prod").StringVar(&AddHAProxy.Environment)
	AddHAProxyC.Flag("cluster", "Cluster name").
		PlaceHolder("east-cluster").StringVar(&AddHAProxy.Cluster)
	AddHAProxyC.Flag("replication-set", "Replication set name").
		PlaceHolder("rs1").StringVar(&AddHAProxy.ReplicationSet)
	AddHAProxyC.Flag("custom-labels", "Custom user-assigned labels. Example: region=east,app=app1").StringVar(&AddHAProxy.CustomLabels)
	AddHAProxyC.Flag("metrics-mode", "Metrics flow mode, can be push - agent will push metrics,"+
		" pull - server scrape metrics from agent  or auto - chosen by server").
		Default("auto").
		EnumVar(&AddHAProxy.MetricsMode, metricsModes...)
	AddHAProxyC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddHAProxy.SkipConnectionCheck)
}
