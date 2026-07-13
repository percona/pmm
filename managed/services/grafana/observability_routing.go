// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package grafana

import (
	"fmt"
	"strings"
)

// Engine identifies a monitored technology for observability routing.
type Engine string

// Intent classifies the investigation focus for observability routing.
type Intent string

const (
	EngineMySQL      Engine = "mysql" //nolint:revive
	EnginePostgreSQL Engine = "postgresql"
	EngineMongoDB    Engine = "mongodb"
	EngineValkey     Engine = "valkey"
	EngineNode       Engine = "node"

	IntentWorkload         Intent = "workload"
	IntentConnections      Intent = "connections"
	IntentSlowQueries      Intent = "slow_queries"
	IntentReplication      Intent = "replication"
	IntentGroupReplication Intent = "group_replication"
	IntentInnoDB           Intent = "innodb"
	IntentWAL              Intent = "wal"
	IntentLocks            Intent = "locks"
	IntentLatency          Intent = "latency"
	IntentMemory           Intent = "memory"
	IntentCPUMemory        Intent = "cpu_memory"
	IntentDiskIO           Intent = "disk_io"
	IntentNetwork          Intent = "network"
	IntentAvailability     Intent = "availability"
	IntentOverview         Intent = "overview"
)

type secondaryRoute struct {
	DashboardUID string `json:"dashboard_uid"`
	UseWhen      string `json:"use_when"`
}

type observabilityRoute struct {
	DashboardUID   string
	Title          string
	UseWhen        string
	PanelIDs       []int
	Secondary      []secondaryRoute
	FallbackPrefix string
}

var observabilityRoutes = map[Engine]map[Intent]observabilityRoute{
	EngineMySQL: {
		IntentWorkload: {
			DashboardUID:   "mysql-instance-summary", //nolint:goconst
			Title:          "MySQL Instance Summary",
			UseWhen:        "General instance workload, connections, slow queries, host CPU/disk",
			PanelIDs:       []int{8, 92, 48, 337, 341},
			FallbackPrefix: "mysql_", //nolint:goconst
		},
		IntentConnections: {
			DashboardUID:   "mysql-instance-summary",
			Title:          "MySQL Instance Summary",
			UseWhen:        "Connection and thread saturation",
			PanelIDs:       []int{92, 8},
			FallbackPrefix: "mysql_",
		},
		IntentSlowQueries: {
			DashboardUID:   "mysql-instance-summary",
			Title:          "MySQL Instance Summary",
			UseWhen:        "Slow query volume",
			PanelIDs:       []int{48},
			FallbackPrefix: "mysql_",
		},
		IntentReplication: {
			DashboardUID:   "mysql-replicaset-summary",
			Title:          "MySQL Replication Summary",
			UseWhen:        "Async replication lag and binlog health",
			PanelIDs:       []int{16, 17, 26, 33, 35},
			FallbackPrefix: "mysql_",
		},
		IntentGroupReplication: {
			DashboardUID:   "mysql-group-replicaset-summary",
			Title:          "MySQL Group Replication Summary",
			UseWhen:        "Group Replication member state",
			PanelIDs:       []int{1016, 1044},
			FallbackPrefix: "mysql_",
		},
		IntentInnoDB: {
			DashboardUID: "mysql-innodb",
			Title:        "MySQL InnoDB Details",
			UseWhen:      "InnoDB buffer pool, locks, redo — cap panels to avoid bloat",
			PanelIDs:     []int{3, 47, 69, 133, 140, 207},
			Secondary: []secondaryRoute{
				{DashboardUID: "mysql-instance-summary", UseWhen: "Instance-level context"},
			},
			FallbackPrefix: "mysql_",
		},
		IntentLocks: {
			DashboardUID:   "mysql-innodb",
			Title:          "MySQL InnoDB Details",
			UseWhen:        "Row lock waits and lock activity",
			PanelIDs:       []int{47, 69, 207, 208, 209},
			FallbackPrefix: "mysql_",
		},
		IntentOverview: {
			DashboardUID:   "mysql-instance-overview",
			Title:          "MySQL Instances Overview",
			UseWhen:        "Fleet-wide MySQL comparison",
			PanelIDs:       []int{9, 10, 11, 12, 13},
			FallbackPrefix: "mysql_",
		},
		IntentAvailability: {
			DashboardUID:   "mysql-instance-summary",
			Title:          "MySQL Instance Summary",
			UseWhen:        "Use mysql_up instant check; panels optional",
			PanelIDs:       []int{92},
			FallbackPrefix: "mysql_up",
		},
	},
	EnginePostgreSQL: {
		IntentWorkload: {
			DashboardUID:   "postgresql-instance-summary", //nolint:goconst
			Title:          "PostgreSQL Instance Summary", //nolint:goconst
			UseWhen:        "General PostgreSQL workload",
			PanelIDs:       []int{23, 1025, 1057, 337, 341},
			FallbackPrefix: "pg_", //nolint:goconst
		},
		IntentConnections: {
			DashboardUID:   "postgresql-instance-summary",
			Title:          "PostgreSQL Instance Summary",
			UseWhen:        "Connections and session state",
			PanelIDs:       []int{23, 1015, 1019},
			FallbackPrefix: "pg_",
		},
		IntentLocks: {
			DashboardUID:   "postgresql-instance-summary",
			Title:          "PostgreSQL Instance Summary",
			UseWhen:        "Locks, deadlocks, conflicts",
			PanelIDs:       []int{61, 1071, 1073},
			FallbackPrefix: "pg_",
		},
		IntentWAL: {
			DashboardUID:   "postgresql-instance-summary",
			Title:          "PostgreSQL Instance Summary",
			UseWhen:        "Write/WAL pressure — no dedicated WAL panel; use txn/disk panels + pg_*wal* fallback",
			PanelIDs:       []int{1057, 1049, 1033, 1027},
			FallbackPrefix: "pg_",
		},
		IntentSlowQueries: {
			DashboardUID:   "postgresql-instance-summary",
			Title:          "PostgreSQL Instance Summary",
			UseWhen:        "Slow queries and query volume",
			PanelIDs:       []int{1013, 1021},
			FallbackPrefix: "pg_",
		},
		IntentOverview: {
			DashboardUID:   "postgresql-instance-overview",
			Title:          "PostgreSQL Instances Overview",
			UseWhen:        "Fleet-wide PostgreSQL comparison",
			PanelIDs:       []int{1065, 1077, 1079},
			FallbackPrefix: "pg_",
		},
		IntentAvailability: {
			DashboardUID:   "postgresql-instance-summary",
			Title:          "PostgreSQL Instance Summary",
			UseWhen:        "Use postgresql_up instant check",
			PanelIDs:       []int{23},
			FallbackPrefix: "postgresql_up",
		},
	},
	EngineMongoDB: {
		IntentWorkload: {
			DashboardUID:   "mongodb-instance-summary", //nolint:goconst
			Title:          "MongoDB Instance Summary", //nolint:goconst
			UseWhen:        "General MongoDB workload",
			PanelIDs:       []int{15, 38, 1005, 1014, 337, 341},
			FallbackPrefix: "mongodb_", //nolint:goconst
		},
		IntentConnections: {
			DashboardUID:   "mongodb-instance-summary",
			Title:          "MongoDB Instance Summary",
			UseWhen:        "Connections",
			PanelIDs:       []int{38},
			FallbackPrefix: "mongodb_",
		},
		IntentLatency: {
			DashboardUID:   "mongodb-instance-summary",
			Title:          "MongoDB Instance Summary",
			UseWhen:        "Operation latency",
			PanelIDs:       []int{1007, 1014},
			FallbackPrefix: "mongodb_",
		},
		IntentReplication: {
			DashboardUID:   "mongodb-instance-summary",
			Title:          "MongoDB Instance Summary",
			UseWhen:        "Replica set state",
			PanelIDs:       []int{1016, 1018, 1020},
			FallbackPrefix: "mongodb_",
		},
		IntentOverview: {
			DashboardUID:   "mongodb-instance-overview",
			Title:          "MongoDB Instances Overview",
			UseWhen:        "Fleet-wide MongoDB comparison",
			PanelIDs:       []int{36, 38, 40},
			FallbackPrefix: "mongodb_",
		},
		IntentAvailability: {
			DashboardUID:   "mongodb-instance-summary",
			Title:          "MongoDB Instance Summary",
			UseWhen:        "Use mongodb_up instant check",
			PanelIDs:       []int{38},
			FallbackPrefix: "mongodb_up",
		},
	},
	EngineValkey: {
		IntentWorkload: {
			DashboardUID: "valkey-overview",
			Title:        "Valkey Overview",
			UseWhen:      "General Valkey/Redis workload",
			PanelIDs:     []int{7, 16, 10, 27, 632},
			Secondary: []secondaryRoute{
				{DashboardUID: "valkey-load", UseWhen: "Command rate drill-down"},
			},
			FallbackPrefix: "redis_", //nolint:goconst
		},
		IntentConnections: {
			DashboardUID:   "valkey-clients",
			Title:          "Valkey Clients",
			UseWhen:        "Connected/blocked clients",
			PanelIDs:       []int{627, 30, 66, 67},
			FallbackPrefix: "redis_",
		},
		IntentLatency: {
			DashboardUID:   "valkey-overview",
			Title:          "Valkey Overview",
			UseWhen:        "Command latency",
			PanelIDs:       []int{629, 632},
			FallbackPrefix: "redis_",
		},
		IntentSlowQueries: {
			DashboardUID:   "valkey-slowlog",
			Title:          "Valkey Slowlog",
			UseWhen:        "Slow command log",
			PanelIDs:       []int{99, 100, 101, 102},
			FallbackPrefix: "redis_",
		},
		IntentReplication: {
			DashboardUID:   "valkey-replication",
			Title:          "Valkey Replication",
			UseWhen:        "Replica offsets and resyncs",
			PanelIDs:       []int{23, 53, 62, 63, 65},
			FallbackPrefix: "redis_",
		},
		IntentNetwork: {
			DashboardUID:   "valkey-network",
			Title:          "Valkey Network",
			UseWhen:        "Network throughput",
			PanelIDs:       []int{628, 629},
			FallbackPrefix: "redis_",
		},
		IntentMemory: {
			DashboardUID:   "valkey-memory",
			Title:          "Valkey Memory",
			UseWhen:        "Memory usage and evictions",
			PanelIDs:       []int{31, 25, 8, 626},
			FallbackPrefix: "redis_",
		},
		IntentAvailability: {
			DashboardUID:   "valkey-overview",
			Title:          "Valkey Overview",
			UseWhen:        "Use redis_up instant check (PMM uses redis_exporter for Valkey)",
			PanelIDs:       []int{16},
			FallbackPrefix: "redis_up",
		},
	},
	EngineNode: {
		IntentCPUMemory: {
			DashboardUID:   "node-instance-summary",
			Title:          "Node Summary",
			UseWhen:        "Host CPU and memory",
			PanelIDs:       []int{2, 29, 33, 57},
			FallbackPrefix: "node_",
		},
		IntentDiskIO: {
			DashboardUID:   "node-instance-summary",
			Title:          "Node Summary",
			UseWhen:        "Disk I/O and space",
			PanelIDs:       []int{51, 61, 38},
			FallbackPrefix: "node_",
		},
		IntentNetwork: {
			DashboardUID:   "node-instance-summary",
			Title:          "Node Summary",
			UseWhen:        "Network traffic and errors",
			PanelIDs:       []int{21, 22, 52, 53},
			FallbackPrefix: "node_",
		},
		IntentOverview: {
			DashboardUID:   "node-instance-summary",
			Title:          "Node Summary",
			UseWhen:        "General node health",
			PanelIDs:       []int{2, 6, 21, 23},
			FallbackPrefix: "node_",
		},
	},
}

// ParseEngine normalizes engine query values from inventory or Holmes tools.
func ParseEngine(raw string) (Engine, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mysql", "service_type_mysql_service":
		return EngineMySQL, nil
	case "postgresql", "postgres", "service_type_postgresql_service":
		return EnginePostgreSQL, nil
	case "mongodb", "mongo", "service_type_mongodb_service":
		return EngineMongoDB, nil
	case "valkey", "redis", "service_type_valkey_service":
		return EngineValkey, nil
	case "node", "generic", "service_type_generic_service":
		return EngineNode, nil
	default:
		return "", fmt.Errorf("unknown engine %q", raw)
	}
}

// ParseIntent normalizes intent query values.
func ParseIntent(raw string) (Intent, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "workload":
		return IntentWorkload, nil
	case "connections", "threads":
		return IntentConnections, nil
	case "slow_queries", "slow_query":
		return IntentSlowQueries, nil
	case "replication":
		return IntentReplication, nil
	case "group_replication", "gr":
		return IntentGroupReplication, nil
	case "innodb":
		return IntentInnoDB, nil
	case "wal":
		return IntentWAL, nil
	case "locks":
		return IntentLocks, nil
	case "latency":
		return IntentLatency, nil
	case "memory":
		return IntentMemory, nil
	case "cpu_memory", "cpu", "memory_host":
		return IntentCPUMemory, nil
	case "disk_io", "disk":
		return IntentDiskIO, nil
	case "network":
		return IntentNetwork, nil
	case "availability", "up":
		return IntentAvailability, nil
	case "overview":
		return IntentOverview, nil
	default:
		return "", fmt.Errorf("unknown intent %q", raw)
	}
}

// LookupRoute returns the curated route for engine+intent.
func LookupRoute(engine Engine, intent Intent) (observabilityRoute, error) { //nolint:revive
	byIntent, ok := observabilityRoutes[engine]
	if !ok {
		return observabilityRoute{}, fmt.Errorf("no routes for engine %q", engine)
	}
	route, ok := byIntent[intent]
	if !ok {
		return observabilityRoute{}, fmt.Errorf("no route for engine %q intent %q", engine, intent)
	}
	return route, nil
}

func scopedSeriesMatch(serviceID, prefix string) string {
	if serviceID == "" {
		return fmt.Sprintf("{__name__=~\"%s.*\"}", prefix)
	}
	return fmt.Sprintf("{service_id=\"%s\", __name__=~\"%s.*\"}", serviceID, prefix)
}
