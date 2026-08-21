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
	"cmp"
	"slices"
	"time"

	"github.com/AlekSi/pointer"
	"google.golang.org/protobuf/types/known/wrapperspb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// Four rules the document shape enforces, all of them load-bearing for the UI:
//
//   - A service that inventory knows and no source ever saw is still in the document, as
//     status DOWN. Dropping it would shrink the estate every time something broke.
//   - Grouping is by environment then cluster, where cluster falls back to the replica
//     set. Both keys are emitted as null rather than the internal sentinel, so the
//     document never invents a name.
//   - Numeric gauges use -1 for "not measured", not 0 and not null.
//   - Nothing here is a verdict. The document reports what was observed; health rules
//     read it, they do not live here.

// document is one assembled topology document plus what it took to assemble it.
type document struct {
	summary      *omv1.Summary
	environments []*omv1.Environment
	// observedAt is the newest observation behind the document, or the zero time when
	// nothing datable went into it.
	observedAt time.Time
	// staleServices is how many services carry a volatile fact too old to read.
	staleServices int
}

// buildDocument folds inventory and the merged facts into the topology document.
func buildDocument(services []*models.Service, merged map[string]map[string]MergedField, now time.Time, maxAge time.Duration) document {
	out := document{}
	grouped := make(map[string]map[string][]*omv1.TopologyService)

	for _, service := range services {
		fields := fieldSet{fields: merged[service.ServiceID], now: now, maxAge: maxAge}
		if fields.fields == nil {
			fields.fields = map[string]MergedField{}
		}
		if stale := fields.staleVolatile(); stale > 0 {
			out.staleServices++
		}
		for _, held := range fields.fields {
			if held.ObservedAt != nil && held.ObservedAt.After(out.observedAt) {
				out.observedAt = *held.ObservedAt
			}
		}

		env, cluster := groupingKeys(service, fields)
		if _, ok := grouped[env]; !ok {
			grouped[env] = make(map[string][]*omv1.TopologyService)
		}
		grouped[env][cluster] = append(grouped[env][cluster], serviceDocument(service, fields))
	}

	out.environments = make([]*omv1.Environment, 0, len(grouped))
	for _, env := range sortedKeys(grouped) {
		clusters := make([]*omv1.Cluster, 0, len(grouped[env]))
		for _, cluster := range sortedKeys(grouped[env]) {
			svcs := grouped[env][cluster]
			slices.SortFunc(svcs, func(a, b *omv1.TopologyService) int {
				return cmp.Compare(a.ServiceName, b.ServiceName)
			})
			clusters = append(clusters, &omv1.Cluster{Name: named(cluster), Services: svcs})
		}
		out.environments = append(out.environments, &omv1.Environment{EnvName: named(env), Clusters: clusters})
	}

	out.summary = buildSummary(out.environments)
	return out
}

// serviceDocument builds one service entry.
func serviceDocument(service *models.Service, fields fieldSet) *omv1.TopologyService {
	host := fields.str(fieldHost)
	if host == "" {
		host = service.ServiceName
	}
	serviceType := fields.str(fieldServiceType)
	if serviceType == "" {
		serviceType = string(models.MongoDBServiceType)
	}

	up := pointer.GetFloat64(fields.f64(fieldExporterUp)) == 1

	return &omv1.TopologyService{
		ServiceName:            service.ServiceName,
		Host:                   optional(host),
		Endpoint:               optional(fields.str(fieldEndpoint)),
		ServiceId:              optional(service.ServiceID),
		ServiceType:            optional(serviceType),
		Version:                optional(fields.str(fieldVersion)),
		Vendor:                 optional(fields.str(fieldVendor)),
		Edition:                optional(fields.str(fieldEdition)),
		ReplicationSet:         optional(fields.str(fieldReplicationSet)),
		State:                  optional(fields.str(fieldState)),
		Status:                 serviceStatus(up),
		CpuUsagePercent:        gauge(fields.f64(fieldCPUUsage), up),
		ConnectionsFreePercent: gauge(fields.f64(fieldConnectionsFree), up),
		ProcessRole:            processRole(fields),
		ReplicationLagSeconds:  optionalDouble(fields.f64(fieldReplicationLag)),
		OplogWindowSeconds:     optionalDouble(oplogWindow(fields)),

		// Probe-only, and null wherever no probe has run. Not volatile: an installed
		// binary and a config path are properties of the node, not of this moment, so
		// they keep reporting what the last sweep found rather than blanking between
		// sweeps that are minutes apart.
		InstalledVersion: optional(fields.str(fieldInstalledVersion)),
		ConfigPath:       optional(fields.str(fieldConfigPath)),
		Argv:             optional(fields.str(fieldArgv)),
	}
}

// groupingKeys returns the (environment, cluster) keys a service groups under.
//
// Cluster falls back to the replica set, because a replica set registered without a
// --cluster= string is still a cluster and would otherwise land in one anonymous bucket
// with everything else. The merged fields already carry inventory as the fallback source
// for both, so this reads them once rather than re-deriving the precedence.
func groupingKeys(_ *models.Service, fields fieldSet) (string, string) {
	env := firstNonEmpty(fields.str(fieldEnvironment), unspecified)
	cluster := firstNonEmpty(fields.str(fieldCluster), fields.str(fieldReplicationSet), unspecified)
	return env, cluster
}

// processRole returns a service's process role.
//
// The mongodb_mongos_sharding_shards_total metric is emitted only by a router, so its
// presence is the test -- no port-convention guessing. Otherwise the exporter's cl_role
// names a config or shard server, and anything else is a plain mongod.
func processRole(fields fieldSet) omv1.ProcessRole {
	if fields.truthy(fieldIsMongos) {
		return omv1.ProcessRole_PROCESS_ROLE_MONGOS
	}
	if role, ok := clusterRoles[fields.str(fieldClusterRole)]; ok {
		return role
	}
	return omv1.ProcessRole_PROCESS_ROLE_MONGOD
}

// gauge returns a reading, or the unmeasured sentinel when there is none.
//
// A DOWN service reports the sentinel even if a reading survives in the query window: a
// CPU figure for a process that is not running would be read as current, and a number
// nobody can act on is worse than an explicit "not measured".
func gauge(value *float64, up bool) float64 {
	if !up || value == nil {
		return unmeasured
	}
	return *value
}

// oplogWindow returns the oplog window in seconds, or nil when it cannot be computed.
//
// Head and tail come from the same scrape, so subtracting two separately queried values
// is safe in practice. The head >= tail guard still rejects the pathological case --
// mismatched scrapes, clock skew -- as unknown rather than surfacing a negative duration.
func oplogWindow(fields fieldSet) *float64 {
	head, tail := fields.f64(fieldOplogHead), fields.f64(fieldOplogTail)
	if head == nil || tail == nil || *head < *tail {
		return nil
	}
	window := *head - *tail
	return &window
}

// buildSummary builds the fleet-level counts the list view shows above the table.
//
// The down_services count is the honest headline and is a first-class field rather than
// something every caller re-derives.
//
// The process_role_counts map is keyed by the ProcessRole enum's name rather than its
// number, so it reads as itself in JSON instead of as {"1": 5}.
func buildSummary(environments []*omv1.Environment) *omv1.Summary {
	summary := &omv1.Summary{
		Environments:      int32(len(environments)), //nolint:gosec
		ProcessRoleCounts: make(map[string]int32),
	}
	for _, env := range environments {
		summary.Clusters += int32(len(env.Clusters)) //nolint:gosec
		for _, cluster := range env.Clusters {
			for _, service := range cluster.Services {
				summary.TotalServices++
				if service.Status == omv1.ServiceStatus_SERVICE_STATUS_UP {
					summary.UpServices++
				} else {
					summary.DownServices++
				}
				summary.ProcessRoleCounts[service.ProcessRole.String()]++
			}
		}
	}
	return summary
}

func serviceStatus(up bool) omv1.ServiceStatus {
	if up {
		return omv1.ServiceStatus_SERVICE_STATUS_UP
	}
	return omv1.ServiceStatus_SERVICE_STATUS_DOWN
}

// named returns a grouping key as the document spells it: null for the internal
// sentinel, the key itself otherwise.
func named(key string) *wrapperspb.StringValue {
	if key == unspecified {
		return nil
	}
	return wrapperspb.String(key)
}

// optional returns nil for an empty string, so the document says null rather than "".
//
// The wrapper type is what makes that null reachable: protojson treats a proto3
// `optional` scalar as a synthetic oneof and omits the key entirely when it is unset,
// even under EmitUnpopulated, while an unset message field marshals to null. The
// document's contract distinguishes the two, so the field has to be a message.
func optional(value string) *wrapperspb.StringValue {
	if value == "" {
		return nil
	}
	return wrapperspb.String(value)
}

// optionalDouble is optional's counterpart for a gauge that may not apply at all.
func optionalDouble(value *float64) *wrapperspb.DoubleValue {
	if value == nil {
		return nil
	}
	return wrapperspb.Double(*value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
