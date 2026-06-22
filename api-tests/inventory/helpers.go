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
	"testing"

	"github.com/stretchr/testify/assert"

	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

func assertPostgreSQLServiceExists(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Postgresql {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be PostgreSQL service with id `%s`", serviceID)
}

func assertMySQLServiceExists(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Mysql {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be MySQL service with id `%s`", serviceID)
}

func assertMySQLServiceNotExist(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Mysql {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be MySQL service with id `%s`", serviceID)
}

func assertExternalServiceExists(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.External {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be External service with id `%s`", serviceID)
}

func assertExternalServiceNotExist(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.External {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be External service with id `%s`", serviceID)
}

func assertHAProxyServiceExists(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Haproxy {
			if v.ServiceID == serviceID {
				return true
			}
		}
		return false
	}, "There should be HAProxy service with id `%s`", serviceID)
}

func assertHAProxyServiceNotExist(t *testing.T, res *services.ListServicesOK, serviceID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.Haproxy {
			if v.ServiceID == serviceID {
				return false
			}
		}
		return true
	}, "There should not be HAProxy service with id `%s`", serviceID)
}

func assertMySQLExporterExists(t *testing.T, res *agents.ListAgentsOK, mySqldExporterID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.MysqldExporter {
			if v.AgentID == mySqldExporterID {
				return true
			}
		}
		return false
	}, "There should be MySQL agent with id `%s`", mySqldExporterID)
}

func assertMySQLExporterNotExists(t *testing.T, res *agents.ListAgentsOK, mySqldExporterID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.MysqldExporter {
			if v.AgentID == mySqldExporterID {
				return false
			}
		}
		return true
	}, "There should not be MySQL agent with id `%s`", mySqldExporterID)
}

func assertPMMAgentExists(t *testing.T, res *agents.ListAgentsOK, pmmAgentID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.PMMAgent {
			if v.AgentID == pmmAgentID {
				return true
			}
		}
		return false
	}, "There should be PMM-agent with id `%s`", pmmAgentID)
}

func assertPMMAgentNotExists(t *testing.T, res *agents.ListAgentsOK, pmmAgentID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.PMMAgent {
			if v.AgentID == pmmAgentID {
				return false
			}
		}
		return true
	}, "There should not be PMM-agent with id `%s`", pmmAgentID)
}

func assertNodeExporterExists(t *testing.T, res *agents.ListAgentsOK, nodeExporterID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.NodeExporter {
			if v.AgentID == nodeExporterID {
				return true
			}
		}
		return false
	}, "There should be Node exporter with id `%s`", nodeExporterID)
}

func assertNodeExporterNotExists(t *testing.T, res *agents.ListAgentsOK, nodeExporterID string) {
	t.Helper()

	assert.Conditionf(t, func() bool {
		for _, v := range res.Payload.NodeExporter {
			if v.AgentID == nodeExporterID {
				return false
			}
		}
		return true
	}, "There should not be Node exporter with id `%s`", nodeExporterID)
}
