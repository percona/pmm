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
	"fmt"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// Rule represents alert rule configuration.
//
//reform:ia_rules
type Rule struct {
	ID                string                     `reform:"id,pk"`
	Name              string                     `reform:"name"`
	Summary           string                     `reform:"summary"`
	TemplateName      string                     `reform:"template_name"`
	Disabled          bool                       `reform:"disabled"`
	ExprTemplate      string                     `reform:"expr_template"`
	ParamsDefinitions AlertExprParamsDefinitions `reform:"params_definitions"`
	ParamsValues      AlertExprParamsValues      `reform:"params_values"`
	DefaultFor        time.Duration              `reform:"default_for"`
	For               time.Duration              `reform:"for"`
	DefaultSeverity   Severity                   `reform:"default_severity"`
	Severity          Severity                   `reform:"severity"`
	CustomLabels      []byte                     `reform:"custom_labels"`
	Labels            []byte                     `reform:"labels"`
	Annotations       []byte                     `reform:"annotations"`
	Filters           Filters                    `reform:"filters"`
	ChannelIDs        ChannelIDs                 `reform:"channel_ids"`
	CreatedAt         time.Time                  `reform:"created_at"`
	UpdatedAt         time.Time                  `reform:"updated_at"`
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

// GetLabels decodes template labels.
func (r *Rule) GetLabels() (map[string]string, error) {
	return getLabels(r.Labels)
}

// SetLabels encodes template labels.
func (r *Rule) SetLabels(m map[string]string) error {
	return setLabels(m, &r.Labels)
}

// GetAnnotations decodes template annotations.
func (r *Rule) GetAnnotations() (map[string]string, error) {
	return getLabels(r.Annotations)
}

// SetAnnotations encodes template annotations.
func (r *Rule) SetAnnotations(m map[string]string) error {
	return setLabels(m, &r.Annotations)
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

// AlertExprParamsValues represents rule parameters values slice.
type AlertExprParamsValues []AlertExprParamValue

// Value implements database/sql/driver Valuer interface.
func (p AlertExprParamsValues) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql Scanner interface.
func (p *AlertExprParamsValues) Scan(src interface{}) error { return jsonScan(p, src) }

// AsStringMap convert param values to string map, where parameter name is a map key and parameter value is a map value.
func (p AlertExprParamsValues) AsStringMap() map[string]string {
	m := make(map[string]string, len(p))
	for _, rp := range p {
		var value string
		switch rp.Type {
		case Float:
			value = fmt.Sprint(rp.FloatValue)
		case Bool:
			value = fmt.Sprint(rp.BoolValue)
		case String:
			value = rp.StringValue
		}
		// do not add `default:` to make exhaustive linter do its job

		m[rp.Name] = value
	}

	return m
}

// AlertExprParamValue represents rule parameter value.
type AlertExprParamValue struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	BoolValue   bool      `json:"bool"`
	FloatValue  float64   `json:"float"`
	StringValue string    `json:"string"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p AlertExprParamValue) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *AlertExprParamValue) Scan(src interface{}) error { return jsonScan(p, src) }

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
