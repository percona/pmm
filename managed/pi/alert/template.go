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
	"io"

	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/pi/common"
)

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
		params = new(ParseParams)
	}

	d := yaml.NewDecoder(reader)
	d.KnownFields(params.DisallowUnknownFields)

	var res []Template
	for {
		var c templates
		if err := d.Decode(&c); err != nil {
			if errors.Is(err, io.EOF) {
				return res, nil
			}
			return nil, errors.Wrap(err, "failed to parse templates")
		}

		for _, template := range c.Templates {
			if err := template.Validate(); err != nil {
				if params.DisallowInvalidTemplates {
					return nil, errors.Wrapf(err, "failed to validate template '%s'", template.Name)
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
		return "", errors.Wrap(err, "failed to marshal templates to YAML")
	}

	return string(b), nil
}

// Template represents Integrated Alerting rule template.
type Template struct {
	Name        string              `yaml:"name"`                  // required
	Version     uint32              `yaml:"version"`               // required
	Summary     string              `yaml:"summary"`               // required
	Expr        string              `yaml:"expr"`                  // required
	Params      []Parameter         `yaml:"params,omitempty"`      // optional
	For         promconfig.Duration `yaml:"for"`                   // required
	Severity    common.Severity     `yaml:"severity"`              // required
	Labels      map[string]string   `yaml:"labels,omitempty"`      // optional
	Annotations map[string]string   `yaml:"annotations,omitempty"` // optional
}

// Validate validates template.
func (r *Template) Validate() error {
	var err error
	if r.Version != 1 {
		return errors.Errorf("unexpected version %d", r.Version)
	}

	if r.Name == "" {
		return errors.New("template name is empty")
	}

	if r.Summary == "" {
		return errors.New("template summary is empty")
	}

	if r.Expr == "" {
		return errors.New("template expression is empty")
	}

	if err = r.validateParams(); err != nil {
		return err
	}

	return r.Severity.Validate()
}

func (r *Template) validateParams() error {
	var err error
	for _, param := range r.Params {
		if err = param.Validate(); err != nil {
			return errors.Wrapf(err, "parameter '%s' is invalid", param.Name)
		}
	}

	return nil
}
