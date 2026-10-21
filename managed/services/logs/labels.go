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

package logs

import (
	"cmp"
	"context"
	"slices"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

// labelsQuery reads every service with a cluster, environment or replication set, with its node's name.
const labelsQuery = "SELECT s.cluster, s.environment, s.replication_set, n.node_name FROM services s " +
	"JOIN nodes n ON n.node_id = s.node_id WHERE s.cluster != '' OR s.environment != '' OR s.replication_set != ''"

// serviceLabels are a service's labels and the node it runs on; log lines carry the node, not the service.
type serviceLabels struct {
	cluster, environment, replicationSet, nodeName string
}

// labeledServices returns the services with a cluster, environment or replication set.
func (s *Service) labeledServices(ctx context.Context) ([]serviceLabels, error) {
	rows, err := s.inv.WithContext(ctx).Query(labelsQuery)
	if err != nil {
		return nil, s.inventoryError(err)
	}
	defer rows.Close() //nolint:errcheck
	var res []serviceLabels
	for rows.Next() {
		var l serviceLabels
		err = rows.Scan(&l.cluster, &l.environment, &l.replicationSet, &l.nodeName)
		if err != nil {
			return nil, s.inventoryError(err)
		}
		res = append(res, l)
	}
	err = rows.Err()
	if err != nil {
		return nil, s.inventoryError(err)
	}
	return res, nil
}

// labelNodes returns, sorted, the nodes running services with one of the values of each label given.
func labelNodes(services []serviceLabels, clusters, environments, replicationSets []string) []string {
	matches := func(values []string, v string) bool {
		return len(values) == 0 || slices.Contains(values, v)
	}
	var nodes []string
	for _, s := range services {
		if matches(clusters, s.cluster) && matches(environments, s.environment) && matches(replicationSets, s.replicationSet) {
			nodes = append(nodes, s.nodeName)
		}
	}
	slices.Sort(nodes)
	return slices.Compact(nodes)
}

// labelFacets returns each label value with the lines of the nodes running its services, most first; values without
// lines are left out.
func labelFacets(services []serviceLabels, nodeLines map[string]int64) ([]*logsv1.Facet, []*logsv1.Facet, []*logsv1.Facet) {
	facets := func(label func(serviceLabels) string) []*logsv1.Facet {
		nodes := make(map[string]map[string]struct{})
		for _, s := range services {
			v := label(s)
			if v == "" {
				continue
			}
			if nodes[v] == nil {
				nodes[v] = make(map[string]struct{})
			}
			nodes[v][s.nodeName] = struct{}{}
		}
		var res []*logsv1.Facet
		for v, set := range nodes {
			var n int64
			for node := range set {
				n += nodeLines[node]
			}
			if n > 0 {
				res = append(res, &logsv1.Facet{Value: v, Count: n})
			}
		}
		slices.SortFunc(res, func(a, b *logsv1.Facet) int {
			return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Value, b.Value))
		})
		return res
	}
	return facets(func(s serviceLabels) string { return s.cluster }),
		facets(func(s serviceLabels) string { return s.environment }),
		facets(func(s serviceLabels) string { return s.replicationSet })
}
