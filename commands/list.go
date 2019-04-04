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
	"github.com/percona/pmm-admin/agentlocal"

	"github.com/percona/pmm/api/inventory/json/client"
	"github.com/percona/pmm/api/inventory/json/client/services"
	"gopkg.in/alecthomas/kingpin.v2"
)

// TODO maybe it is better to use tabwriter there
var listResultT = ParseTemplate(`
Service type   Service name   Service ID
{{ range .Services }}
{{- printf "%-14s" .ServiceType }} {{ printf "%-14s" .ServiceName }} {{ .ServiceID }}
{{ end }}
`)

type listResultService struct {
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
}

type listResult struct {
	Services []listResultService `json:"service"`
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
		status, err := agentlocal.GetStatus()
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
		})
	}
	for _, s := range servicesRes.Payload.Mongodb {
		services = append(services, listResultService{
			ServiceType: "MongoDB",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
		})
	}
	for _, s := range servicesRes.Payload.Postgresql {
		services = append(services, listResultService{
			ServiceType: "PostgreSQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
		})
	}

	return &listResult{
		Services: services,
	}, nil
}

// register command
var (
	List  = new(listCommand)
	ListC = kingpin.Command("list", "Show Agents statuses.")
)

func init() {
	ListC.Flag("node-id", "Default is autodetected.").StringVar(&List.NodeID)
}
