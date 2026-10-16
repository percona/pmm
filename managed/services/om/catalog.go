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

package om

import (
	"time"

	omv1 "github.com/percona/pmm/api/om/v1"
)

// Every query aggregates or is keyed by service_id: names are reused across re-registrations while
// superseded series persist, so a name-keyed join would mix generations.
const (
	metricVersionInfo = "mongodb_version_info"
	// One series for the asked node's own member_state, unlike the per-peer mongodb_rs_members_state.
	metricMembersSelf = "mongodb_members_self"
	// A down service produces no series rather than a 0, so absence reads as DOWN.
	metricUp = "mongodb_up"
	// Emitted only by a mongos, so its presence identifies a router.
	metricMongosShards = "mongodb_mongos_sharding_shards_total"
	// One series per (reporting node, secondary peer), so it needs a reducer. A primary or a
	// single-member set has none, which means not applicable rather than not collected.
	metricReplicationLag = "mongodb_mongod_replset_member_replication_lag"
	// Oplog bounds as epoch seconds; their difference is the window. A standalone emits neither.
	metricOplogHead = "mongodb_mongod_replset_oplog_head_timestamp"
	metricOplogTail = "mongodb_mongod_replset_oplog_tail_timestamp"
)

// Derived percentages: %[1]s is the series matcher and %[2]s the window. An irate over 30s needs two
// raw samples, so a service that stopped reporting drops out without last_over_time.
const (
	queryCPUUsage = `100 * (1 - sum by (service_id) (irate(mongodb_sys_cpu_idle_ms{%[1]s}[30s]))` +
		` / (1000 * max by (service_id) (last_over_time(mongodb_sys_cpu_num_logical_cores{%[1]s}[%[2]s]))))`
	queryConnectionsFree = `100 * max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="available"}[%[2]s]))` +
		` / (max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="current"}[%[2]s]))` +
		` + max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="available"}[%[2]s])))`
)

// queryLastUp is the newest moment in the window at which mongodb_up read 1, as epoch seconds.
// MetricsQL only: PromQL has no tmax_over_time.
const queryLastUp = `max by (service_id) (tmax_over_time(` + metricUp + `{%[1]s}[%[2]s])` +
	` and max_over_time(` + metricUp + `{%[1]s}[%[2]s]) == 1)`

// metricsLookback is long so a since-unreachable node still contributes its last report, with its age.
// Whether that age is too old is decided per field by volatileFields.
const metricsLookback = "24h"

// highResolutionJob narrows every query to the high-resolution job: mongodb_up arrives once per job, and the
// low-resolution series' age swings up to its 60s interval, which would defeat volatileMaxAge.
const highResolutionJob = `job=~".*_hr$"`

// A volatile observation counts as current for volatileMaxAgeFactor high-resolution scrape intervals,
// but at least minVolatileMaxAge.
const (
	volatileMaxAgeFactor = 4
	minVolatileMaxAge    = 30 * time.Second
)

// sourceQueries is carried in the document to name what it was derived from, in collector order.
var sourceQueries = []string{
	metricVersionInfo,
	metricMembersSelf,
	metricUp,
	metricMongosShards,
	"mongodb_sys_cpu_idle_ms",
	"mongodb_connections",
	metricReplicationLag,
	metricOplogHead,
	metricOplogTail,
}

// Document field names. They match PMM Extensions' om_inventory app, so its facts need no translation.
const (
	fieldHost             = "host"
	fieldServiceType      = "service_type"
	fieldCluster          = "cluster"
	fieldReplicationSet   = "replication_set"
	fieldEnvironment      = "environment"
	fieldVersion          = "version"
	fieldVendor           = "vendor"
	fieldEdition          = "edition"
	fieldState            = "state"
	fieldEndpoint         = "endpoint"
	fieldClusterRole      = "cluster_role"
	fieldExporterUp       = "exporter_up"
	fieldCPUUsage         = "cpu_usage_percent"
	fieldConnectionsFree  = "connections_free_percent"
	fieldIsMongos         = "is_mongos"
	fieldReplicationLag   = "replication_lag_seconds"
	fieldOplogHead        = "oplog_head_timestamp"
	fieldOplogTail        = "oplog_tail_timestamp"
	fieldInstalledVersion = "installed_version"
	fieldConfigPath       = "config_path"
	fieldArgv             = "argv"
	fieldLastUp           = "last_up_at"
)

// volatileFields describe now rather than the service. Read past volatileMaxAge they yield nothing,
// so a stopped database reads DOWN instead of a day-old UP.
var volatileFields = map[string]bool{
	fieldState:           true,
	fieldClusterRole:     true,
	fieldExporterUp:      true,
	fieldCPUUsage:        true,
	fieldConnectionsFree: true,
	fieldIsMongos:        true,
	fieldReplicationLag:  true,
	fieldOplogHead:       true,
	fieldOplogTail:       true,
}

// clusterRoles maps the exporter's cl_role label onto process_role. An unrecognised label means
// PROCESS_ROLE_MONGOD rather than UNSPECIFIED: an unlabelled mongod is the ordinary case.
var clusterRoles = map[string]omv1.ProcessRole{
	"configsvr": omv1.ProcessRole_PROCESS_ROLE_CONFIGSVR,
	"shardsvr":  omv1.ProcessRole_PROCESS_ROLE_SHARDSVR,
}

// unmeasured is the gauge sentinel: a number for every service, and not 0, which is a legitimate reading.
const unmeasured = -1

// unspecified is the internal grouping key for an unset environment or cluster; the document emits null.
const unspecified = "UNSPECIFIED"

// schemaVersion is the version of the emitted document, so a stored snapshot stays readable across changes.
const schemaVersion = 1
