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

package models

// thresholdScopeSpecificity ranks scopes most-specific-first, so a narrower override
// wins over a broader one covering the same target.
//
// Service ranks highest because it is the only relation that actually holds: a service
// runs on exactly one node and belongs to at most one cluster, so a service override is
// strictly narrower than either. Node over cluster is a convention rather than a
// containment - the two cross-cut, since a cluster spans several nodes while a node
// hosts services from several clusters - but one machine is the narrower intent.
//
// Precedence cannot be expressed in PromQL: reducing both sides of an `or` to a common
// label set is what makes `or` prefer the left operand, but that reduction is exactly
// what destroys the scope information needed to rank by. So precedence is resolved here,
// in Go, and there is no backstop in the query if this function is wrong.
var thresholdScopeSpecificity = map[ThresholdScope]int{
	ThresholdScopeService: 3,
	ThresholdScopeNode:    2,
	ThresholdScopeCluster: 1,
}

// ThresholdInventory maps override targets onto the join-label values an alert rule
// matches on. Targets missing from it no longer exist and are skipped.
type ThresholdInventory struct {
	// NodeNames maps node_id to node_name.
	NodeNames map[string]string
	// ServiceNames maps service_id to service_name.
	ServiceNames map[string]string
	// ServicesByCluster maps a cluster label value to the names of its services.
	ServicesByCluster map[string][]string
}

// targetNames returns the join-label values an override applies to. A node or service
// override yields at most one; a cluster override fans out onto every service in that
// cluster. An unresolvable target yields none, so a row left behind by a deleted entity
// is inert rather than wrong.
func (inv ThresholdInventory) targetNames(override *AlertRuleThresholdOverride) []string {
	switch override.Scope {
	case ThresholdScopeNode:
		name, ok := inv.NodeNames[override.Target]
		if !ok {
			return nil
		}

		return []string{name}

	case ThresholdScopeService:
		name, ok := inv.ServiceNames[override.Target]
		if !ok {
			return nil
		}

		return []string{name}

	case ThresholdScopeCluster:
		return inv.ServicesByCluster[override.Target]
	}

	// do not add `default:` to make exhaustive linter do its job

	return nil
}

// ResolveThresholds returns the effective threshold for every target covered by an
// override or a tombstone, keyed by the join-label value the rule matches on.
//
// This is the single implementation of precedence. Both the metrics collector and the
// API must call it: if they resolved separately and drifted, the value the API reports
// and the value the rule evaluates against would silently disagree.
//
// Tombstoned rows contribute no candidate for their own scope, so clearing a service
// override correctly falls through to a covering cluster override rather than jumping
// straight to the default. A tombstoned target with no surviving override at any scope
// resolves to defaultValue - which is what makes clearing an override a value change on
// an existing series rather than the series disappearing.
func ResolveThresholds(overrides []*AlertRuleThresholdOverride, defaultValue float64, inv ThresholdInventory) map[string]float64 {
	resolved := make(map[string]float64, len(overrides))
	specificity := make(map[string]int, len(overrides))

	var cleared []string

	for _, override := range overrides {
		names := inv.targetNames(override)
		if len(names) == 0 {
			continue
		}

		if override.IsCleared() {
			cleared = append(cleared, names...)
			continue
		}

		rank := thresholdScopeSpecificity[override.Scope]
		for _, name := range names {
			existing, ok := specificity[name]
			if ok && existing >= rank {
				continue
			}

			resolved[name] = override.Value
			specificity[name] = rank
		}
	}

	// A cleared target keeps its series alive at the rule's default, unless a coarser
	// override still applies to it.
	for _, name := range cleared {
		_, ok := resolved[name]
		if !ok {
			resolved[name] = defaultValue
		}
	}

	return resolved
}
