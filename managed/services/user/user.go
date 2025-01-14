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

// Package user provides API for user related tasks.
package user

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	userv1 "github.com/percona/pmm/api/user/v1"
	"github.com/percona/pmm/managed/models"
)

// Service is responsible for user related APIs.
type Service struct {
	db *reform.DB
	l  *logrus.Entry
	c  grafanaClient

	userv1.UnimplementedUserServiceServer
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
func (s *Service) GetUser(ctx context.Context, _ *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	userID, err := s.c.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	userInfo, err := models.GetOrCreateUser(s.db.Querier, userID)
	if err != nil {
		return nil, err
	}

	resp := &userv1.GetUserResponse{
		UserId:                  uint32(userInfo.ID),
		ProductTourCompleted:    userInfo.Tour,
		AlertingTourCompleted:   userInfo.AlertingTour,
		SnoozedPmmVersion:       userInfo.SnoozedPMMVersion,
		SnoozedApiKeysMigration: userInfo.SnoozedAPIKeysMigration,
	}
	return resp, nil
}

// UpdateUser updates data for given user.
func (s *Service) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	userID, err := s.c.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	userInfo := &models.UserDetails{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		userInfo, err = models.FindUser(tx.Querier, userID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				return status.Errorf(codes.Unavailable, "User not found")
			}
			return err
		}

		params := &models.UpdateUserParams{
			UserID:                  userInfo.ID,
			Tour:                    req.ProductTourCompleted,
			AlertingTour:            req.AlertingTourCompleted,
			SnoozedAPIKeysMigration: req.SnoozedApiKeysMigration,
		}
		if req.SnoozedPmmVersion != nil {
			params.SnoozedPMMVersion = req.SnoozedPmmVersion
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

	resp := &userv1.UpdateUserResponse{
		UserId:                  uint32(userInfo.ID),
		ProductTourCompleted:    userInfo.Tour,
		AlertingTourCompleted:   userInfo.AlertingTour,
		SnoozedPmmVersion:       userInfo.SnoozedPMMVersion,
		SnoozedApiKeysMigration: userInfo.SnoozedAPIKeysMigration,
	}
	return resp, nil
}

// ListUsers lists all users and their details.
func (s *Service) ListUsers(_ context.Context, _ *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	userRoles, err := models.ListUsers(s.db.Querier)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListUsersResponse{
		Users: make([]*userv1.ListUsersResponse_UserDetail, 0, len(userRoles)),
	}
	for userID, roleIDs := range userRoles {
		resp.Users = append(resp.Users, &userv1.ListUsersResponse_UserDetail{
			UserId:  uint32(userID),
			RoleIds: roleIDs,
		})
	}

	return resp, nil
}
