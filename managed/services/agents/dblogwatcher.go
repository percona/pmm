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
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

// dbLogWatcherAgentConfig returns the desired configuration of the database log-watcher built-in agent.
func dbLogWatcherAgentConfig(service *models.Service, agent *models.Agent) *agentv1.SetStateRequest_BuiltinAgent {
	watched := make([]*inventoryv1.WatchedLog, 0, len(agent.LogWatcherOptions.Files))
	for _, f := range agent.LogWatcherOptions.Files {
		watched = append(watched, &inventoryv1.WatchedLog{Path: f.Path, Type: f.Type})
	}

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:        inventoryv1.AgentType_AGENT_TYPE_DB_LOG_WATCHER_AGENT,
		ServiceId:   service.ServiceID,
		ServiceName: service.ServiceName,
		DbSystem:    serviceTypeToDBSystem(service.ServiceType),
		WatchedLogs: watched,
	}
}

// serviceTypeToDBSystem maps a PMM service type to the OpenTelemetry db.system value.
func serviceTypeToDBSystem(t models.ServiceType) string {
	switch t {
	case models.MySQLServiceType:
		return "mysql"
	case models.PostgreSQLServiceType:
		return "postgresql"
	case models.MongoDBServiceType:
		return "mongodb"
	case models.ValkeyServiceType:
		return "valkey"
	case models.ProxySQLServiceType, models.HAProxyServiceType, models.ExternalServiceType:
		return ""
	default:
		return ""
	}
}
