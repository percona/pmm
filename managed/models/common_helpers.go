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

import (
	"fmt"
)

// InvalidArgumentError returned when some passed argument is invalid.
type InvalidArgumentError struct {
	Details string
}

// NewInvalidArgumentError creates InvalidArgumentError with given formatting.
func NewInvalidArgumentError(format string, a ...any) *InvalidArgumentError {
	return &InvalidArgumentError{Details: fmt.Sprintf(format, a...)}
}

func (e *InvalidArgumentError) Error() string {
	return "invalid argument: " + e.Details
}

// LocalhostAddr is the IPv4 loopback address used by PMM Server's co-located services.
const LocalhostAddr = "127.0.0.1"

// internalAddr reports whether host refers to PMM's built-in,
// co-located services.
func internalAddr(host string) bool {
	return host == LocalhostAddr || host == "localhost"
}
