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

import "github.com/pkg/errors"

var (
	// ErrNotFound returned when entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists returned when an entity with the same value already exists and has unique constraint on the requested field.
	ErrAlreadyExists = errors.New("already exists")

	// ErrRoleNotFound is returned when a role is not found.
	ErrRoleNotFound = errors.New("role not found")
	// ErrRoleIsDefaultRole is returned when trying to delete a default role.
	ErrRoleIsDefaultRole = errors.New("role is a default role")
)
