// Package common contains common type definitions and functions.
package common

import "github.com/pkg/errors"

// Tier represents platform user tier.
type Tier string

// Supported check tiers.
const (
	Anonymous  = Tier("anonymous")
	Registered = Tier("registered")
	Paid       = Tier("paid")
)

// Validate validates tier value.
func (t Tier) Validate() error {
	switch t {
	case Anonymous:
	case Registered:
	case Paid:
	case "":
		return errors.New("tier is empty")
	default:
		return errors.Errorf("unknown check tier: %q", t)
	}

	return nil
}

// ValidateTiers validates tiers and checks them for duplicates.
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
