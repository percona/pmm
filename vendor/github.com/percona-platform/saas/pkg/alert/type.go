package alert

import "github.com/pkg/errors"

// Supported parameter types.
const (
	Float = Type("float")
)

// Type represent Integrated Alerting parameter type.
type Type string

// Validate returns error in case of invalid type value.
func (t Type) Validate() error {
	switch t {
	case Float:
		return nil
	case "":
		return errors.New("parameter type is empty")
	default:
		return errors.Errorf("unknown parameter type: %s", t)
	}
}
