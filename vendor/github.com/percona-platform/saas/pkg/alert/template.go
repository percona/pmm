// Package alert implements alert templates parsing and validation.
package alert

import (
	"io"

	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/percona-platform/saas/pkg/common"
)

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

	type templates struct {
		Templates []Template `yaml:"templates"`
	}

	var res []Template
	for {
		var c templates
		if err := d.Decode(&c); err != nil {
			if err == io.EOF {
				return res, nil
			}
			return nil, errors.Wrap(err, "failed to parse templates")
		}

		for _, template := range c.Templates {
			if err := template.Validate(); err != nil {
				if params.DisallowInvalidTemplates {
					return nil, err
				}

				continue // skip invalid template
			}

			res = append(res, template)
		}
	}
}

// Template represents Integrated Alerting rule template.
type Template struct {
	Name        string              `yaml:"name"`                  // required
	Version     uint32              `yaml:"version"`               // required
	Summary     string              `yaml:"summary"`               // required
	Tiers       []common.Tier       `yaml:"tiers,flow,omitempty"`  // optional
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

	if err = common.ValidateTiers(r.Tiers); err != nil {
		return err
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
			return err
		}
	}

	return nil
}
