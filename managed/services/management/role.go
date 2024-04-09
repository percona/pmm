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

package management

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	rolev1beta1 "github.com/percona/pmm/api/role/v1beta1"
	"github.com/percona/pmm/managed/models"
)

// ErrInvalidRoleData is returned when a row cannot be asserted to role.
var ErrInvalidRoleData = errors.New("InvalidRoleData")

// RoleService represents service for working with roles.
type RoleService struct {
	db *reform.DB

	rolev1beta1.UnimplementedRoleServiceServer
}

// NewRoleService creates a RoleService instance.
func NewRoleService(db *reform.DB) *RoleService {
	//nolint:exhaustruct
	return &RoleService{
		db: db,
	}
}

// Enabled returns if service is enabled and can be used.
func (r *RoleService) Enabled() bool {
	settings, err := models.GetSettings(r.db)
	if err != nil {
		logrus.WithError(err).Error("cannot get settings")
		return false
	}
	return settings.IsAccessControlEnabled()
}

// CreateRole creates a new Role.
func (r *RoleService) CreateRole(_ context.Context, req *rolev1beta1.CreateRoleRequest) (*rolev1beta1.CreateRoleResponse, error) {
	role := models.Role{
		Title:       req.Title,
		Description: req.Description,
		Filter:      req.Filter,
	}

	if err := models.CreateRole(r.db.Querier, &role); err != nil {
		return nil, err
	}

	return &rolev1beta1.CreateRoleResponse{
		RoleId: role.ID,
	}, nil
}

// UpdateRole updates a Role.
//
//nolint:unparam
func (r *RoleService) UpdateRole(_ context.Context, req *rolev1beta1.UpdateRoleRequest) (*rolev1beta1.UpdateRoleResponse, error) {
	var role models.Role
	if err := r.db.FindByPrimaryKeyTo(&role, req.RoleId); err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}
		return nil, err
	}

	if req.Title != nil {
		role.Title = *req.Title
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Filter != nil {
		role.Filter = *req.Filter
	}

	if err := r.db.Update(&role); err != nil {
		return nil, err
	}

	return &rolev1beta1.UpdateRoleResponse{}, nil
}

// DeleteRole deletes a Role.
//
//nolint:unparam
func (r *RoleService) DeleteRole(ctx context.Context, req *rolev1beta1.DeleteRoleRequest) (*rolev1beta1.DeleteRoleResponse, error) {
	errTx := r.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if err := models.DeleteRole(tx, int(req.RoleId), int(req.ReplacementRoleId)); err != nil {
			if errors.Is(err, models.ErrRoleNotFound) {
				return status.Errorf(codes.NotFound, "Role not found")
			}

			return err
		}

		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	return &rolev1beta1.DeleteRoleResponse{}, nil
}

// GetRole retrieves a Role.
func (r *RoleService) GetRole(_ context.Context, req *rolev1beta1.GetRoleRequest) (*rolev1beta1.GetRoleResponse, error) {
	var role models.Role
	if err := r.db.Querier.FindByPrimaryKeyTo(&role, req.RoleId); err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}

		return nil, err
	}

	return &rolev1beta1.GetRoleResponse{
		RoleId:      role.ID,
		Title:       role.Title,
		Description: role.Description,
		Filter:      role.Filter,
	}, nil
}

// ListRoles lists all Roles.
func (r *RoleService) ListRoles(_ context.Context, _ *rolev1beta1.ListRolesRequest) (*rolev1beta1.ListRolesResponse, error) {
	rows, err := r.db.Querier.SelectAllFrom(models.RoleTable, "")
	if err != nil {
		return nil, err
	}

	res := &rolev1beta1.ListRolesResponse{
		Roles: make([]*rolev1beta1.ListRolesResponse_RoleData, 0, len(rows)), //nolint:nosnakecase
	}

	for _, row := range rows {
		role, ok := row.(*models.Role)
		if !ok {
			return nil, fmt.Errorf("%w: invalid role data in table", ErrInvalidRoleData)
		}

		//nolint:nosnakecase
		res.Roles = append(res.Roles, &rolev1beta1.ListRolesResponse_RoleData{
			RoleId:      role.ID,
			Title:       role.Title,
			Description: role.Description,
			Filter:      role.Filter,
		})
	}

	return res, nil
}

// AssignRoles assigns a Role to a user.
//
//nolint:unparam
func (r *RoleService) AssignRoles(ctx context.Context, req *rolev1beta1.AssignRolesRequest) (*rolev1beta1.AssignRolesResponse, error) {
	err := r.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		roleIDs := make([]int, 0, len(req.RoleIds))
		for _, id := range req.RoleIds {
			roleIDs = append(roleIDs, int(id))
		}
		return models.AssignRoles(tx, int(req.UserId), roleIDs)
	})
	if err != nil {
		if errors.Is(err, models.ErrRoleNotFound) {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}
		return nil, err
	}

	return &rolev1beta1.AssignRolesResponse{}, nil
}

// SetDefaultRole configures default role to be assigned to users.
//
//nolint:unparam
func (r *RoleService) SetDefaultRole(ctx context.Context, req *rolev1beta1.SetDefaultRoleRequest) (*rolev1beta1.SetDefaultRoleResponse, error) {
	err := r.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return models.ChangeDefaultRole(tx, int(req.RoleId))
	})
	if err != nil {
		return nil, err
	}

	return &rolev1beta1.SetDefaultRoleResponse{}, nil
}
