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

package models

import "fmt"

// InvalidArgumentError returned when some passed argument is invalid.
type InvalidArgumentError struct {
	Details string
}

func (e *InvalidArgumentError) Error() string {
	return "invalid argument: " + e.Details
}

// NewInvalidArgumentError creates InvalidArgumentError with given formatting.
func NewInvalidArgumentError(format string, a ...interface{}) *InvalidArgumentError {
	return &InvalidArgumentError{Details: fmt.Sprintf(format, a...)}
}
