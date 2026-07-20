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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// FindAdvisorChecks returns all user-authored advisor checks ordered by name.
func FindAdvisorChecks(q *reform.Querier) ([]*AdvisorCheck, error) {
	rows, err := q.SelectAllFrom(AdvisorCheckTable, "ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to select advisor checks: %w", err)
	}

	checks := make([]*AdvisorCheck, 0, len(rows))
	for _, r := range rows {
		checks = append(checks, r.(*AdvisorCheck)) //nolint:forcetypeassert
	}
	return checks, nil
}

// FindAdvisorCheckByName finds a user-authored advisor check by name.
// It returns reform.ErrNoRows if the check does not exist.
func FindAdvisorCheckByName(q *reform.Querier, name string) (*AdvisorCheck, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty advisor check name.")
	}

	c := &AdvisorCheck{Name: name}
	err := q.Reload(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// CreateAdvisorCheck persists a new user-authored advisor check.
func CreateAdvisorCheck(q *reform.Querier, c *AdvisorCheck) (*AdvisorCheck, error) {
	err := q.Insert(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create advisor check: %w", err)
	}

	return c, nil
}

// UpdateAdvisorCheck updates an existing user-authored advisor check, preserving its creation time.
// It returns reform.ErrNoRows if the check does not exist.
func UpdateAdvisorCheck(q *reform.Querier, c *AdvisorCheck) (*AdvisorCheck, error) {
	existing := &AdvisorCheck{Name: c.Name}
	err := q.Reload(existing)
	if err != nil {
		return nil, err
	}

	c.CreatedAt = existing.CreatedAt
	err = q.Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update advisor check: %w", err)
	}

	return c, nil
}

// RemoveAdvisorCheck deletes a user-authored advisor check by name.
// It returns reform.ErrNoRows if the check does not exist.
func RemoveAdvisorCheck(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "Empty advisor check name.")
	}

	err := q.Delete(&AdvisorCheck{Name: name})
	if err != nil {
		return err
	}

	return nil
}
