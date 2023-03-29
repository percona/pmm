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

// Package roles manages roles and their assignments to various entities such
// as users, teams or organizations.
package roles

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// EntityType is a type of entity in the registry.
type EntityType int

const (
	// EntityInvalid represents zero value
	EntityInvalid EntityType = iota
	// EntityUser represents a user
	EntityUser
	// EntityTeam represents a team
	EntityTeam
)

// Registry holds a list of entity services and supports interactions on top of them.
type Registry struct {
	services map[EntityType]EntityService
}

// NewRegistry creates new Registry.
func NewRegistry(services map[EntityType]EntityService) *Registry {
	return &Registry{
		services: services,
	}
}

// AssignRoles assigns roles to the specified entity.
func (r *Registry) AssignRoles(tx *reform.TX, entityType EntityType, entityID int, roleIDs []int) error {
	srv, err := r.findService(entityType)
	if err != nil {
		return err
	}

	err = srv.AssignRoles(tx, entityID, roleIDs)
	return err
}

// AssignDefaultRole assigns a default role to a user.
func (r *Registry) AssignDefaultRole(tx *reform.TX, userID int) error {
	settings, err := models.GetSettings(tx)
	if err != nil {
		return err
	}

	if settings.DefaultRoleID <= 0 {
		logrus.Panicf("Default role ID is %d", settings.DefaultRoleID)
	}

	return r.AssignRoles(tx, EntityUser, userID, []int{settings.DefaultRoleID})
}

// BeforeDeleteRole shall be run before a role is deleted to prepare for role removal
// across all entities in the registry.
func (r *Registry) BeforeDeleteRole(tx *reform.TX, roleID, newRoleID int) error {
	for _, s := range r.services {
		if err := s.BeforeDeleteRole(tx, roleID, newRoleID); err != nil {
			return err
		}
	}

	return nil
}

// GetUserRoles retrieves all roles assigned to a user across all entities in the registry.
func (r *Registry) GetUserRoles(q *reform.Querier, userID int) ([]models.Role, error) {
	var roles []models.Role

	srv, ok := r.services[EntityUser]
	if ok {
		r, err := srv.GetEntityRoles(q, userID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r...)
	}

	return roles, nil
}

// findService returns a service based on the entity type in the registry, if found.
func (r *Registry) findService(entityType EntityType) (EntityService, error) {
	srv, ok := r.services[entityType]
	if !ok {
		return nil, errors.Errorf("Cannot find entity type %d in the registry", entityType)
	}

	return srv, nil
}
