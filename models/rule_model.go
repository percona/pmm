// pmm-managed
// Copyright (C) 2017 Percona LLC
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

//go:generate reform

// Rule represents alert rule configuration.
//reform:ia_rules
type Rule struct {
	TemplateName string        `reform:"template_name"`
	ID           string        `reform:"id,pk"`
	Summary      string        `reform:"summary"`
	Disabled     bool          `reform:"disabled"`
	Params       RuleParams    `reform:"params"`
	For          time.Duration `reform:"for"`
	Severity     Severity      `reform:"severity"`
	CustomLabels []byte        `reform:"custom_labels"`
	Filters      Filters       `reform:"filters"`
	ChannelIDs   ChannelIDs    `reform:"channel_ids"`
	CreatedAt    time.Time     `reform:"created_at"`
	UpdatedAt    time.Time     `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *Rule) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *Rule) BeforeUpdate() error {
	r.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *Rule) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()

	return nil
}

// GetCustomLabels decodes template labels.
func (r *Rule) GetCustomLabels() (map[string]string, error) {
	return getLabels(r.CustomLabels)
}

// SetCustomLabels encodes template labels.
func (r *Rule) SetCustomLabels(m map[string]string) error {
	return setLabels(m, &r.CustomLabels)
}

// FilterType represents rule filter type.
type FilterType string

// Available filter types.
const (
	Equal = FilterType("=")
	Regex = FilterType("=~")
)

// Filters represents filters slice.
type Filters []Filter

// Value implements database/sql/driver Valuer interface.
func (t Filters) Value() (driver.Value, error) { return jsonValue(t) }

// Scan implements database/sql Scanner interface.
func (t *Filters) Scan(src interface{}) error { return jsonScan(t, src) }

// Filter represents rule filter.
type Filter struct {
	Type FilterType `json:"type"`
	Key  string     `json:"key"`
	Val  string     `json:"value"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (f Filter) Value() (driver.Value, error) { return jsonValue(f) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (f *Filter) Scan(src interface{}) error { return jsonScan(f, src) }

// RuleParams represents rule parameters slice.
type RuleParams []RuleParam

// Value implements database/sql/driver Valuer interface.
func (t RuleParams) Value() (driver.Value, error) { return jsonValue(t) }

// Scan implements database/sql Scanner interface.
func (t *RuleParams) Scan(src interface{}) error { return jsonScan(t, src) }

// RuleParam represents rule parameter.
type RuleParam struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	BoolValue   bool      `json:"bool"`
	FloatValue  float32   `json:"float"`
	StringValue string    `json:"string"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p RuleParam) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *RuleParam) Scan(src interface{}) error { return jsonScan(p, src) }

// ChannelIDs is a slice of notification channel ids.
type ChannelIDs []string

// Value implements database/sql/driver Valuer interface.
func (t ChannelIDs) Value() (driver.Value, error) { return jsonValue(t) }

// Scan implements database/sql Scanner interface.
func (t *ChannelIDs) Scan(src interface{}) error { return jsonScan(t, src) }

// check interfaces.
var (
	_ reform.BeforeInserter = (*Rule)(nil)
	_ reform.BeforeUpdater  = (*Rule)(nil)
	_ reform.AfterFinder    = (*Rule)(nil)
)
