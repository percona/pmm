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
	UserID                  int
	Tour                    *bool
	AlertingTour            *bool
	SnoozedPMMVersion       *string
	SnoozedAPIKeysMigration *bool
}

// GetOrCreateUser returns user and optionally creates it, if not in database yet.
func GetOrCreateUser(q *reform.Querier, userID int) (*UserDetails, error) {
	userInfo, err := FindUser(q, userID)
	if errors.Is(err, ErrNotFound) {
		// User entry missing; create entry
		params := CreateUserParams{
			UserID: userID,
		}
		userInfo, err = CreateUser(q, &params)

		// Handling race-condition
		if errors.Is(err, ErrUserAlreadyExists) {
			userInfo, err = FindUser(q, userID)
		}
	}

	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// ErrUserAlreadyExists is returned when a user already exists in db.
var ErrUserAlreadyExists = fmt.Errorf("UserAlreadyExists")

// CreateUser create a new user with given parameters.
func CreateUser(q *reform.Querier, params *CreateUserParams) (*UserDetails, error) {
	if params.UserID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID")
	}

	// Check user ID is unique
	row := &UserDetails{ID: params.UserID}
	err := q.Reload(row)
	switch {
	case err == nil:
		return nil, ErrUserAlreadyExists
	case errors.Is(err, reform.ErrNoRows):
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
	if params.Tour != nil {
		row.Tour = *params.Tour
	}
	if params.AlertingTour != nil {
		row.AlertingTour = *params.AlertingTour
	}
	if params.SnoozedPMMVersion != nil {
		row.SnoozedPMMVersion = *params.SnoozedPMMVersion
	}
	if params.SnoozedAPIKeysMigration != nil {
		row.SnoozedAPIKeysMigration = *params.SnoozedAPIKeysMigration
	}

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
	err := q.Reload(row)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, errors.WithStack(err)
	}

	return row, nil
}
