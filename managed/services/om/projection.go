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
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/AlekSi/pointer"
	"google.golang.org/protobuf/types/known/timestamppb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

type document struct {
	summary       *omv1.Summary
	environments  []*omv1.Environment
	observedAt    time.Time
	staleServices int
}

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
			clusters = append(clusters, &omv1.Cluster{
				Name:     named(cluster),
				Services: svcs,
				Id:       clusterID(env, cluster),
				Type:     clusterType(svcs),
			})
		}
		out.environments = append(out.environments, &omv1.Environment{EnvName: named(env), Clusters: clusters})
	}

	out.summary = buildSummary(out.environments)
	return out
}

// applyHealth is the stage for health verdicts, kept out of buildDocument, which only reports observations.
func applyHealth(doc document) document {
	return doc
}

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

		InstalledVersion: optional(fields.str(fieldInstalledVersion)),
		ConfigPath:       optional(fields.str(fieldConfigPath)),
		Argv:             optional(fields.str(fieldArgv)),

		ObservedAt: observedAtTimestamp(fields.newestObservedAt()),

		LastUpAt: lastUpAt(fields, up),
	}
}

func lastUpAt(fields fieldSet, up bool) *timestamppb.Timestamp {
	seconds := fields.f64(fieldLastUp)
	if up || seconds == nil {
		return nil
	}
	return timestamppb.New(time.Unix(0, int64(*seconds*float64(time.Second))))
}

func observedAtTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// clusterID is derived rather than random so a cluster keeps the same id across runs without being stored.
func clusterID(env, groupKey string) string {
	sum := sha256.Sum256([]byte(env + "\x00" + groupKey))
	return hex.EncodeToString(sum[:])[:16]
}

func clusterType(services []*omv1.TopologyService) omv1.ClusterType {
	replicationSets := make(map[string]bool)
	mongos := false
	for _, svc := range services {
		if svc.GetProcessRole() == omv1.ProcessRole_PROCESS_ROLE_MONGOS {
			mongos = true
		}
		if rs := svc.GetReplicationSet(); rs != "" {
			replicationSets[rs] = true
		}
	}
	switch {
	case mongos || len(replicationSets) > 1:
		return omv1.ClusterType_CLUSTER_TYPE_SHARDED
	case len(replicationSets) == 1:
		return omv1.ClusterType_CLUSTER_TYPE_REPLICA_SET
	default:
		return omv1.ClusterType_CLUSTER_TYPE_STANDALONE
	}
}

// groupingKeys falls back to the replica set for the cluster: a replica set registered without --cluster
// would otherwise share one anonymous bucket with everything else.
func groupingKeys(_ *models.Service, fields fieldSet) (string, string) {
	env := firstNonEmpty(fields.str(fieldEnvironment), unspecified)
	cluster := firstNonEmpty(fields.str(fieldCluster), fields.str(fieldReplicationSet), unspecified)
	return env, cluster
}

func processRole(fields fieldSet) omv1.ProcessRole {
	if fields.truthy(fieldIsMongos) {
		return omv1.ProcessRole_PROCESS_ROLE_MONGOS
	}
	if role, ok := clusterRoles[fields.str(fieldClusterRole)]; ok {
		return role
	}
	return omv1.ProcessRole_PROCESS_ROLE_MONGOD
}

// gauge reports a down service as unmeasured even if a reading survives in the query window,
// so a figure for a process that is not running is not read as current.
func gauge(value *float64, up bool) float64 {
	if !up || value == nil {
		return unmeasured
	}
	return *value
}

func oplogWindow(fields fieldSet) *float64 {
	head, tail := fields.f64(fieldOplogHead), fields.f64(fieldOplogTail)
	if head == nil || tail == nil || *head < *tail {
		return nil
	}
	window := *head - *tail
	return &window
}

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

func named(key string) *string {
	if key == unspecified {
		return nil
	}
	return &key
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// optionalDouble is optional's counterpart for a gauge that may not apply at all.
func optionalDouble(value *float64) *float64 {
	return value
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
