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
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

var addClickHouseResultT = commands.ParseTemplate(`
ClickHouse Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addClickHouseResult struct {
	Service            *mservice.AddServiceOKBodyClickhouseService            `json:"service"`
	ClickHouseExporter *mservice.AddServiceOKBodyClickhouseClickhouseExporter `json:"clickhouse_exporter,omitempty"`
	ExternalExporter   *mservice.AddServiceOKBodyClickhouseExternalExporter   `json:"external_exporter,omitempty"`
}

func (res *addClickHouseResult) Result() {}

func (res *addClickHouseResult) String() string {
	return commands.RenderTemplate(addClickHouseResultT, res)
}

// metricsSourceValues maps the --metrics-source CLI value to the API enum.
var metricsSourceValues = map[string]string{
	"auto":     "METRICS_SOURCE_UNSPECIFIED",
	"native":   "METRICS_SOURCE_NATIVE",
	"exporter": "METRICS_SOURCE_EXPORTER",
}

// AddClickHouseCommand is used by Kong for CLI flags and commands.
type AddClickHouseCommand struct {
	ServiceName         string            `name:"name" arg:"" default:"${hostname}-clickhouse" help:"Service name (autodetected default: ${hostname}-clickhouse)"`
	Address             string            `arg:"" optional:"" help:"ClickHouse address and port (default: 127.0.0.1:9000)"`
	Socket              string            `help:"Path to ClickHouse socket"`
	NodeID              string            `help:"Node ID (default is autodetected)"`
	PMMAgentID          string            `help:"The pmm-agent identifier which runs this instance (default is autodetected)"`
	Username            string            `help:"ClickHouse username"`
	Password            string            `help:"ClickHouse password"`
	AgentPassword       string            `help:"Custom password for /metrics endpoint"`
	Environment         string            `help:"Environment name"`
	Cluster             string            `help:"Cluster name"`
	ReplicationSet      string            `help:"Replication set name"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	MetricsSource       string            `enum:"auto,native,exporter" default:"auto" help:"Metrics source: auto (probe the native endpoint), native, or exporter"`
	NativeMetricsPort   uint16            `default:"9363" help:"ClickHouse native Prometheus endpoint port (used for native source and auto-probe)"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificates validation"`
	TLSCaFile           string            `name:"tls-ca" help:"Path to certificate authority certificate file"`
	TLSCertFile         string            `name:"tls-cert" help:"Path to client certificate file"`
	TLSKeyFile          string            `name:"tls-key" help:"Path to client key file"`
	DisableCollectors   []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	ExposeExporter      bool              `name:"expose-exporter" help:"Optionally expose the address of the exporter publicly on 0.0.0.0"`

	AddCommonFlags
	flags.MetricsModeFlags
	flags.LogLevelNoFatalFlags
}

// GetServiceName returns the service name for AddClickHouseCommand.
func (cmd *AddClickHouseCommand) GetServiceName() string {
	return cmd.ServiceName
}

// GetAddress returns the address for AddClickHouseCommand.
func (cmd *AddClickHouseCommand) GetAddress() string {
	return cmd.Address
}

// GetDefaultAddress returns the default address for AddClickHouseCommand.
func (cmd *AddClickHouseCommand) GetDefaultAddress() string {
	return "127.0.0.1:9000"
}

// GetSocket returns the socket for AddClickHouseCommand.
func (cmd *AddClickHouseCommand) GetSocket() string {
	return cmd.Socket
}

// RunCmd runs the command for AddClickHouseCommand.
func (cmd *AddClickHouseCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseKeyValuePair(&cmd.CustomLabels)

	var (
		err                    error
		tlsCa, tlsCert, tlsKey string
	)
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCaFile)
		if err != nil {
			return nil, err
		}

		tlsCert, err = commands.ReadFile(cmd.TLSCertFile)
		if err != nil {
			return nil, err
		}

		tlsKey, err = commands.ReadFile(cmd.TLSKeyFile)
		if err != nil {
			return nil, err
		}
	}

	if cmd.PMMAgentID == "" || cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.PMMAgentID == "" {
			cmd.PMMAgentID = status.AgentID
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	serviceName, socket, host, port, err := processGlobalAddFlagsWithSocket(cmd, cmd.AddCommonFlags)
	if err != nil {
		return nil, err
	}

	metricsSource := metricsSourceValues[cmd.MetricsSource]

	params := &mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Clickhouse: &mservice.AddServiceParamsBodyClickhouse{
				NodeID:         cmd.NodeID,
				ServiceName:    serviceName,
				Address:        host,
				Socket:         socket,
				Port:           int64(port),
				ExposeExporter: cmd.ExposeExporter,
				PMMAgentID:     cmd.PMMAgentID,
				Environment:    cmd.Environment,
				Cluster:        cmd.Cluster,
				ReplicationSet: cmd.ReplicationSet,
				Username:       cmd.Username,
				Password:       cmd.Password,
				AgentPassword:  cmd.AgentPassword,
				CustomLabels:   pointer.Get(customLabels),

				SkipConnectionCheck: cmd.SkipConnectionCheck,

				TLS:               cmd.TLS,
				TLSSkipVerify:     cmd.TLSSkipVerify,
				TLSCa:             tlsCa,
				TLSCert:           tlsCert,
				TLSKey:            tlsKey,
				MetricsMode:       cmd.MetricsMode.EnumValue(),
				LogLevel:          cmd.LogLevel.EnumValue(),
				MetricsSource:     pointer.ToString(metricsSource),
				NativeMetricsPort: int64(cmd.NativeMetricsPort),
			},
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.ManagementService.AddService(params)
	if err != nil {
		return nil, err
	}

	return &addClickHouseResult{
		Service:            resp.Payload.Clickhouse.Service,
		ClickHouseExporter: resp.Payload.Clickhouse.ClickhouseExporter,
		ExternalExporter:   resp.Payload.Clickhouse.ExternalExporter,
	}, nil
}
