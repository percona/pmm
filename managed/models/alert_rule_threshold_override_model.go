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

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// ThresholdScope says what an override's target refers to.
type ThresholdScope string

// Threshold override scopes. Declaration order carries no meaning: precedence lives in
// thresholdScopeSpecificity, which ranks service above node above cluster.
const (
	ThresholdScopeNode    = ThresholdScope("node")
	ThresholdScopeService = ThresholdScope("service")
	ThresholdScopeCluster = ThresholdScope("cluster")
)

// Validate validates the threshold override scope.
//
// This returns a gRPC status error rather than an InvalidArgumentError, matching
// template_helpers.go in the same feature area, so that every validation failure the
// threshold helpers can produce surfaces as the same code without the service layer
// having to convert two different error shapes.
func (s ThresholdScope) Validate() error {
	switch s {
	case ThresholdScopeNode:
	case ThresholdScopeService:
	case ThresholdScopeCluster:
	default:
		return status.Errorf(codes.InvalidArgument, "Invalid threshold scope %q.", string(s))
	}

	return nil
}

// AlertRuleThresholdOverride is a per-target threshold for one parameter of one alert
// rule. Clearing an override tombstones the row rather than deleting it: a deleted row
// stops being emitted, and a series that stops being emitted keeps resolving for the
// whole of VictoriaMetrics' lookbehind, so the clear would take minutes to take effect
// instead of one scrape.
//
//reform:alert_rule_threshold_overrides
type AlertRuleThresholdOverride struct {
	ID        string         `reform:"id,pk"`
	RuleID    string         `reform:"rule_id"`
	ParamName string         `reform:"param_name"`
	Scope     ThresholdScope `reform:"scope"`
	// Target is a node_id, a service_id, or a cluster label value, depending on Scope.
	Target string  `reform:"target"`
	Value  float64 `reform:"value"`
	// ClearedAt marks the row as a tombstone. The stale Value is kept for audit and
	// must never be emitted: a cleared override resolves through the remaining scopes,
	// falling back to the rule's default only when none apply.
	ClearedAt *time.Time `reform:"cleared_at"`
	CreatedAt time.Time  `reform:"created_at"`
	UpdatedAt time.Time  `reform:"updated_at"`
}

// IsCleared reports whether the override has been cleared and is therefore a tombstone.
func (o *AlertRuleThresholdOverride) IsCleared() bool {
	return o.ClearedAt != nil
}

// BeforeInsert implements reform.BeforeInserter interface.
func (o *AlertRuleThresholdOverride) BeforeInsert() error {
	now := Now()
	o.CreatedAt = now
	o.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (o *AlertRuleThresholdOverride) BeforeUpdate() error {
	o.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (o *AlertRuleThresholdOverride) AfterFind() error {
	o.CreatedAt = o.CreatedAt.UTC()
	o.UpdatedAt = o.UpdatedAt.UTC()
	if o.ClearedAt != nil {
		cleared := o.ClearedAt.UTC()
		o.ClearedAt = &cleared
	}

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AlertRuleThresholdOverride)(nil)
	_ reform.BeforeUpdater  = (*AlertRuleThresholdOverride)(nil)
	_ reform.AfterFinder    = (*AlertRuleThresholdOverride)(nil)
)
