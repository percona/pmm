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

// Service is responsible for user related APIs.
type Service struct {
	db *reform.DB
	l  *logrus.Entry
	c  grafanaClient

	userpb.UnimplementedUserServer
}

type grafanaClient interface {
	GetUserID(ctx context.Context) (int, error)
}

// NewUserService return a user service.
func NewUserService(db *reform.DB, client grafanaClient) *Service {
	l := logrus.WithField("component", "user")

	s := Service{
		db: db,
		l:  l,

		c: client,
	}
	return &s
}

// GetUser creates a new user.
func (s *Service) GetUser(ctx context.Context, req *userpb.UserDetailsRequest) (*userpb.UserDetailsResponse, error) {
	userID, err := s.c.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	userInfo, err := models.GetOrCreateUser(s.db.Querier, userID)
	if err != nil {
		return nil, err
	}

	resp := &userpb.UserDetailsResponse{
		UserId:               uint32(userInfo.ID),
		ProductTourCompleted: userInfo.Tour,
	}
	return resp, nil
}

// UpdateUser updates data for given user.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UserUpdateRequest) (*userpb.UserDetailsResponse, error) {
	if !req.ProductTourCompleted {
		return nil, status.Errorf(codes.InvalidArgument, "Tour flag cannot be unset")
	}

	userID, err := s.c.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	userInfo := &models.UserDetails{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		userInfo, err = models.FindUser(tx.Querier, userID)
		if err != nil {
			if err == models.ErrNotFound {
				return status.Errorf(codes.Unavailable, "User not found")
			}
			return err
		}

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
