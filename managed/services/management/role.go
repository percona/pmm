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

package management

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
)

// ErrInvalidRoleData is returned when a row cannot be asserted to role.
var ErrInvalidRoleData = fmt.Errorf("InvalidRoleData")

// RoleService represents service for working with roles.
type RoleService struct {
	db *reform.DB

	managementpb.UnimplementedRoleServer
}

// NewRoleService creates a RoleService instance.
func NewRoleService(db *reform.DB) *RoleService {
	//nolint:exhaustruct
	return &RoleService{
		db: db,
	}
}

// CreateRole creates a new Role.
func (r *RoleService) CreateRole(_ context.Context, req *managementpb.RoleData) (*managementpb.RoleID, error) {
	var role models.Role
	role.Title = req.Title
	role.Filter = req.Filter

	if err := models.CreateRole(r.db.Querier, &role); err != nil {
		return nil, err
	}

	return &managementpb.RoleID{
		RoleId: role.ID,
	}, nil
}

// UpdateRole updates a Role.
//
//nolint:unparam
func (r *RoleService) UpdateRole(_ context.Context, req *managementpb.RoleData) (*managementpb.EmptyResponse, error) {
	err := r.db.InTransaction(func(tx *reform.TX) error {
		var role models.Role
		if err := tx.FindByPrimaryKeyTo(&role, req.RoleId); err != nil {
			if ok := errors.As(err, &reform.ErrNoRows); ok {
				return status.Errorf(codes.NotFound, "Role not found")
			}
			return err
		}

		role.Title = req.Title
		role.Filter = req.Filter

		if err := tx.Update(&role); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &managementpb.EmptyResponse{}, nil
}

// DeleteRole deletes a Role.
//
//nolint:unparam
func (r *RoleService) DeleteRole(_ context.Context, req *managementpb.RoleID) (*managementpb.EmptyResponse, error) {
	var role models.Role
	role.ID = req.RoleId

	if err := r.db.Querier.Delete(&role); err != nil {
		if ok := errors.As(err, &reform.ErrNoRows); ok {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}

		return nil, err
	}

	return &managementpb.EmptyResponse{}, nil
}

// GetRole retrieves a Role.
func (r *RoleService) GetRole(_ context.Context, req *managementpb.RoleID) (*managementpb.RoleData, error) {
	var role models.Role
	if err := r.db.Querier.FindByPrimaryKeyTo(&role, req.RoleId); err != nil {
		if ok := errors.As(err, &reform.ErrNoRows); ok {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}

		return nil, err
	}

	return roleToResponse(&role), nil
}

// ListRoles lists all Roles.
func (r *RoleService) ListRoles(_ context.Context, _ *managementpb.ListRolesRequest) (*managementpb.ListRolesResponse, error) {
	rows, err := r.db.Querier.SelectAllFrom(models.RoleTable, "")
	if err != nil {
		return nil, err
	}

	res := &managementpb.ListRolesResponse{
		Roles: make([]*managementpb.RoleData, 0, len(rows)),
	}

	for _, row := range rows {
		role, ok := row.(*models.Role)
		if !ok {
			return nil, fmt.Errorf("%w: invalid role data in table", ErrInvalidRoleData)
		}

		res.Roles = append(res.Roles, roleToResponse(role))
	}

	return res, nil
}

func roleToResponse(role *models.Role) *managementpb.RoleData {
	return &managementpb.RoleData{
		RoleId: role.ID,
		Title:  role.Title,
		Filter: role.Filter,
	}
}

// AssignRole assigns a Role to a user.
//
//nolint:unparam
func (r *RoleService) AssignRole(_ context.Context, req *managementpb.AssignRoleRequest) (*managementpb.EmptyResponse, error) {
	err := r.db.InTransaction(func(tx *reform.TX) error {
		return models.AssignRole(tx, int(req.UserId), int(req.RoleId))
	})
	if err != nil {
		if errors.Is(err, models.ErrRoleNotFound) {
			return nil, status.Errorf(codes.NotFound, "Role not found")
		}
		return nil, err
	}

	return &managementpb.EmptyResponse{}, nil
}
