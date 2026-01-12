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

package check

import (
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Advisor represents group of checks with the common idea.
type Advisor struct {
	Version     uint32  `yaml:"version"`
	Name        string  `yaml:"name"`
	Summary     string  `yaml:"summary"`
	Description string  `yaml:"description"`
	Category    string  `yaml:"category"`
	Checks      []Check `yaml:"checks"`
}

// ParseAdvisors returns a slice of validated advisors parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains advisors from every parsed document.
func ParseAdvisors(reader io.Reader, params *ParseParams) ([]Advisor, error) {
	if params == nil {
		params = new(ParseParams)
	}

	d := yaml.NewDecoder(reader)
	d.KnownFields(params.DisallowUnknownFields)

	type advisors struct {
		Advisors []Advisor `yaml:"advisors"`
	}

	var res []Advisor
	for {
		var c advisors
		if err := d.Decode(&c); err != nil { //nolint:musttag
			if errors.Is(err, io.EOF) {
				return res, nil
			}
			return nil, errors.Wrap(err, "failed to parse advisors")
		}

		for _, advisor := range c.Advisors {
			if err := advisor.Validate(); err != nil {
				if params.DisallowInvalidChecks {
					return nil, err
				}

				continue // skip invalid advisors
			}

			res = append(res, advisor)
		}
	}
}

// Validate advisor.
func (a *Advisor) Validate() error { //nolint:cyclop
	if a.Version != 1 {
		return errors.Errorf("unexpected version %d", a.Version)
	}

	if !nameRE.MatchString(a.Name) {
		return errors.New("invalid advisor name")
	}

	if a.Summary == "" {
		return errors.New("summary is empty")
	}

	if a.Description == "" {
		return errors.New("description is empty")
	}

	if a.Category == "" {
		return errors.New("category is empty")
	}

	checkNames := make(map[string]struct{}, len(a.Checks))
	for _, check := range a.Checks {
		if err := check.Validate(); err != nil {
			return err
		}

		if check.Advisor != a.Name {
			return errors.Errorf("advisor name '%s' doesn't match name '%s' specified in corresponding check '%s'",
				a.Name, check.Advisor, check.Name,
			)
		}

		if _, ok := checkNames[check.Name]; ok {
			return errors.Errorf("check name collision `%s` detected in '%s' advisor", check.Name, a.Name)
		}
		checkNames[check.Name] = struct{}{}
	}

	return nil
}
