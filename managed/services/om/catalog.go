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

// The signal catalog: exactly the queries the topology document needs, and nothing
// speculative. Every query aggregates or is keyed by service_id, which is the only safe
// join key -- a service name is reused across re-registrations while the superseded
// series live on until retention expires, so a name-keyed join silently mixes
// generations.
const (
	// The exporter's info metric: identity, running version, vendor and edition.
	metricVersionInfo = "mongodb_version_info"
	// The replSetGetStatus view as seen from the member itself -- its own member_state.
	// The sibling mongodb_rs_members_state carries one series per peer; this one is a
	// single series describing the node being asked.
	metricMembersSelf = "mongodb_members_self"
	// The exporter's own reachability flag. A service that is down produces no series at
	// all rather than a 0, so absence is what the document reads as DOWN.
	metricUp = "mongodb_up"
	// Emitted only by a mongos, so its presence is the router test -- more reliable than
	// the port convention, which only ever guessed.
	metricMongosShards = "mongodb_mongos_sharding_shards_total"
	// Seconds a peer trails the primary's optime. Several series per service, one per
	// (reporting node, secondary peer), so it needs a reducer. The primary appears in no
	// series by construction, and a single-member set emits nothing at all -- which is
	// "not applicable" rather than "not collected".
	metricReplicationLag = "mongodb_mongod_replset_member_replication_lag"
	// Oplog bounds as epoch seconds; their difference is the window. One series per
	// service each. A standalone emits neither -- no replica set, no oplog.
	metricOplogHead = "mongodb_mongod_replset_oplog_head_timestamp"
	metricOplogTail = "mongodb_mongod_replset_oplog_tail_timestamp"
)

// Derived percentages. Neither exists as a series, so both are expressions. %[1]s is the
// series matcher -- the service_id set and highResolutionJob -- and %[2]s the freshness
// window; both aggregate by (service_id) so the result still joins back to a service.
//
// The CPU expression needs no last_over_time: irate over a 30s range already requires
// two raw samples inside it, so a service that stopped reporting drops out on its own.
const (
	queryCPUUsage = `100 * (1 - sum by (service_id) (irate(mongodb_sys_cpu_idle_ms{%[1]s}[30s]))` +
		` / (1000 * max by (service_id) (last_over_time(mongodb_sys_cpu_num_logical_cores{%[1]s}[%[2]s]))))`
	queryConnectionsFree = `100 * max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="available"}[%[2]s]))` +
		` / (max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="current"}[%[2]s]))` +
		` + max by (service_id) (last_over_time(mongodb_connections{%[1]s,state="available"}[%[2]s])))`
)

// metricsLookback is the window every query is read over.
//
// Deliberately long, and deliberately not the freshness rule. Without an explicit window
// this would be VictoriaMetrics' instant-query lookbehind of five minutes, so a query
// returns nothing whenever scraping paused -- indistinguishable from "no such service".
// Reading over a day instead means a since-unreachable node still contributes what it
// last reported, carrying the age that says so. Whether that age is too old to act on is
// a separate decision, made per field by volatileFields and volatileMaxAge.
const metricsLookback = "24h"

// highResolutionJob narrows every query to the high-resolution scrape job, which is what
// makes volatileMaxAge meaningful.
//
// Each exporter is scraped once per resolution, so a metric it emits on every scrape
// regardless of collector flags arrives once per job. That is what mongodb_up is: an hr and
// an lr series per service. The lr series' age swings up to its own 60s interval, so a
// volatile fact read from it is stale for most of every minute while the hr sample is
// seconds old -- and sizing volatileMaxAge off lr instead would blunt every volatile field
// to minutes to accommodate one metric. The management package selects its own "up" metrics
// the same way.
//
// What this does not make true is "one series per service per metric", which nothing does:
// an exporter's label set changes over its own lifetime -- cluster_role appears once it can
// determine the role -- so it abandons a series and begins another under the same
// service_id. Pairing each sample with its own series' age is what handles that, not this
// matcher; see seriesSample.supersedes.
const highResolutionJob = `job=~".*_hr$"`

// How old a volatile observation may be and still count as current.
//
// Sized against the scrape interval rather than picked: every query carries
// highResolutionJob, so the window is volatileMaxAgeFactor of PMM's configured high
// resolution, floored by minVolatileMaxAge. Four intervals tolerates several missed
// scrapes while still noticing a stopped database in well under a minute.
const (
	volatileMaxAgeFactor = 4
	minVolatileMaxAge    = 30 * time.Second
)

// sourceQueries names what the document was derived from, and is carried in the document
// so a reader can see that without reading this file. Ordered as the collector issues
// them.
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

// The document fields sources set. Names match SEP's om_inventory app so a fact
// produced there needs no translation on the way in.
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
)

// volatileFields are the fields that describe now rather than describe the service.
//
// Everything is collected over metricsLookback and kept with its age, so the document can
// still say what a since-unreachable node last reported. But these fields are only
// meaningful while fresh: a replica-set state or an up flag from an hour ago is not a
// fact about the present, whereas an edition is. Reading one past volatileMaxAge yields
// nothing, which is what turns a stopped database into status DOWN instead of a
// confident, day-old UP.
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

// clusterRoles maps the exporter's cl_role label onto the document's process_role.
//
// The label is the exporter's vocabulary and the enum is ours, so this is the only place
// the two meet. A label we do not recognise falls through to PROCESS_ROLE_MONGOD rather
// than UNSPECIFIED: an unlabelled mongod is the ordinary case, not a gap.
var clusterRoles = map[string]omv1.ProcessRole{
	"configsvr": omv1.ProcessRole_PROCESS_ROLE_CONFIGSVR,
	"shardsvr":  omv1.ProcessRole_PROCESS_ROLE_SHARDSVR,
}

// unmeasured is the gauge sentinel: -1 rather than null so the field stays a number for
// every service, and rather than 0 because zero CPU is a legitimate reading.
const unmeasured = -1

// unspecified is the grouping key for a service whose environment or cluster is unset.
// Internal only -- the document emits null for it, because inventing the string
// "UNSPECIFIED" as a name would make it indistinguishable from a real environment
// called that.
const unspecified = "UNSPECIFIED"

// schemaVersion is the version of the emitted document. Versioned so that a stored
// snapshot stays readable across changes.
//
// Starts at 1: nothing has shipped yet, so there is no prior version's snapshot a reader
// needs to keep distinguishing from this one. A dev database is one `./om reset data`
// away, which is what makes 1 the honest number rather than leftover numbering from
// development.
const schemaVersion = 1
