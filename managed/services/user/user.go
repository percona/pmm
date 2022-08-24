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

// Package user provides API for user related tasks
package user

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/userpb"
	"github.com/percona/pmm/managed/models"
)

// Service is responsible for user related APIs
type Service struct {
	db *reform.DB
	l  *logrus.Entry

	userpb.UnimplementedUserServer
}

// NewUserService return a user service
func NewUserService(db *reform.DB) *Service {
	l := logrus.WithField("component", "user")

	s := Service{
		db: db,
		l:  l,
	}
	return &s
}

// GetUser creates a new user
func (s *Service) GetUser(ctx context.Context, req *userpb.UserDetailsRequest) (*userpb.UserDetailsResponse, error) {
	// TODO : Get user ID from Grafana
	userID := 32

	userInfo := &models.UserDetails{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		userInfo, err = models.FindUser(tx.Querier, userID)
		if err != nil {
			// User entry missing; create entry
			params := &models.CreateUserParams{
				UserID: userID,
			}
			userInfo, err = models.CreateUser(tx.Querier, params)
			return err
		}

		return nil
	})

	if e != nil {
		return nil, e
	}

	resp := &userpb.UserDetailsResponse{
		UserId:               uint32(userInfo.ID),
		ProductTourCompleted: userInfo.Tour,
	}
	return resp, nil
}

// UpdateUser updates data for given user
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UserUpdateRequest) (*userpb.UserDetailsResponse, error) {
	if !req.ProductTourCompleted {
		return nil, status.Errorf(codes.InvalidArgument, "Tour flag cannot be unset")
	}

	// TODO : Get ID from Grafana
	userID := 32

	userInfo := &models.UserDetails{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		userInfo, err = models.FindUser(tx.Querier, userID)
		if err != nil {
			return err
		}

		return nil
	})

	if e != nil {
		return nil, e
	}

	e = s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		params := &models.UpdateUserParams{
			UserID: userInfo.ID,
			Tour:   req.ProductTourCompleted,
		}
		userInfo, err = models.UpdateUser(tx.Querier, params)
		if err != nil {
			return err
		}

		return nil
	})

	if e != nil {
		return nil, e
	}

	resp := &userpb.UserDetailsResponse{
		UserId:               uint32(userInfo.ID),
		ProductTourCompleted: userInfo.Tour,
	}
	return resp, nil
}
