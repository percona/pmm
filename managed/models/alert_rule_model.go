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

// AlertRuleParam is the snapshot of one overridable parameter, taken when the rule was
// created. The template it came from can be edited or deleted afterwards, so this is the
// only durable record of what the rule actually evaluates against, and the only place the
// effective default can be read back from.
type AlertRuleParam struct {
	Default   float64  `json:"default"`
	JoinLabel string   `json:"join_label"`
	Scopes    []string `json:"scopes"`
	Unit      string   `json:"unit,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
}

// AlertRuleParams maps a parameter name to its snapshot.
type AlertRuleParams map[string]AlertRuleParam

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p AlertRuleParams) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *AlertRuleParams) Scan(src any) error { return jsonScan(p, src) }

// AlertRule represents a PMM-created Grafana alert rule that carries overridable
// thresholds. The row is a registry entry, not the rule itself: Grafana remains the
// authority for the rule definition, and PMM keeps only what Grafana cannot supply.
//
//reform:alert_rules
type AlertRule struct {
	RuleID string `reform:"rule_id,pk"`
	// GrafanaRuleUID is a cached handle for the rule in Grafana, never the identity.
	// It is nil until the rule has been created there.
	GrafanaRuleUID *string         `reform:"grafana_rule_uid"`
	Params         AlertRuleParams `reform:"params"`
	CreatedAt      time.Time       `reform:"created_at"`
	UpdatedAt      time.Time       `reform:"updated_at"`
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
