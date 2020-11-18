// Package common contains common type definitions and functions.
package common

import "github.com/pkg/errors"

// Tier represents platform user tier.
type Tier string

// Supported check tiers.
const (
	Anonymous  = Tier("anonymous")
	Registered = Tier("registered")
)

// Validate validates tier value.
func (t Tier) Validate() error {
	switch t {
	case Anonymous:
	case Registered:
	default:
		return errors.Errorf("unknown check tier: %q", t)
	}

	return nil
}

// ValidateTiers validates tiers.
func ValidateTiers(tiers []Tier) error {
	m := make(map[Tier]struct{}, len(tiers))
	for _, tier := range tiers {
		if err := tier.Validate(); err != nil {
			return err
		}

		if _, ok := m[tier]; ok {
			return errors.Errorf("duplicate tier: %q", tier)
		}
		m[tier] = struct{}{}
	}

	return nil
}
