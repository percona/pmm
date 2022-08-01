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
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/external"
)

var addExternalServerlessResultT = commands.ParseTemplate(`
External Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
Group       : {{ .Service.Group }}
`)

type addExternalServerlessResult struct {
	Service *external.AddExternalOKBodyService `json:"service"`
}

func (res *addExternalServerlessResult) Result() {}

func (res *addExternalServerlessResult) String() string {
	return commands.RenderTemplate(addExternalServerlessResultT, res)
}

type addExternalServerlessCommand struct {
	Name              string
	Username          string
	Password          string
	CredentialsSource string

	URL         string
	Scheme      string
	Address     string
	Host        string
	ListenPort  uint16
	MetricsPath string

	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
	Group          string

	MachineID           string
	Distro              string
	ContainerID         string
	ContainerName       string
	NodeModel           string
	Region              string
	Az                  string
	SkipConnectionCheck bool
}

func (cmd *addExternalServerlessCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	scheme, metricsPath, address, port, err := cmd.processURLFlags()
	if err != nil {
		return nil, err
	}

	serviceName := cmd.Name
	if serviceName == "" {
		serviceName = fmt.Sprintf("%s-external", address)
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &external.AddExternalParams{
		Body: external.AddExternalBody{
			AddNode: &external.AddExternalParamsBodyAddNode{
				NodeType:      pointer.ToString(external.AddExternalParamsBodyAddNodeNodeTypeREMOTENODE),
				NodeName:      serviceName,
				MachineID:     cmd.MachineID,
				Distro:        cmd.Distro,
				ContainerID:   cmd.ContainerID,
				ContainerName: cmd.ContainerName,
				NodeModel:     cmd.NodeModel,
				Region:        cmd.Region,
				Az:            cmd.Az,
				CustomLabels:  customLabels,
			},
			Address:             address,
			ServiceName:         serviceName,
			Username:            cmd.Username,
			Password:            cmd.Password,
			Scheme:              scheme,
			MetricsPath:         metricsPath,
			ListenPort:          int64(port),
			Environment:         cmd.Environment,
			Cluster:             cmd.Cluster,
			ReplicationSet:      cmd.ReplicationSet,
			CustomLabels:        customLabels,
			MetricsMode:         pointer.ToString(external.AddExternalBodyMetricsModePULL),
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

	return &addExternalServerlessResult{
		Service: resp.Payload.Service,
	}, nil
}

func (cmd *addExternalServerlessCommand) processURLFlags() (scheme, metricsPath, address string, port uint16, err error) {
	scheme = cmd.Scheme
	address = cmd.Host
	port = cmd.ListenPort
	metricsPath = cmd.MetricsPath

	switch {
	case cmd.URL != "":
		uri, err := url.Parse(cmd.URL)
		if err != nil {
			return "", "", "", 0, errors.Wrapf(err, "couldn't parse URL: %s", cmd.URL)
		}
		scheme = uri.Scheme
		address = uri.Hostname()
		portS := uri.Port()
		if portS != "" {
			portI, err := strconv.Atoi(portS)
			if err != nil {
				return "", "", "", 0, err
			}
			port = uint16(portI)
		}
		metricsPath = uri.Path
	case cmd.Address != "":
		host, portS, err := net.SplitHostPort(cmd.Address)
		if err != nil {
			return "", "", "", 0, err
		}
		address = host
		portI, err := strconv.Atoi(portS)
		if err != nil {
			return "", "", "", 0, err
		}
		port = uint16(portI)
	}

	return scheme, metricsPath, address, port, nil
}

const (
	serverlessHelp = `Add External Service on Remote node to monitoring.

Usage example:
sudo pmm-admin add external-serverless --url=http://1.2.3.4:9093/metrics

Also, individual parameters can be set instead of --url like:
sudo pmm-admin add external-serverless --scheme=http --host=1.2.3.4 --listen-port=9093 --metrics-path=/metrics --container-name=ddd --external-name=e125

Notice that some parameters are mandatory depending on the context. 
For example, if you specify --url, --schema and other related parameters are not mandatory but,
if you specify --host you must provide all other parameters needed to build the destination URL 
or even you can specify --address instead of host and port as individual parameters.
`
)

// register command.
var (
	AddExternalServerless  addExternalServerlessCommand
	AddExternalServerlessC = AddC.Command("external-serverless", serverlessHelp)
)

func init() {
	AddExternalServerlessC.Flag("external-name", "Service name").StringVar(&AddExternalServerless.Name)

	AddExternalServerlessC.Flag("url", "Full URL to exporter metrics endpoints").StringVar(&AddExternalServerless.URL)
	AddExternalServerlessC.Flag("scheme", "Scheme to generate URL to exporter metrics endpoints").
		PlaceHolder("https").StringVar(&AddExternalServerless.Scheme)

	AddExternalServerlessC.Flag("username", "External username").StringVar(&AddExternalServerless.Username)
	AddExternalServerlessC.Flag("password", "External password").StringVar(&AddExternalServerless.Password)
	AddExternalServerlessC.Flag("credentials-source", "Credentials provider").StringVar(&AddExternalServerless.CredentialsSource)

	AddExternalServerlessC.Flag("address", "External exporter address and port").
		PlaceHolder("1.2.3.4:9000").StringVar(&AddExternalServerless.Address)

	AddExternalServerlessC.Flag("host", "External exporters hostname or IP address").
		PlaceHolder("1.2.3.4").StringVar(&AddExternalServerless.Host)

	AddExternalServerlessC.Flag("listen-port", "Listen port of external exporter for scraping metrics.").
		PlaceHolder("9999").Uint16Var(&AddExternalServerless.ListenPort)

	AddExternalServerlessC.Flag("metrics-path", "Path under which metrics are exposed, used to generate URL.").
		PlaceHolder("/metrics").StringVar(&AddExternalServerless.MetricsPath)

	AddExternalServerlessC.Flag("environment", "Environment name").
		PlaceHolder("testing").StringVar(&AddExternalServerless.Environment)

	AddExternalServerlessC.Flag("cluster", "Cluster name").StringVar(&AddExternalServerless.Cluster)
	AddExternalServerlessC.Flag("replication-set", "Replication set name").
		PlaceHolder("rs1").StringVar(&AddExternalServerless.ReplicationSet)

	AddExternalServerlessC.Flag("custom-labels", "Custom user-assigned labels").
		PlaceHolder("'app=myapp,region=s1'").StringVar(&AddExternalServerless.CustomLabels)

	groupHelp := fmt.Sprintf("Group name of external service (default: %s)", defaultGroupExternalExporter)
	AddExternalServerlessC.Flag("group", groupHelp).Default(defaultGroupExternalExporter).StringVar(&AddExternalServerless.Group)

	AddExternalServerlessC.Flag("machine-id", "Node machine-id").StringVar(&AddExternalServerless.MachineID)
	AddExternalServerlessC.Flag("distro", "Node OS distribution").StringVar(&AddExternalServerless.Distro)
	AddExternalServerlessC.Flag("container-id", "Container ID").StringVar(&AddExternalServerless.ContainerID)
	AddExternalServerlessC.Flag("container-name", "Container name").StringVar(&AddExternalServerless.ContainerName)
	AddExternalServerlessC.Flag("node-model", "Node model").StringVar(&AddExternalServerless.NodeModel)
	AddExternalServerlessC.Flag("region", "Node region").StringVar(&AddExternalServerless.Region)
	AddExternalServerlessC.Flag("az", "Node availability zone").StringVar(&AddExternalServerless.Az)
	AddExternalServerlessC.Flag("skip-connection-check", "Skip exporter connection checks").BoolVar(&AddExternalServerless.SkipConnectionCheck)
}
