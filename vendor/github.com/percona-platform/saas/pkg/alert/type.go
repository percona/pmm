package alert

import "github.com/pkg/errors"

// Supported parameter types.
const (
	Bool   = Type("bool")
	Float  = Type("float")
	String = Type("string")
)

// Type represent Integrated Alerting parameter type.
type Type string

// Validate returns error in case of invalid type value.
func (t Type) Validate() error {
	switch t {
	case Bool:
		return nil
	case Float:
		return nil
	case String:
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return errors.Errorf("unhandled parameter type %s", string(t))
}
