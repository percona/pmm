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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

func addRemoteRDSNode(t pmmapitests.TestingT, nodeName string) *nodes.AddRemoteRDSNodeOKBody {
	t.Helper()

	params := &nodes.AddRemoteRDSNodeParams{
		Body: nodes.AddRemoteRDSNodeBody{
			NodeName: nodeName,
			Address:  "some-address",
			Region:   "region",
		},

		Context: pmmapitests.Context,
	}
	res, err := client.Default.Nodes.AddRemoteRDSNode(params)
	assert.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addRDSExporter(t pmmapitests.TestingT, body agents.AddRDSExporterBody) *agents.AddRDSExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddRDSExporter(&agents.AddRDSExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addRemoteAzureDatabaseNode(t pmmapitests.TestingT, nodeName string) *nodes.AddRemoteAzureDatabaseNodeOKBody {
	t.Helper()

	params := &nodes.AddRemoteAzureDatabaseNodeParams{
		Body: nodes.AddRemoteAzureDatabaseNodeBody{
			NodeName: nodeName,
			Address:  "some-address",
			Region:   "region",
		},

		Context: pmmapitests.Context,
	}
	res, err := client.Default.Nodes.AddRemoteAzureDatabaseNode(params)
	assert.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addAzureDatabaseExporter(t pmmapitests.TestingT, body agents.AddAzureDatabaseExporterBody) *agents.AddAzureDatabaseExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddAzureDatabaseExporter(&agents.AddAzureDatabaseExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)

	return res.Payload
}

func addMySQLService(t pmmapitests.TestingT, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addMongoDBService(t pmmapitests.TestingT, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addPostgreSQLService(t pmmapitests.TestingT, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addProxySQLService(t pmmapitests.TestingT, body services.AddServiceBody) *services.AddServiceOKBody {
	t.Helper()

	params := &services.AddServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addExternalService(t pmmapitests.TestingT, body services.AddExternalServiceBody) *services.AddExternalServiceOKBody {
	t.Helper()

	params := &services.AddExternalServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddExternalService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addHAProxyService(t pmmapitests.TestingT, body services.AddHAProxyServiceBody) *services.AddHAProxyServiceOKBody {
	t.Helper()

	params := &services.AddHAProxyServiceParams{
		Body:    body,
		Context: pmmapitests.Context,
	}
	res, err := client.Default.Services.AddHAProxyService(params)
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addNodeExporter(t pmmapitests.TestingT, pmmAgentID string, customLabels map[string]string) *agents.AddNodeExporterOK {
	res, err := client.Default.Agents.AddNodeExporter(&agents.AddNodeExporterParams{
		Body: agents.AddNodeExporterBody{
			PMMAgentID:   pmmAgentID,
			CustomLabels: customLabels,
		},
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Payload.NodeExporter)
	require.Equal(t, pmmAgentID, res.Payload.NodeExporter.PMMAgentID)
	return res
}

func addMySQLdExporter(t pmmapitests.TestingT, body agents.AddMySQLdExporterBody) *agents.AddMySQLdExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddMySQLdExporter(&agents.AddMySQLdExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addMongoDBExporter(t pmmapitests.TestingT, body agents.AddMongoDBExporterBody) *agents.AddMongoDBExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddMongoDBExporter(&agents.AddMongoDBExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addPostgresExporter(t pmmapitests.TestingT, body agents.AddPostgresExporterBody) *agents.AddPostgresExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddPostgresExporter(&agents.AddPostgresExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addProxySQLExporter(t pmmapitests.TestingT, body agents.AddProxySQLExporterBody) *agents.AddProxySQLExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddProxySQLExporter(&agents.AddProxySQLExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.NotNil(t, res)
	return res.Payload
}

func addExternalExporter(t pmmapitests.TestingT, body agents.AddExternalExporterBody) *agents.AddExternalExporterOKBody {
	t.Helper()

	res, err := client.Default.Agents.AddExternalExporter(&agents.AddExternalExporterParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
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
	}, "There should be MySQL agent with id `%s`", mySqldExporterID)
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
	}, "There should be PMM-agent with id `%s`", pmmAgentID)
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
	}, "There should be Node exporter with id `%s`", nodeExporterID)
}
