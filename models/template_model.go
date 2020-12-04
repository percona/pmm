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

	"github.com/percona-platform/saas/pkg/common"
	"gopkg.in/reform.v1"
)

//go:generate reform

// Template represents Integrated Alerting rule template.
//reform:ia_templates
type Template struct {
	Name        string        `reform:"name,pk"`
	Version     uint32        `reform:"version"`
	Summary     string        `reform:"summary"`
	Tiers       Tiers         `reform:"tiers"`
	Expr        string        `reform:"expr"`
	Params      Params        `reform:"params"`
	For         time.Duration `reform:"for"`
	Severity    Severity      `reform:"severity"`
	Labels      []byte        `reform:"labels"`
	Annotations []byte        `reform:"annotations"`
	Source      Source        `reform:"source"`
	Yaml        string        `reform:"yaml"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (t *Template) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (t *Template) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (t *Template) AfterFind() error {
	t.CreatedAt = t.CreatedAt.UTC()
	t.UpdatedAt = t.UpdatedAt.UTC()

	return nil
}

// GetLabels decodes template labels.
func (t *Template) GetLabels() (map[string]string, error) {
	return getLabels(t.Labels)
}

// SetLabels encodes template labels.
func (t *Template) SetLabels(m map[string]string) error {
	return setLabels(m, &t.Labels)
}

// GetAnnotations decodes template annotations.
func (t *Template) GetAnnotations() (map[string]string, error) {
	return getLabels(t.Annotations)
}

// SetAnnotations encodes template annotations.
func (t *Template) SetAnnotations(m map[string]string) error {
	return setLabels(m, &t.Annotations)
}

// Tiers represents tiers slice.
type Tiers []common.Tier

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (t Tiers) Value() (driver.Value, error) { return jsonValue(t) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (t *Tiers) Scan(src interface{}) error { return jsonScan(t, src) }

// Params represent Param slice.
type Params []Param

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p Params) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *Params) Scan(src interface{}) error { return jsonScan(p, src) }

// ParamType represents parameter type.
type ParamType string

// Available parameter types.
const (
	Float  = ParamType("float")
	Bool   = ParamType("bool")
	String = ParamType("string")
)

// Param represents template parameter.
type Param struct {
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	Unit    string    `json:"unit"`
	Type    ParamType `json:"type"`

	FloatParam *FloatParam `json:"float_param"`
	// BoolParam   *BoolParam   `json:"bool_param"`
	// StringParam *StringParam `json:"string_param"`
}

// BoolParam represents boolean template parameter.
type BoolParam struct {
	Default bool `json:"default"`
}

// FloatParam represents float template parameter.
type FloatParam struct {
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

// StringParam represents string template parameter.
type StringParam struct {
	Default string `json:"default"`
}

// Severity represents alert severity.
type Severity string

// Available severity levels.
const (
	UnknownSeverity   = Severity("unknown")
	EmergencySeverity = Severity("emergency")
	AlertSeverity     = Severity("alert")
	CriticalSeverity  = Severity("critical")
	ErrorSeverity     = Severity("error")
	WarningSeverity   = Severity("warning")
	NoticeSeverity    = Severity("notice")
	InfoSeverity      = Severity("info")
	DebugSeverity     = Severity("debug")
)

// Source represents template source.
type Source string

// Available template sources.
const (
	UnknownSource  = Source("unknown")
	BuiltInSource  = Source("built_in")
	SAASSource     = Source("saas")
	UserFileSource = Source("user_file")
	UserAPISource  = Source("user_api")
)

// check interfaces
var (
	_ reform.BeforeInserter = (*Template)(nil)
	_ reform.BeforeUpdater  = (*Template)(nil)
	_ reform.AfterFinder    = (*Template)(nil)
)
