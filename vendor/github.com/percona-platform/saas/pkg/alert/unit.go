package alert

import "github.com/pkg/errors"

// Supported parameter units.
const (
	Percentage = Unit("%")
	Seconds    = Unit("s")
)

// Unit represent Integrated Alerting parameter unit.
type Unit string

// Validate returns error in case of invalid unit.
func (u Unit) Validate() error {
	switch u {
	case "": // can be empty
		return nil
	case Percentage:
		return nil
	case Seconds:
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return errors.Errorf("unhandled parameter unit %s", string(u))
}
