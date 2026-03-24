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

// Package inventory contains inventory API tests.
package inventory

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

func addRemoteRDSNode(t pmmapitests.TestingT, nodeName string) *nodes.AddNodeOKBody {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			RemoteRDS: &nodes.AddNodeParamsBodyRemoteRDS{
				NodeName: nodeName,
				Address:  pmmapitests.TestString(t, "rds-address"),
				Region:   pmmapitests.TestString(t, "rds-region"),
			},
		},
		Context: pmmapitests.Context,
	}
	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addRemoteAzureDatabaseNode(t pmmapitests.TestingT, nodeName string) *nodes.AddNodeOKBody {
	t.Helper()

	params := &nodes.AddNodeParams{
		Body: nodes.AddNodeBody{
			RemoteAzure: &nodes.AddNodeParamsBodyRemoteAzure{
				NodeName: nodeName,
				Address:  pmmapitests.TestString(t, "azure-address"),
				Region:   pmmapitests.TestString(t, "azure-region"),
			},
		},
		Context: pmmapitests.Context,
	}
	res, err := client.Default.NodesService.AddNode(params)
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addService(t pmmapitests.TestingT, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}

	res, err := client.Default.ServicesService.AddService(params)
	require.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addNodeExporter(t pmmapitests.TestingT, pmmAgentID string, customLabels map[string]string) *agents.AddAgentOK {
	res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
		Body: agents.AddAgentBody{
			NodeExporter: &agents.AddAgentParamsBodyNodeExporter{
				PMMAgentID:   pmmAgentID,
				CustomLabels: customLabels,
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload.NodeExporter)
	require.Equal(t, pmmAgentID, res.Payload.NodeExporter.PMMAgentID)
	return res
}

func addAgent(t pmmapitests.TestingT, body agents.AddAgentBody) *agents.AddAgentOKBody {
	t.Helper()

	res, err := client.Default.AgentsService.AddAgent(&agents.AddAgentParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func assertPostgreSQLServiceExists(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool { //nolint:unparam
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Postgresql {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be PostgreSQL service with id `%s`", serviceID)
}

func assertMySQLServiceExists(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool { //nolint:unparam
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Mysql {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be MySQL service with id `%s`", serviceID)
}

func assertMySQLServiceNotExist(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool { //nolint:unparam
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Mysql {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be MySQL service with id `%s`", serviceID)
}

func assertExternalServiceExists(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool { //nolint:unparam
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.External {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be External service with id `%s`", serviceID)
}

func assertExternalServiceNotExist(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool {
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.External {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be External service with id `%s`", serviceID)
}

func assertHAProxyServiceExists(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool { //nolint:unparam
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Haproxy {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be HAProxy service with id `%s`", serviceID)
}

func assertHAProxyServiceNotExist(t pmmapitests.TestingT, res *services.ListServicesOK, serviceID string) bool {
	t.Helper()

	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Haproxy {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be HAProxy service with id `%s`", serviceID)
}

func assertMySQLExporterExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, mySqldExporterID string) bool { //nolint:unparam
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.MysqldExporter {
			if v.AgentID == mySqldExporterID {
				return true
			}
		}
		return false
	}, "There should be MySQL agent with id `%s`", mySqldExporterID)
}

func assertMySQLExporterNotExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, mySqldExporterID string) bool {
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.MysqldExporter {
			if v.AgentID == mySqldExporterID {
				return false
			}
		}
		return true
	}, "There should not be MySQL agent with id `%s`", mySqldExporterID)
}

func assertPMMAgentExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, pmmAgentID string) bool {
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.PMMAgent {
			if v.AgentID == pmmAgentID {
				return true
			}
		}
		return false
	}, "There should be PMM-agent with id `%s`", pmmAgentID)
}

func assertPMMAgentNotExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, pmmAgentID string) bool { //nolint:unparam
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.PMMAgent {
			if v.AgentID == pmmAgentID {
				return false
			}
		}
		return true
	}, "There should not be PMM-agent with id `%s`", pmmAgentID)
}

func assertNodeExporterExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, nodeExporterID string) bool { //nolint:unparam
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.NodeExporter {
			if v.AgentID == nodeExporterID {
				return true
			}
		}
		return false
	}, "There should be Node exporter with id `%s`", nodeExporterID)
}

func assertNodeExporterNotExists(t pmmapitests.TestingT, res *agents.ListAgentsOK, nodeExporterID string) bool { //nolint:unparam
	return assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.NodeExporter {
			if v.AgentID == nodeExporterID {
				return false
			}
		}
		return true
	}, "There should not be Node exporter with id `%s`", nodeExporterID)
}
