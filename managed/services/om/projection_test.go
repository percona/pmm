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
	"maps"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

var (
	projectionNow    = time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	projectionMaxAge = 30 * time.Second
)

func mongoService(id, name, cluster, replicationSet, environment string) *models.Service {
	return &models.Service{
		ServiceID:      id,
		ServiceName:    name,
		ServiceType:    models.MongoDBServiceType,
		Cluster:        cluster,
		ReplicationSet: replicationSet,
		Environment:    environment,
		NodeID:         "node-" + id,
	}
}

// liveFacts is a healthy replica-set member as the metrics source would report it.
func liveFacts(state string, extra map[string]MergedField) map[string]MergedField {
	observed := projectionNow.Add(-5 * time.Second)
	fields := map[string]MergedField{
		fieldExporterUp:      {Value: 1.0, Source: sourceMetrics, ObservedAt: &observed},
		fieldState:           {Value: state, Source: sourceMetrics, ObservedAt: &observed},
		fieldVersion:         {Value: "7.0.39-21", Source: sourceMetrics, ObservedAt: &observed},
		fieldVendor:          {Value: "Percona", Source: sourceMetrics, ObservedAt: &observed},
		fieldEdition:         {Value: "Community", Source: sourceMetrics, ObservedAt: &observed},
		fieldHost:            {Value: "host-a", Source: sourceMetrics, ObservedAt: &observed},
		fieldEndpoint:        {Value: "host-a:27017", Source: sourceMetrics, ObservedAt: &observed},
		fieldCPUUsage:        {Value: 12.5, Source: sourceMetrics, ObservedAt: &observed},
		fieldConnectionsFree: {Value: 99.5, Source: sourceMetrics, ObservedAt: &observed},
	}
	maps.Copy(fields, extra)
	return fields
}

func onlyService(t *testing.T, doc document) *omv1.Service {
	t.Helper()
	require.Len(t, doc.environments, 1)
	require.Len(t, doc.environments[0].Clusters, 1)
	require.Len(t, doc.environments[0].Clusters[0].Services, 1)
	return doc.environments[0].Clusters[0].Services[0]
}

func TestBuildDocument(t *testing.T) {
	t.Parallel()

	t.Run("a service no source saw is present and DOWN", func(t *testing.T) {
		t.Parallel()

		// Dropping it would shrink the estate every time something broke.
		services := []*models.Service{mongoService("s1", "mongo-1", "c1", "rs0", "prod")}
		doc := buildDocument(services, map[string]map[string]MergedField{}, projectionNow, projectionMaxAge)

		svc := onlyService(t, doc)
		assert.Equal(t, "mongo-1", svc.ServiceName)
		assert.Equal(t, statusDown, svc.Status)
		assert.InDelta(t, float64(unmeasured), svc.CpuUsagePercent, 0.001)
		assert.InDelta(t, float64(unmeasured), svc.ConnectionsFreePercent, 0.001)
		assert.Nil(t, svc.Version)
		assert.Equal(t, int32(1), doc.summary.ServicesTotal)
		assert.Equal(t, int32(1), doc.summary.ServicesDown)
	})

	t.Run("a stopped service reads DOWN once its up flag ages out", func(t *testing.T) {
		t.Parallel()

		// The regression that mattered: the sample survives the long lookback window, so
		// only the age can tell that it is not a fact about now.
		stale := projectionNow.Add(-10 * time.Minute)
		fields := liveFacts("PRIMARY", nil)
		for _, key := range []string{fieldExporterUp, fieldState, fieldCPUUsage, fieldConnectionsFree} {
			held := fields[key]
			held.ObservedAt = &stale
			fields[key] = held
		}

		services := []*models.Service{mongoService("s1", "mongo-1", "c1", "rs0", "prod")}
		doc := buildDocument(services, map[string]map[string]MergedField{"s1": fields}, projectionNow, projectionMaxAge)

		svc := onlyService(t, doc)
		assert.Equal(t, statusDown, svc.Status)
		assert.Nil(t, svc.State, "a replica-set state from ten minutes ago is not current")
		assert.InDelta(t, float64(unmeasured), svc.CpuUsagePercent, 0.001)
		assert.Equal(t, "7.0.39-21", svc.Version.GetValue(), "identity is still worth reporting")
		assert.Equal(t, 1, doc.staleServices)
	})

	t.Run("gauges are suppressed for a service that is not up", func(t *testing.T) {
		t.Parallel()

		fields := liveFacts("PRIMARY", nil)
		delete(fields, fieldExporterUp)

		services := []*models.Service{mongoService("s1", "mongo-1", "c1", "rs0", "prod")}
		doc := buildDocument(services, map[string]map[string]MergedField{"s1": fields}, projectionNow, projectionMaxAge)

		svc := onlyService(t, doc)
		assert.Equal(t, statusDown, svc.Status)
		assert.InDelta(t, float64(unmeasured), svc.CpuUsagePercent, 0.001, "a CPU figure for a stopped process reads as current")
	})

	t.Run("the oplog window is the difference, and nil when it cannot be one", func(t *testing.T) {
		t.Parallel()

		observed := projectionNow.Add(-5 * time.Second)
		withOplog := func(head, tail float64) map[string]MergedField {
			return liveFacts("PRIMARY", map[string]MergedField{
				fieldOplogHead: {Value: head, Source: sourceMetrics, ObservedAt: &observed},
				fieldOplogTail: {Value: tail, Source: sourceMetrics, ObservedAt: &observed},
			})
		}
		services := []*models.Service{mongoService("s1", "mongo-1", "c1", "rs0", "prod")}

		doc := buildDocument(services, map[string]map[string]MergedField{"s1": withOplog(1000, 400)}, projectionNow, projectionMaxAge)
		assert.InDelta(t, 600.0, onlyService(t, doc).OplogWindowSeconds.GetValue(), 0.001)

		// head < tail is mismatched scrapes or clock skew: unknown, not a negative duration.
		doc = buildDocument(services, map[string]map[string]MergedField{"s1": withOplog(400, 1000)}, projectionNow, projectionMaxAge)
		assert.Nil(t, onlyService(t, doc).OplogWindowSeconds)

		// A standalone emits neither.
		doc = buildDocument(services, map[string]map[string]MergedField{"s1": liveFacts("PRIMARY", nil)}, projectionNow, projectionMaxAge)
		assert.Nil(t, onlyService(t, doc).OplogWindowSeconds)
	})

	t.Run("the process role comes from the exporter, not a port convention", func(t *testing.T) {
		t.Parallel()

		observed := projectionNow.Add(-5 * time.Second)
		services := []*models.Service{mongoService("s1", "mongo-1", "c1", "rs0", "prod")}

		for _, tc := range []struct {
			name  string
			extra map[string]MergedField
			want  string
		}{
			{"plain mongod", nil, processRoleMongod},
			{"router", map[string]MergedField{
				fieldIsMongos: {Value: true, Source: sourceMetrics, ObservedAt: &observed},
			}, processRoleMongos},
			{"config server", map[string]MergedField{
				fieldClusterRole: {Value: "configsvr", Source: sourceMetrics, ObservedAt: &observed},
			}, processRoleConfigsvr},
			{"shard server", map[string]MergedField{
				fieldClusterRole: {Value: "shardsvr", Source: sourceMetrics, ObservedAt: &observed},
			}, processRoleShardsvr},
		} {
			doc := buildDocument(services,
				map[string]map[string]MergedField{"s1": liveFacts("PRIMARY", tc.extra)},
				projectionNow, projectionMaxAge)
			assert.Equal(t, tc.want, onlyService(t, doc).ProcessRole, tc.name)
		}
	})

	t.Run("grouping falls back replica set, then to null", func(t *testing.T) {
		t.Parallel()

		observed := projectionNow.Add(-5 * time.Second)
		inv := func(cluster, rs, env string) map[string]MergedField {
			fields := map[string]MergedField{}
			for key, value := range map[string]string{
				fieldCluster: cluster, fieldReplicationSet: rs, fieldEnvironment: env,
			} {
				if value != "" {
					fields[key] = MergedField{Value: value, Source: sourceInventory, ObservedAt: &observed}
				}
			}
			return fields
		}

		services := []*models.Service{
			mongoService("s1", "a", "", "", ""),
			mongoService("s2", "b", "", "rs0", "prod"),
			mongoService("s3", "c", "cl1", "rs0", "prod"),
		}
		doc := buildDocument(services, map[string]map[string]MergedField{
			"s1": inv("", "", ""),
			"s2": inv("", "rs0", "prod"),
			"s3": inv("cl1", "rs0", "prod"),
		}, projectionNow, projectionMaxAge)

		// Environments sort by the internal grouping key, so the unnamed bucket --
		// "UNSPECIFIED" -- sorts ahead of "prod" by codepoint. Matches SEP, whose
		// sorted() orders the same way.
		require.Len(t, doc.environments, 2)
		assert.Nil(t, doc.environments[0].EnvName, "an unset environment is null, never invented")
		assert.Equal(t, "prod", doc.environments[1].EnvName.GetValue())

		require.Len(t, doc.environments[0].Clusters, 1)
		assert.Nil(t, doc.environments[0].Clusters[0].Name)

		require.Len(t, doc.environments[1].Clusters, 2)
		assert.Equal(t, "cl1", doc.environments[1].Clusters[0].Name.GetValue())
		assert.Equal(t, "rs0", doc.environments[1].Clusters[1].Name.GetValue(), "cluster falls back to the replica set")
	})

	t.Run("services sort by name inside a cluster", func(t *testing.T) {
		t.Parallel()

		services := []*models.Service{
			mongoService("s1", "mongo-c", "cl1", "rs0", "prod"),
			mongoService("s2", "mongo-a", "cl1", "rs0", "prod"),
			mongoService("s3", "mongo-b", "cl1", "rs0", "prod"),
		}
		merged := map[string]map[string]MergedField{}
		for _, service := range services {
			merged[service.ServiceID] = liveFacts("SECONDARY", nil)
		}
		doc := buildDocument(services, merged, projectionNow, projectionMaxAge)

		names := make([]string, 0, len(doc.environments[0].Clusters[0].Services))
		for _, svc := range doc.environments[0].Clusters[0].Services {
			names = append(names, svc.ServiceName)
		}
		assert.Equal(t, []string{"mongo-a", "mongo-b", "mongo-c"}, names)
	})

	t.Run("the summary counts what the tree holds", func(t *testing.T) {
		t.Parallel()

		observed := projectionNow.Add(-5 * time.Second)
		services := []*models.Service{
			mongoService("s1", "mongo-1", "cl1", "rs0", "prod"),
			mongoService("s2", "mongo-2", "cl1", "rs0", "prod"),
			mongoService("s3", "router", "cl1", "", "prod"),
		}
		doc := buildDocument(services, map[string]map[string]MergedField{
			"s1": liveFacts("PRIMARY", nil),
			"s2": {}, // never seen
			"s3": liveFacts("", map[string]MergedField{
				fieldIsMongos: {Value: true, Source: sourceMetrics, ObservedAt: &observed},
			}),
		}, projectionNow, projectionMaxAge)

		assert.Equal(t, int32(3), doc.summary.ServicesTotal)
		assert.Equal(t, int32(2), doc.summary.ServicesUp)
		assert.Equal(t, int32(1), doc.summary.ServicesDown)
		assert.Equal(t, map[string]int32{processRoleMongod: 2, processRoleMongos: 1}, doc.summary.ByProcessRole)
	})

	t.Run("observed_at is the newest observation behind the document", func(t *testing.T) {
		t.Parallel()

		newest := projectionNow.Add(-3 * time.Second)
		older := projectionNow.Add(-20 * time.Second)
		services := []*models.Service{
			mongoService("s1", "mongo-1", "cl1", "rs0", "prod"),
			mongoService("s2", "mongo-2", "cl1", "rs0", "prod"),
		}
		doc := buildDocument(services, map[string]map[string]MergedField{
			"s1": {fieldExporterUp: {Value: 1.0, Source: sourceMetrics, ObservedAt: &older}},
			"s2": {fieldExporterUp: {Value: 1.0, Source: sourceMetrics, ObservedAt: &newest}},
		}, projectionNow, projectionMaxAge)

		assert.Equal(t, newest, doc.observedAt)
		assert.False(t, isStale(doc, projectionNow))
	})
}

func TestInventoryEndpoint(t *testing.T) {
	t.Parallel()

	service := mongoService("s1", "mongo-1", "cl1", "rs0", "prod")
	service.Port = pointer.ToUint16(27017)
	service.Address = new("127.0.0.1")

	t.Run("prefers the node over the service address", func(t *testing.T) {
		t.Parallel()

		// A service address is where its own agent reaches it -- 127.0.0.1 for a local
		// agent, which is true and useless to anyone else.
		node := &models.Node{NodeName: "db-host-1", Address: "10.0.0.11"}
		assert.Equal(t, "10.0.0.11:27017", inventoryEndpoint(service, node))
	})

	t.Run("falls back to the node name, then the service address", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "db-host-1:27017", inventoryEndpoint(service, &models.Node{NodeName: "db-host-1"}))
		assert.Equal(t, "127.0.0.1:27017", inventoryEndpoint(service, nil))
	})

	t.Run("no port means no endpoint", func(t *testing.T) {
		t.Parallel()

		portless := mongoService("s2", "mongo-2", "", "", "")
		assert.Empty(t, inventoryEndpoint(portless, &models.Node{NodeName: "db-host-1"}))
	})
}
