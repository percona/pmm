// pmm-managed
// Copyright (C) 2017 Percona LLC
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

// Package validators contains settings validators.
package validators

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateMySQLConnectionOptions validates MySQL connection options.
func ValidateMySQLConnectionOptions(socket, host *string, port *uint16) error {
	if host == nil && socket == nil {
		return status.Error(codes.InvalidArgument, "address or socket is required")
	}

	if host != nil {
		if socket != nil {
			return status.Error(codes.InvalidArgument, "setting both address and socket in once is disallowed")
		}

		if port == nil {
			return status.Errorf(codes.InvalidArgument, "invalid field Port: value '%d' must be greater than '0'", port)
		}
	}

	if socket != nil && port != nil {
		return status.Error(codes.InvalidArgument, "port is only allowed with address")
	}
	return nil
}
