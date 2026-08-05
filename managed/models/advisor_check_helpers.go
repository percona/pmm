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
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// FindAdvisorChecks returns all advisor checks ordered by name.
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

// FindAdvisorCheckByName finds an advisor check by name.
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

// UpdateAdvisorCheck updates the content of an existing user-authored advisor check,
// preserving its creation time, source and settings (interval override, disabled state,
// per-service disables). It returns reform.ErrNoRows if the check does not exist.
func UpdateAdvisorCheck(q *reform.Querier, c *AdvisorCheck) (*AdvisorCheck, error) {
	existing := &AdvisorCheck{Name: c.Name}
	err := q.Reload(existing)
	if err != nil {
		return nil, err
	}

	c.CreatedAt = existing.CreatedAt
	c.Source = existing.Source
	c.IntervalOverride = existing.IntervalOverride
	c.Disabled = existing.Disabled
	c.DisabledServiceIDs = existing.DisabledServiceIDs
	err = q.Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update advisor check: %w", err)
	}

	return c, nil
}

// UpsertAdvisorCheckContent inserts a built-in advisor check or refreshes its
// content columns if a row with the same name already exists. Settings columns
// (interval_override, disabled, disabled_service_ids) are never touched on update,
// so user-set overrides survive content refreshes across restarts.
func UpsertAdvisorCheckContent(ctx context.Context, q *reform.Querier, c *AdvisorCheck) error {
	now := Now()
	_, err := q.ExecContext(ctx, `
		INSERT INTO advisor_checks (
			name, source, version, summary, description, category, subcategory,
			technology, interval, interval_override, disabled, disabled_service_ids,
			queries, script, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NULL, false, NULL, $10, $11, $12, $12
		)
		ON CONFLICT (name) DO UPDATE SET
			source = EXCLUDED.source,
			version = EXCLUDED.version,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			subcategory = EXCLUDED.subcategory,
			technology = EXCLUDED.technology,
			interval = EXCLUDED.interval,
			queries = EXCLUDED.queries,
			script = EXCLUDED.script,
			updated_at = EXCLUDED.updated_at`,
		c.Name, BuiltinCheckSource, c.Version, c.Summary, c.Description, c.Category, c.Subcategory,
		c.Technology, c.Interval, c.Queries, c.Script, now)
	if err != nil {
		return fmt.Errorf("failed to upsert advisor check %s: %w", c.Name, err)
	}

	return nil
}

// RemoveAdvisorChecksNotIn deletes built-in advisor checks whose names are not
// in the given list. User-authored checks are never touched. It is used to prune rows
// for checks removed from the checks package.
func RemoveAdvisorChecksNotIn(ctx context.Context, q *reform.Querier, names []string) error {
	if len(names) == 0 {
		// An empty list means the checks failed to load; do not wipe the table.
		return nil
	}

	args := make([]any, 0, len(names)+1)
	args = append(args, BuiltinCheckSource)
	placeholders := make([]string, 0, len(names))
	for i, n := range names {
		args = append(args, n)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2)) //nolint:mnd
	}

	_, err := q.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM advisor_checks WHERE source = $1 AND name NOT IN (%s)", strings.Join(placeholders, ", ")),
		args...)
	if err != nil {
		return fmt.Errorf("failed to prune advisor checks: %w", err)
	}

	return nil
}

// FindDisabledAdvisorCheckNames returns the names of globally-disabled advisor checks.
func FindDisabledAdvisorCheckNames(ctx context.Context, q *reform.Querier) ([]string, error) {
	rows, err := q.WithContext(ctx).SelectAllFrom(AdvisorCheckTable, "WHERE disabled ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to select disabled advisor checks: %w", err)
	}

	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.(*AdvisorCheck).Name) //nolint:forcetypeassert
	}
	return names, nil
}

// SetAdvisorChecksDisabled sets the global disabled flag for the named advisor checks.
// Per-service disable settings are intentionally left untouched: they still apply
// once a check is re-enabled globally.
func SetAdvisorChecksDisabled(ctx context.Context, q *reform.Querier, names []string, disabled bool) error {
	if len(names) == 0 {
		return nil
	}

	args := make([]any, 0, len(names)+1)
	args = append(args, disabled)
	placeholders := make([]string, 0, len(names))
	for i, n := range names {
		args = append(args, n)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2)) //nolint:mnd
	}

	_, err := q.ExecContext(ctx,
		fmt.Sprintf("UPDATE advisor_checks SET disabled = $1, updated_at = now() WHERE name IN (%s)", strings.Join(placeholders, ", ")),
		args...)
	if err != nil {
		return fmt.Errorf("failed to change disabled state of advisor checks: %w", err)
	}

	return nil
}

// ChangeAdvisorCheckInterval sets the user interval override for the named advisor check.
// It returns reform.ErrNoRows if the check does not exist.
func ChangeAdvisorCheckInterval(ctx context.Context, q *reform.Querier, name string, interval Interval) (*AdvisorCheck, error) {
	c, err := FindAdvisorCheckByName(q.WithContext(ctx), name)
	if err != nil {
		return nil, err
	}

	override := string(interval)
	c.IntervalOverride = &override
	err = q.WithContext(ctx).Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to change interval of advisor check %s: %w", name, err)
	}

	return c, nil
}

// ChangeAdvisorCheckDisabledServices replaces the set of service IDs for which the
// named advisor check is disabled. It returns reform.ErrNoRows if the check does not exist.
func ChangeAdvisorCheckDisabledServices(ctx context.Context, q *reform.Querier, name string, serviceIDs []string) (*AdvisorCheck, error) {
	c, err := FindAdvisorCheckByName(q.WithContext(ctx), name)
	if err != nil {
		return nil, err
	}

	err = c.SetDisabledServiceIDs(serviceIDs)
	if err != nil {
		return nil, err
	}

	err = q.WithContext(ctx).Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to change disabled services of advisor check %s: %w", name, err)
	}

	return c, nil
}

// FindAdvisorCheckDisabledServices returns a map of check name to the service IDs
// for which that check is disabled. Checks with no per-service disables are omitted.
func FindAdvisorCheckDisabledServices(ctx context.Context, q *reform.Querier) (map[string][]string, error) {
	rows, err := q.WithContext(ctx).SelectAllFrom(AdvisorCheckTable, "WHERE disabled_service_ids IS NOT NULL ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to select advisor checks with disabled services: %w", err)
	}

	res := make(map[string][]string, len(rows))
	for _, r := range rows {
		c := r.(*AdvisorCheck) //nolint:forcetypeassert
		ids, err := c.GetDisabledServiceIDs()
		if err != nil {
			return nil, err
		}
		if len(ids) != 0 {
			res[c.Name] = ids
		}
	}
	return res, nil
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
