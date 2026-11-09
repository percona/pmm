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
//
// The iota order below is the precedence chain itself, narrowest last.
const (
	thresholdSpecificityCluster = iota + 1
	thresholdSpecificityNode
	thresholdSpecificityService
)

var thresholdScopeSpecificity = map[ThresholdScope]int{
	ThresholdScopeService: thresholdSpecificityService,
	ThresholdScopeNode:    thresholdSpecificityNode,
	ThresholdScopeCluster: thresholdSpecificityCluster,
}

// ThresholdInventory maps override targets onto the join-label values an alert rule
// matches on. Targets missing from it no longer exist and are skipped.
type ThresholdInventory struct {
	// NodeNames maps node_id to node_name.
	NodeNames map[string]string
	// ServiceNames maps service_id to service_name.
	ServiceNames map[string]string
}

// targetNames returns the join-label values an override applies to: at most one for a node
// or service, and none for a target that no longer exists, so a row it left behind is inert.
// A cluster override resolves to nothing until services can be looked up by cluster.
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
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return nil
}

// ResolvedThreshold is the effective threshold for one target, with the override it came
// from. Source is nil when the value is the rule's default.
type ResolvedThreshold struct {
	Value  float64
	Source *AlertRuleThresholdOverride
}

// IsOverridden reports whether the value comes from an override rather than the default.
func (r ResolvedThreshold) IsOverridden() bool {
	return r.Source != nil
}

// ResolveThresholds returns the effective threshold, and the override that won, for every
// target an override covers, keyed by the join-label value the rule matches on. Targets it
// omits use the rule's default.
//
// This is the single implementation of precedence. Both the metrics collector and the
// API must call it: if they resolved separately and drifted, the value the API reports
// and the value the rule evaluates against would silently disagree.
func ResolveThresholds(overrides []*AlertRuleThresholdOverride, inv ThresholdInventory) map[string]ResolvedThreshold {
	resolved := make(map[string]ResolvedThreshold, len(overrides))
	specificity := make(map[string]int, len(overrides))

	for _, override := range overrides {
		names := inv.targetNames(override)
		if len(names) == 0 {
			continue
		}

		rank := thresholdScopeSpecificity[override.Scope]
		for _, name := range names {
			existing, ok := specificity[name]
			if ok && existing >= rank {
				continue
			}

			resolved[name] = ResolvedThreshold{Value: override.Value, Source: override}
			specificity[name] = rank
		}
	}

	return resolved
}
