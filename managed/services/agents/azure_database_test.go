// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package agents

import (
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
)

func TestAzureExporterConfig(t *testing.T) {
	node1 := &models.Node{
		NodeID:    "/node_id/node1",
		NodeType:  models.RemoteAzureDatabaseNodeType,
		NodeName:  "prod-mysql56",
		NodeModel: "B_Gen5_1",
		Region:    pointer.ToString("eastus"),
		AZ:        "eastus-1c",
		Address:   "pmm-dev-mysql-db1.mysql.database.azure.com",
	}
	err := node1.SetCustomLabels(map[string]string{
		"foo": "bar",
	})
	require.NoError(t, err)

	service1 := &models.Service{
		ServiceID:   "/service_id/service1",
		NodeID:      node1.NodeID,
		Address:     pointer.ToString("pmm-dev-mysql-db1.mysql.database.azure.com"),
		Port:        pointer.ToUint16(3306),
		ServiceName: "service1",
		ServiceType: models.MySQLServiceType,
	}

	agent := &models.Agent{
		AgentID:   "/agent_id/agent1",
		AgentType: models.AzureDatabaseExporterType,
		NodeID:    &node1.NodeID,
		ServiceID: &service1.ServiceID,
		AzureOptions: &models.AzureOptions{
			ClientID:       "azure_database_client_id",
			ClientSecret:   "azure_database_client_secret",
			TenantID:       "azure_database_tenant_id",
			SubscriptionID: "azure_database_subscription_id",
			ResourceGroup:  "azure_database_resource_group",
		},
	}

	actual, err := azureDatabaseExporterConfig(agent, service1, redactSecrets)
	require.NoError(t, err)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_AZURE_DATABASE_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--config.file={{ .TextFiles.config }}",
			"--web.listen-address=:{{ .listen_port }}",
		},
		TextFiles: map[string]string{
			`config`: strings.TrimSpace(`---
active_directory_authority_url: "https://login.microsoftonline.com/"
resource_manager_url: "https://management.azure.com/"
credentials:
  client_id: "azure_database_client_id"
  client_secret: "azure_database_client_secret"
  tenant_id: "azure_database_tenant_id"
  subscription_id: "azure_database_subscription_id"

targets:
resource_groups:
  - resource_group: "azure_database_resource_group"
    aggregations:
      - Average
    resource_types:
      - "Microsoft.DBforMySQL/servers"
      - "Microsoft.DBforMySQL/felexibleServers"
      - "Microsoft.DBforMariaDB/servers"
    metrics:
      - name: "cpu_percent"
      - name: "memory_percent"
      - name: "storage_percent"
      - name: "storage_used"
      - name: "storage_limit"
      - name: "network_bytes_egress"
      - name: "network_bytes_ingress"
			`) + "\n",
		},
		RedactWords: []string{"azure_database_client_secret"},
	}
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected.TextFiles["config"], actual.TextFiles["config"])
	require.Equal(t, expected, actual)
}
