// Copyright (C) 2022 Percona LLC
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
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// CreateUserParams has parameters to create a new user.
type CreateUserParams struct {
	UserID int
}

// UpdateUserParams has parameters to update existing user.
type UpdateUserParams struct {
	UserID       int
	Tour         bool
	AlertingTour bool
}

// CreateUser create a new user with given parameters.
func CreateUser(q *reform.Querier, params *CreateUserParams) (*UserDetails, error) {
	if params.UserID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID")
	}

	// Check user ID is unique
	row := &UserDetails{ID: params.UserID}
	err := q.Reload(row)
	switch err {
	case nil:
		return nil, status.Errorf(codes.AlreadyExists, "User with ID %d already exists", params.UserID)
	case reform.ErrNoRows:
		break
	default:
		return nil, errors.WithStack(err)
	}

	// Add user entry
	row = &UserDetails{ID: params.UserID}
	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	return row, nil
}

// UpdateUser updates an existing user with given parameters.
func UpdateUser(q *reform.Querier, params *UpdateUserParams) (*UserDetails, error) {
	if params.UserID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID")
	}

	// Find existing entry for user
	row, err := FindUser(q, params.UserID)
	if err != nil {
		return nil, err
	}

	row.Tour = params.Tour
	row.AlertingTour = params.AlertingTour

	if err = q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update user")
	}

	return row, nil
}

// FindUser find user details using given user ID.
func FindUser(q *reform.Querier, userID int) (*UserDetails, error) {
	if userID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID")
	}

	row := &UserDetails{ID: userID}
	switch err := q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, ErrNotFound
	default:
		return nil, errors.WithStack(err)
	}
}
