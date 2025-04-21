// Copyright (C) 2023 Percona LLC
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
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

var addExternalServerlessResultT = commands.ParseTemplate(`
External Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
Group       : {{ .Service.Group }}
`)

type addExternalServerlessResult struct {
	Service *mservice.AddServiceOKBodyExternalService `json:"service"`
}

func (res *addExternalServerlessResult) Result() {}

func (res *addExternalServerlessResult) String() string {
	return commands.RenderTemplate(addExternalServerlessResultT, res)
}

// AddExternalServerlessCommand is used by Kong for CLI flags and commands.
type AddExternalServerlessCommand struct {
	Name                string            `name:"external-name" help:"Service name"`
	URL                 string            `help:"Full URL to exporter metrics endpoints"`
	Scheme              string            `placeholder:"https" help:"Scheme to generate URI to exporter metrics endpoints"`
	Username            string            `help:"External username"`
	Password            string            `help:"External password"`
	CredentialsSource   string            `type:"existingfile" help:"Credentials provider"`
	Address             string            `placeholder:"1.2.3.4:9000" help:"External exporter address and port"`
	Host                string            `placeholder:"1.2.3.4" help:"External exporters hostname or IP address"`
	ListenPort          uint16            `placeholder:"9999" help:"Listen port of external exporter for scraping metrics"`
	MetricsPath         string            `placeholder:"/metrics" help:"Path under which metrics are exposed, used to generate URL"`
	Environment         string            `placeholder:"testing" help:"Environment name"`
	Cluster             string            `help:"Cluster name"`
	ReplicationSet      string            `placeholder:"rs1" help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	Group               string            `default:"${externalDefaultGroupExporter}" help:"Group name of external service (default: ${externalDefaultGroupExporter})"`
	MachineID           string            `help:"Node machine-id"`
	Distro              string            `help:"Node OS distribution"`
	ContainerID         string            `help:"Container ID"`
	ContainerName       string            `help:"Container name"`
	NodeModel           string            `help:"Node model"`
	Region              string            `help:"Node region"`
	Az                  string            `help:"Node availability zone"`
	SkipConnectionCheck bool              `help:"Skip exporter connection checks"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
}

// Help returns cli usage help.
func (cmd *AddExternalServerlessCommand) Help() string {
	return `Usage example:
sudo pmm-admin add external-serverless --url=http://1.2.3.4:9093/metrics

Also, individual parameters can be set instead of --url like:
sudo pmm-admin add external-serverless --scheme=http --host=1.2.3.4 --listen-port=9093 --metrics-path=/metrics --container-name=ddd --external-name=e125

Notice that some parameters are mandatory depending on the context. 
For example, if you specify --url, --schema and other related parameters are not mandatory but,
if you specify --host you must provide all other parameters needed to build the destination URL 
or even you can specify --address instead of host and port as individual parameters.
`
}

// GetCredentials returns the credentials for AddExternalServerlessCommand.
func (cmd *AddExternalServerlessCommand) GetCredentials() error {
	creds, err := commands.ReadFromSource(cmd.CredentialsSource)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	cmd.Password = creds.Password
	cmd.Username = creds.Username

	return nil
}

// RunCmd runs the command for AddExternalServerlessCommand.
func (cmd *AddExternalServerlessCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

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

	if cmd.CredentialsSource != "" {
		if err := cmd.GetCredentials(); err != nil {
			return nil, fmt.Errorf("failed to retrieve credentials from %s: %w", cmd.CredentialsSource, err)
		}
	}

	params := &mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			External: &mservice.AddServiceParamsBodyExternal{
				AddNode: &mservice.AddServiceParamsBodyExternalAddNode{
					NodeType:      pointer.ToString(mservice.AddServiceParamsBodyExternalAddNodeNodeTypeNODETYPEREMOTENODE),
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
				MetricsMode:         pointer.ToString(mservice.AddServiceParamsBodyExternalMetricsModeMETRICSMODEPULL),
				Group:               cmd.Group,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
				TLSSkipVerify:       cmd.TLSSkipVerify,
			},
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ManagementService.AddService(params)
	if err != nil {
		return nil, err
	}

	return &addExternalServerlessResult{
		Service: resp.Payload.External.Service,
	}, nil
}

func (cmd *AddExternalServerlessCommand) processURLFlags() (string, string, string, uint16, error) {
	scheme := cmd.Scheme
	address := cmd.Host
	port := cmd.ListenPort
	metricsPath := cmd.MetricsPath

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
