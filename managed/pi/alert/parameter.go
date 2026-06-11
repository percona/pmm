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

package alert

import (
	"errors"
	"fmt"
	"strconv"
)

// Parameter represents alerting template or rule parameter.
type Parameter struct {
	Name    string `yaml:"name"`           // required
	Summary string `yaml:"summary"`        // required
	Unit    Unit   `yaml:"unit,omitempty"` // optional
	Type    Type   `yaml:"type"`           // required
	Range   []any  `yaml:"range,flow,omitempty"`
	Value   any    `yaml:"value,omitempty"`
}

// GetValueForBool casts parameter value to the bool.
func (p *Parameter) GetValueForBool() (bool, error) {
	if p.Type != Bool {
		return false, fmt.Errorf("parameter type is %s, not bool", p.Type)
	}

	switch v := p.Value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, err
		}

		return b, nil
	default:
		// handle integers, etc
		f, err := castValueToFloat64(v)
		return f != 0, err
	}
}

// GetValueForFloat casts parameter value to the float64. Before invocation of this method you should check that
// value is present (not nil), as it's optional.
func (p *Parameter) GetValueForFloat() (float64, error) {
	if p.Type != Float {
		return 0, fmt.Errorf("parameter type is %s, not float", p.Type)
	}

	return castValueToFloat64(p.Value)
}

// GetRangeForFloat casts range parameters to the float64. Before invocation of this method you should check that
// range is present (slice is not empty), as it's optional.
func (p *Parameter) GetRangeForFloat() (float64, float64, error) {
	if p.Type != Float {
		return 0, 0, fmt.Errorf("parameter type is %s, not float", p.Type)
	}

	var (
		lower, higher float64
		err           error
	)

	lower, err = castValueToFloat64(p.Range[0])
	if err != nil {
		return 0, 0, err
	}

	higher, err = castValueToFloat64(p.Range[1])
	if err != nil {
		return 0, 0, err
	}

	return lower, higher, nil
}

// GetValueForString casts parameter value to the string.
func (p *Parameter) GetValueForString() (string, error) {
	if p.Type != String {
		return "", fmt.Errorf("parameter type is %s, not string", p.Type)
	}

	switch v := p.Value.(type) {
	case nil:
		return "", errors.New("value is nil")
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("value has unhandled type %T", v)
	}
}

// Validate validates parameter.
func (p *Parameter) Validate() error {
	var err error

	if p.Name == "" {
		return errors.New("parameter name is empty")
	}

	if p.Summary == "" {
		return errors.New("parameter summary is empty")
	}

	err = p.Unit.Validate()
	if err != nil {
		return err
	}

	err = p.Type.Validate()
	if err != nil {
		return err
	}

	err = p.validateValue()
	if err != nil {
		return err
	}

	return p.validateRange()
}

func (p *Parameter) validateValue() error {
	if p.Value == nil {
		return nil
	}

	switch p.Type {
	case Bool:
		_, err := p.GetValueForBool()
		return err
	case Float:
		_, err := p.GetValueForFloat()
		return err
	case String:
		_, err := p.GetValueForString()
		return err
	}

	// do not add `default:` to make exhaustive linter do its job

	return fmt.Errorf("unknown parameter type: %s", p.Type)
}

func (p *Parameter) validateRange() error {
	if p.Range == nil {
		return nil
	}

	switch p.Type {
	case Bool, String:
		if len(p.Range) != 0 {
			return fmt.Errorf("range should be empty, but it has %d elements", len(p.Range))
		}

		return nil

	case Float:
		if len(p.Range) != 2 { //nolint:mnd
			return fmt.Errorf("range should be empty or have two elements, but it has %d elements", len(p.Range))
		}

		_, err := castValueToFloat64(p.Range[0])
		if err != nil {
			return fmt.Errorf("invalid lower element of range: %w", err)
		}

		_, err = castValueToFloat64(p.Range[1])
		if err != nil {
			return errors.New("invalid higher element of range")
		}

		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return fmt.Errorf("unknown parameter type: %s", p.Type)
}

func castValueToFloat64(v any) (float64, error) {
	switch v := v.(type) {
	case nil:
		return 0, errors.New("value is nil")
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}

		return f, nil
	default:
		return 0, fmt.Errorf("value has unhandled type %T", v)
	}
}
