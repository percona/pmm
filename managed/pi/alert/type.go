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

	return errors.Errorf("unhandled parameter type '%s'", t)
}
