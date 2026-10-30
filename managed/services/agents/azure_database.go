// Copyright (C) 2023 Percona LLC
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
	"bytes"
	"fmt"
	"text/template"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

const azureDatabaseTemplate = `---
active_directory_authority_url: "https://login.microsoftonline.com/"
resource_manager_url: "https://management.azure.com/"
credentials:
  client_id: "{{ .ClientID}}"
  client_secret: "{{ .ClientSecret}}"
  tenant_id: "{{ .TenantID}}"
  subscription_id: "{{ .SubscriptionID}}"

targets:
resource_groups:
{{- range .ResourceGroups }}
  - resource_group: "{{ $.ResourceGroup }}"
    aggregations:
      - Average
    resource_types:
{{- range .ResourceTypes }}
      - "{{ . }}"
{{- end }}
    metrics:
{{- range .Metrics }}
      - name: "{{ . }}"
{{- end }}
{{- end }}
`

// azureDatabaseExporterConfig returns configuration of azure_database_exporter process.
func azureDatabaseExporterConfig(
	exporter *models.Agent,
	service *models.Service,
	redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) (*agentv1.SetStateRequest_AgentProcess, error) {
	t, err := template.New("credentials").Parse(azureDatabaseTemplate)
	if err != nil {
		return nil, err
	}

	metrics := []string{
		"cpu_percent",
		"memory_percent",
		"storage_percent",
		"storage_used",
		"storage_limit",
		"network_bytes_egress",
		"network_bytes_ingress",
	}

	type resourceGroup struct {
		ResourceTypes []string
		Metrics       []string
	}

	var resourceGroups []resourceGroup
	switch service.ServiceType {
	case models.MySQLServiceType:
		resourceGroups = []resourceGroup{{
			ResourceTypes: []string{
				"Microsoft.DBforMySQL/servers",
				"Microsoft.DBforMySQL/flexibleServers",
				"Microsoft.DBforMariaDB/servers",
			},
			Metrics: metrics,
		}}
	case models.PostgreSQLServiceType:
		// Azure Monitor rejects the whole request if any metric is unknown, and PostgreSQL flexible servers have no storage_limit.
		resourceGroups = []resourceGroup{
			{
				ResourceTypes: []string{
					"Microsoft.DBforPostgreSQL/servers",
					"Microsoft.DBforPostgreSQL/serversv2",
				},
				Metrics: metrics,
			},
			{
				ResourceTypes: []string{"Microsoft.DBforPostgreSQL/flexibleServers"},
				Metrics: []string{
					"cpu_percent",
					"memory_percent",
					"storage_percent",
					"storage_used",
					"network_bytes_egress",
					"network_bytes_ingress",
				},
			},
		}
	default:
		return nil, fmt.Errorf("unexpected service type %s", service.ServiceType)
	}

	var config bytes.Buffer
	options := struct {
		models.AzureOptions

		ResourceGroups []resourceGroup
	}{exporter.AzureOptions, resourceGroups}

	err = t.Execute(&config, options)
	if err != nil {
		return nil, err
	}

	var words []string
	if redactMode != exposeSecrets {
		words = append(words, redactWords(exporter)...)
	}

	tdp := models.TemplateDelimsPair()
	args := []string{
		"--config.file=" + tdp.Left + " .TextFiles.config " + tdp.Right,
		"--web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
	}
	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, true)

	return &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_AZURE_DATABASE_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"config": config.String(),
		},
		RedactWords: words,
	}, nil
}
