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

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// AlertRuleThresholdOverride represents a per-node override of an overridable
// threshold parameter of an alert rule. It is keyed by the stable node_id; a
// row without an override means the node uses the rule's default value. The
// dynamic-thresholds collector reads these rows to emit pmm_alert_threshold.
//
//reform:alert_rule_threshold_overrides
type AlertRuleThresholdOverride struct {
	ID        string    `reform:"id,pk"`
	RuleID    string    `reform:"rule_id"`
	ParamName string    `reform:"param_name"`
	NodeID    string    `reform:"node_id"`
	Value     float64   `reform:"value"`
	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
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

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AlertRuleThresholdOverride)(nil)
	_ reform.BeforeUpdater  = (*AlertRuleThresholdOverride)(nil)
	_ reform.AfterFinder    = (*AlertRuleThresholdOverride)(nil)
)
