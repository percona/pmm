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
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// AlertRuleDefaultParams maps an overridable parameter name to the default
// threshold value chosen at rule creation. It is the value used for any node
// that has no per-node override.
type AlertRuleDefaultParams map[string]float64

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p AlertRuleDefaultParams) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *AlertRuleDefaultParams) Scan(src any) error { return jsonScan(p, src) }

// AlertRule represents an alerting rule created from a template. PMM stores this
// registry row (in addition to the rule living in Grafana) so the
// dynamic-thresholds feature can emit per-node threshold metrics and serve the
// node-centric override API. The rule_id is a PMM-minted UUID that is also
// stamped on the Grafana rule as the `pmm_rule_id` label.
//
//reform:alert_rules
type AlertRule struct {
	RuleID        string                 `reform:"rule_id,pk"`
	TemplateName  string                 `reform:"template_name"`
	FolderUID     string                 `reform:"folder_uid"`
	RuleGroup     string                 `reform:"rule_group"`
	RuleTitle     string                 `reform:"rule_title"`
	DefaultParams AlertRuleDefaultParams `reform:"default_params"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *AlertRule) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *AlertRule) BeforeUpdate() error {
	r.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *AlertRule) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AlertRule)(nil)
	_ reform.BeforeUpdater  = (*AlertRule)(nil)
	_ reform.AfterFinder    = (*AlertRule)(nil)
)
