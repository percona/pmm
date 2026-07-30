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

// Package alert implements alert templates parsing and validation.
package alert

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/percona/promconfig"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/pi/common"
)

var tiersDeprecationWarned sync.Map

type templates struct {
	Templates []Template `yaml:"templates"`
}

// ParseParams represents optional Parse function parameters.
type ParseParams struct {
	DisallowUnknownFields    bool // if true, return errors for unexpected YAML fields
	DisallowInvalidTemplates bool // if true, return errors for invalid templates instead of skipping them
}

// Parse returns a slice of validated templates parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains templates form every parsed document.
func Parse(reader io.Reader, params *ParseParams) ([]Template, error) {
	if params == nil {
		params = &ParseParams{}
	}

	d := yaml.NewDecoder(reader)
	d.KnownFields(params.DisallowUnknownFields)

	var res []Template

	for {
		var c templates

		err := d.Decode(&c)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return res, nil
			}

			return nil, err
		}

		for _, template := range c.Templates {
			err := template.Validate()
			if err != nil {
				if params.DisallowInvalidTemplates {
					return nil, err
				}

				continue // skip invalid template
			}

			res = append(res, template)
		}
	}
}

// ToYAML returns YAML representation of given templates.
func ToYAML(ts []Template) (string, error) {
	b, err := yaml.Marshal(&templates{Templates: ts})
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Template represents Integrated Alerting rule template.
type Template struct {
	Name        string               `yaml:"name"`                  // required
	Version     uint32               `yaml:"version"`               // required
	Summary     string               `yaml:"summary"`               // required
	Expr        string               `yaml:"expr,omitempty"`        // required for single-expression templates
	Queries     []TemplateQuery      `yaml:"queries,omitempty"`     // optional PromQL query steps
	Expressions []TemplateExpression `yaml:"expressions,omitempty"` // optional Grafana expression steps
	Condition   string               `yaml:"condition,omitempty"`   // required for multi-expression templates
	Params      []Parameter          `yaml:"params,omitempty"`      // optional
	For         promconfig.Duration  `yaml:"for"`                   // required
	Severity    common.Severity      `yaml:"severity"`              // required
	Labels      map[string]string    `yaml:"labels,omitempty"`      // optional
	Annotations map[string]string    `yaml:"annotations,omitempty"` // optional
	// TODO: Tiers field is deprecated and must be removed in PMM v4.
	Tiers []string `yaml:"tiers,omitempty"` // optional
}

// Validate validates template.
func (r *Template) Validate() error {
	var err error

	if r.Version != 1 {
		return fmt.Errorf("unexpected version %d", r.Version)
	}

	if r.Name == "" {
		return errors.New("template name is empty")
	}

	if r.Summary == "" {
		return errors.New("template summary is empty")
	}

	err = r.validateSteps()
	if err != nil {
		return err
	}

	// Log deprecation warning for tiers field (once per template name)
	if len(r.Tiers) != 0 {
		if _, warned := tiersDeprecationWarned.LoadOrStore(r.Name, true); !warned {
			logrus.WithFields(logrus.Fields{
				"component": "alert/template",
				"template":  r.Name,
				"tiers":     r.Tiers,
			}).Warn("The 'tiers' field in alert templates is deprecated and will be removed in PMM v4. Please update your templates.")
		}
	}

	err = r.validateParams()
	if err != nil {
		return err
	}

	err = r.validateOverridableParams()
	if err != nil {
		return err
	}

	return r.Severity.Validate()
}

func (r *Template) validateParams() error {
	var err error
	for _, param := range r.Params {
		err = param.Validate()
		if err != nil {
			return fmt.Errorf("parameter '%s' is invalid: %w", param.Name, err)
		}
	}

	return nil
}

// validateOverridableParams ensures every parameter marked `overridable: true`
// is a float threshold referenced as a `[[ .name ]]` token in a shape the
// dynamic-thresholds rule builder can rewrite: inside an expression step for
// multi-expression templates, or as the RHS of the final comparison for
// single-expression templates — see managed/services/alerting/rule_builder.go.
func (r *Template) validateOverridableParams() error {
	if !r.UsesMultipleExpressions() {
		overridableCount := 0
		for _, param := range r.Params {
			if param.Overridable {
				overridableCount++
			}
		}
		if overridableCount > 1 {
			return fmt.Errorf("single-expression templates support at most one overridable parameter, got %d", overridableCount)
		}
	}

	for _, param := range r.Params {
		if !param.Overridable {
			continue
		}

		if param.Type != Float {
			return fmt.Errorf("parameter '%s' is overridable but not a float", param.Name)
		}

		if r.UsesMultipleExpressions() {
			if !r.paramReferencedInExpressions(param.Name) {
				return fmt.Errorf("overridable parameter '%s' is not referenced as a [[ .%s ]] token in any expression step", param.Name, param.Name)
			}
			continue
		}

		if _, err := FindSingleExprThresholdRef(r.Expr, param.Name); err != nil {
			return err
		}
	}

	return nil
}

// paramReferencedInExpressions reports whether `[[ .name ]]` (allowing arbitrary
// surrounding whitespace) appears in any expression step's body.
func (r *Template) paramReferencedInExpressions(name string) bool {
	re := paramTokenRegexp(name)
	for _, expression := range r.Expressions {
		if re.MatchString(expression.Expression) {
			return true
		}
	}

	return false
}
