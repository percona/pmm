// Package alert implements alert rules parsing and validation.
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
	DisallowUnknownFields bool // if true, return errors for unexpected YAML fields
	DisallowInvalidRules  bool // if true, return errors for invalid rules instead of skipping them
}

// Parse returns a slice of validated rules parsed from YAML passed via a reader.
// It can handle multi-document YAMLs: parsing result will be a single slice
// that contains rules form every parsed document.
func Parse(reader io.Reader, params *ParseParams) ([]Rule, error) {
	if params == nil {
		params = new(ParseParams)
	}

	d := yaml.NewDecoder(reader)
	d.KnownFields(params.DisallowUnknownFields)

	type rules struct {
		Rules []Rule `yaml:"rules"`
	}

	var res []Rule
	for {
		var c rules
		if err := d.Decode(&c); err != nil {
			if err == io.EOF {
				return res, nil
			}
			return nil, errors.Wrap(err, "failed to parse rules")
		}

		for _, rule := range c.Rules {
			if err := rule.Validate(); err != nil {
				if params.DisallowInvalidRules {
					return nil, err
				}

				continue // skip invalid rule
			}

			res = append(res, rule)
		}
	}
}

// Rule represents alert manager alerting rule.
type Rule struct {
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

// Validate validates rule.
func (r *Rule) Validate() error {
	var err error
	if r.Version != 1 {
		return errors.Errorf("unexpected version %d", r.Version)
	}

	if r.Name == "" {
		return errors.New("rule name is empty")
	}

	if r.Summary == "" {
		return errors.New("rule summary is empty")
	}

	if err = common.ValidateTiers(r.Tiers); err != nil {
		return err
	}

	if r.Expr == "" {
		return errors.New("rule expression is empty")
	}

	if err = r.validateParams(); err != nil {
		return err
	}

	if err = r.Severity.Validate(); err != nil {
		return err
	}

	return nil
}

func (r *Rule) validateParams() error {
	var err error
	for _, param := range r.Params {
		if err = param.Validate(); err != nil {
			return err
		}
	}

	return nil
}
