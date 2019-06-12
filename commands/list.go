// pmm-admin
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package commands

import (
	"net"
	"strconv"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/agentlocal"
)

var listResultT = ParseTemplate(`
Service type  Service name         Address and port  Service ID
{{ range .Services }}
{{- printf "%-13s" .ServiceType }} {{ printf "%-20s" .ServiceName }} {{ printf "%-17s" .AddressPort }} {{ .ServiceID }}
{{ end }}
Agent type                  Status     Agent ID                                        Service ID
{{ range .Agents }}
{{- printf "%-27s" .AgentType }} {{ printf "%-10s" .Status }} {{ .AgentID }}  {{ .ServiceID }}
{{ end }}
`)

type listResultAgent struct {
	AgentType string `json:"agent_type"`
	AgentID   string `json:"agent_id"`
	ServiceID string `json:"service_id"`
	Status    string `json:"status"`
}

type listResultService struct {
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	AddressPort string `json:"address_port"`
}

type listResult struct {
	Services []listResultService `json:"service"`
	Agents   []listResultAgent   `json:"agent"`
}

func (res *listResult) Result() {}

func (res *listResult) String() string {
	return RenderTemplate(listResultT, res)
}

type listCommand struct {
	NodeID string
}

func (cmd *listCommand) Run() (Result, error) {
	// Unlike status, this command uses PMM Server APIs.
	// It does not use local pmm-agent status API beyond getting a Node ID.

	if cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		cmd.NodeID = status.NodeID
	}

	servicesRes, err := client.Default.Services.ListServices(&services.ListServicesParams{
		Body: services.ListServicesBody{
			NodeID: cmd.NodeID,
		},
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	var services []listResultService
	for _, s := range servicesRes.Payload.Mysql {
		services = append(services, listResultService{
			ServiceType: "MySQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range servicesRes.Payload.Mongodb {
		services = append(services, listResultService{
			ServiceType: "MongoDB",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range servicesRes.Payload.Postgresql {
		services = append(services, listResultService{
			ServiceType: "PostgreSQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range servicesRes.Payload.Proxysql {
		services = append(services, listResultService{
			ServiceType: "ProxySQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}

	agentsRes, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	getStatus := func(s *string, disabled bool) string {
		res := strings.ToLower(pointer.GetString(s))
		if res == "" {
			res = "unknown"
		}
		if disabled {
			res += " (disabled)"
		}
		return res
	}

	pmmAgentIDs := map[string]struct{}{}
	var agents []listResultAgent
	for _, a := range agentsRes.Payload.PMMAgent {
		if a.RunsOnNodeID == cmd.NodeID {
			pmmAgentIDs[a.AgentID] = struct{}{}

			status := "disconnected"
			if a.Connected {
				status = "connected"
			}
			agents = append(agents, listResultAgent{
				AgentType: "pmm-agent",
				AgentID:   a.AgentID,
				Status:    status,
			})
		}
	}
	for _, a := range agentsRes.Payload.NodeExporter {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "node_exporter",
				AgentID:   a.AgentID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.MysqldExporter {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "mysqld_exporter",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.QANMysqlPerfschemaAgent {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "qan-mysql-perfschema-agent",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.QANMysqlSlowlogAgent {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "qan-mysql-slowlog-agent",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.MongodbExporter {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "mongodb_exporter",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.PostgresExporter {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "postgres_exporter",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}

	for _, a := range agentsRes.Payload.QANMongodbProfilerAgent {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "qan-mongodb-profiler-agent",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}
	for _, a := range agentsRes.Payload.ProxysqlExporter {
		if _, ok := pmmAgentIDs[a.PMMAgentID]; ok {
			agents = append(agents, listResultAgent{
				AgentType: "proxysql_exporter",
				AgentID:   a.AgentID,
				ServiceID: a.ServiceID,
				Status:    getStatus(a.Status, a.Disabled),
			})
		}
	}

	return &listResult{
		Services: services,
		Agents:   agents,
	}, nil
}

// register command
var (
	List  = new(listCommand)
	ListC = kingpin.Command("list", "Show Services and Agents running on this Node.")
)

func init() {
	ListC.Flag("node-id", "Node ID. Default is autodetected.").StringVar(&List.NodeID)
}
