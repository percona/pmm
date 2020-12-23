package alert

import (
	"strconv"

	"github.com/pkg/errors"
)

// Parameter represents alerting template or rule parameter.
type Parameter struct {
	Name    string        `yaml:"name"`       // required
	Summary string        `yaml:"summary"`    // required
	Unit    string        `yaml:"unit"`       // required FIXME should be optional
	Type    Type          `yaml:"type"`       // required
	Range   []interface{} `yaml:"range,flow"` // required FIXME should be optional
	Value   interface{}   `yaml:"value"`      // required FIXME should be optional in template - then it is required in the rule
}

// GetValueForBool casts parameter value to the bool.
func (p *Parameter) GetValueForBool() (bool, error) {
	if p.Type != Bool {
		return false, errors.Errorf("parameter type is %s, not bool", p.Type)
	}

	switch v := p.Value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, errors.WithStack(err)
		}
		return b, nil
	default:
		// handle integers, etc
		f, err := castValueToFloat64(v)
		return f != 0, err
	}
}

// GetValueForFloat casts parameter value to the float64.
func (p *Parameter) GetValueForFloat() (float64, error) {
	if p.Type != Float {
		return 0, errors.Errorf("parameter type is %s, not float", p.Type)
	}

	return castValueToFloat64(p.Value)
}

// GetRangeForFloat casts range parameters to the float64.
func (p *Parameter) GetRangeForFloat() (float64, float64, error) {
	if p.Type != Float {
		return 0, 0, errors.Errorf("parameter type is %s, not float", p.Type)
	}

	var lower, higher float64
	var err error

	if lower, err = castValueToFloat64(p.Range[0]); err != nil {
		return 0, 0, err
	}
	if higher, err = castValueToFloat64(p.Range[1]); err != nil {
		return 0, 0, err
	}

	return lower, higher, nil
}

// GetValueForString casts parameter value to the string.
func (p *Parameter) GetValueForString() (string, error) {
	if p.Type != String {
		return "", errors.Errorf("parameter type is %s, not string", p.Type)
	}

	switch v := p.Value.(type) {
	case nil:
		return "", errors.New("value is nil")
	case string:
		return v, nil
	default:
		return "", errors.Errorf("value has unhandled type %T", v)
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

	if p.Unit == "" {
		return errors.New("parameter unit is empty")
	}

	if err = p.Type.Validate(); err != nil {
		return err
	}

	if err = p.validateValue(); err != nil {
		return err
	}

	if err = p.validateRange(); err != nil {
		return err
	}

	return nil
}

func (p *Parameter) validateValue() error {
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

	return errors.Errorf("unknown parameter type: %s", p.Type)
}

func (p *Parameter) validateRange() error {
	if p.Range == nil {
		return nil
	}

	switch p.Type {
	case Bool:
		if len(p.Range) != 0 {
			return errors.Errorf("range should have only zero elements, but has %d", len(p.Range))
		}
		return nil

	case Float:
		if len(p.Range) != 2 {
			return errors.Errorf("range should have only two elements, but has %d", len(p.Range))
		}
		if _, err := castValueToFloat64(p.Range[0]); err != nil {
			return errors.Wrapf(err, "invalid lower element of range")
		}
		if _, err := castValueToFloat64(p.Range[1]); err != nil {
			return errors.Errorf("invalid higher element of range")
		}
		return nil

	case String:
		if len(p.Range) != 0 {
			return errors.Errorf("range should have only zero elements, but has %d", len(p.Range))
		}
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return errors.Errorf("unknown parameter type: %s", p.Type)
}

func castValueToFloat64(v interface{}) (float64, error) {
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
			return 0, errors.WithStack(err)
		}
		return f, nil
	default:
		return 0, errors.Errorf("value has unhandled type %T", v)
	}
}
