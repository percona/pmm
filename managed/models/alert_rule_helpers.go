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
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkThresholdOverrideKey(ruleID, paramName string, scope ThresholdScope, target string) error {
	if ruleID == "" {
		return status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	if paramName == "" {
		return status.Error(codes.InvalidArgument, "Empty parameter name.")
	}

	err := scope.Validate()
	if err != nil {
		return err
	}

	if target == "" {
		return status.Error(codes.InvalidArgument, "Empty target.")
	}

	return nil
}

// FindAlertRules returns all alert rules registered by PMM.
func FindAlertRules(q *reform.Querier) ([]*AlertRule, error) {
	structs, err := q.SelectAllFrom(AlertRuleTable, "")
	if err != nil {
		return nil, fmt.Errorf("failed to select alert rules: %w", err)
	}

	rules := make([]*AlertRule, len(structs))
	for i, s := range structs {
		rules[i] = s.(*AlertRule) //nolint:forcetypeassert
	}

	return rules, nil
}

// FindAlertRuleByID returns an alert rule by its PMM-minted ID.
func FindAlertRuleByID(q *reform.Querier, ruleID string) (*AlertRule, error) {
	if ruleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	rule := &AlertRule{RuleID: ruleID}
	err := q.Reload(rule)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Alert rule with ID '%s' not found.", ruleID)
		}

		return nil, err
	}

	return rule, nil
}

// FindAllThresholdOverrides returns every threshold override row.
func FindAllThresholdOverrides(q *reform.Querier) ([]*AlertRuleThresholdOverride, error) {
	return selectThresholdOverrides(q, "")
}

// FindThresholdOverridesByRule returns every override row for one rule.
func FindThresholdOverridesByRule(q *reform.Querier, ruleID string) ([]*AlertRuleThresholdOverride, error) {
	if ruleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	return selectThresholdOverrides(q, whereAllEqual(q, "rule_id"), ruleID)
}

// FindThresholdOverridesByTarget returns every override row for one target.
func FindThresholdOverridesByTarget(q *reform.Querier, scope ThresholdScope, target string) ([]*AlertRuleThresholdOverride, error) {
	err := scope.Validate()
	if err != nil {
		return nil, err
	}

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty target.")
	}

	tail := whereAllEqual(q, "scope", "target")

	return selectThresholdOverrides(q, tail, string(scope), target)
}

// whereAllEqual builds a WHERE clause matching every named column, numbering the
// placeholders in column order. Callers pass their arguments in that same order, so
// the numbering cannot drift out of step with them the way hand-written placeholders can.
func whereAllEqual(q *reform.Querier, columns ...string) string {
	conditions := make([]string, len(columns))
	for i, column := range columns {
		conditions[i] = column + " = " + q.Placeholder(i+1)
	}

	return "WHERE " + strings.Join(conditions, " AND ")
}

func selectThresholdOverrides(q *reform.Querier, tail string, args ...any) ([]*AlertRuleThresholdOverride, error) {
	structs, err := q.SelectAllFrom(AlertRuleThresholdOverrideTable, tail, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select threshold overrides: %w", err)
	}

	overrides := make([]*AlertRuleThresholdOverride, len(structs))
	for i, s := range structs {
		overrides[i] = s.(*AlertRuleThresholdOverride) //nolint:forcetypeassert
	}

	return overrides, nil
}

// CreateAlertRuleParams are params for creating a new alert rule registry row.
type CreateAlertRuleParams struct {
	RuleID string
	Params AlertRuleParams
}

// CreateAlertRule registers an alert rule created by PMM.
func CreateAlertRule(q *reform.Querier, params *CreateAlertRuleParams) (*AlertRule, error) {
	if params.RuleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	rule := &AlertRule{
		RuleID: params.RuleID,
		Params: params.Params,
	}
	if rule.Params == nil {
		rule.Params = AlertRuleParams{}
	}

	err := q.Insert(rule)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}

	return rule, nil
}

// UpsertThresholdOverride sets the override for one parameter of one rule at one target,
// creating the row if it does not exist.
func UpsertThresholdOverride(
	q *reform.Querier,
	ruleID, paramName string,
	scope ThresholdScope,
	target string,
	value float64,
) (*AlertRuleThresholdOverride, error) {
	err := checkThresholdOverrideKey(ruleID, paramName, scope, target)
	if err != nil {
		return nil, err
	}

	override := &AlertRuleThresholdOverride{
		ID:        uuid.New().String(),
		RuleID:    ruleID,
		ParamName: paramName,
		Scope:     scope,
		Target:    target,
		Value:     value,
	}

	err = override.BeforeInsert()
	if err != nil {
		return nil, err
	}

	columns := AlertRuleThresholdOverrideTable.Columns()
	query := fmt.Sprintf(
		`
		INSERT INTO %s (%s)
		VALUES (%s)
		ON CONFLICT (rule_id, param_name, scope, target) DO UPDATE
			SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
		RETURNING %s`,
		AlertRuleThresholdOverrideTable.Name(),
		strings.Join(columns, ", "),
		strings.Join(q.Placeholders(1, len(columns)), ", "),
		strings.Join(columns, ", "),
	)

	err = q.QueryRow(query, override.Values()...).Scan(override.Pointers()...)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert threshold override: %w", err)
	}

	// Raw SQL skips reform's hooks.
	err = override.AfterFind()
	if err != nil {
		return nil, err
	}

	return override, nil
}

// ClearThresholdOverride deletes an override. Clearing one that does not exist is a no-op,
// so a retried or concurrent clear succeeds.
func ClearThresholdOverride(q *reform.Querier, ruleID, paramName string, scope ThresholdScope, target string) error {
	err := checkThresholdOverrideKey(ruleID, paramName, scope, target)
	if err != nil {
		return err
	}

	tail := whereAllEqual(q, "rule_id", "param_name", "scope", "target")
	_, err = q.DeleteFrom(AlertRuleThresholdOverrideTable, tail, ruleID, paramName, string(scope), target)
	if err != nil {
		return fmt.Errorf("failed to clear threshold override: %w", err)
	}

	return nil
}

// DeleteThresholdOverridesForTarget deletes every override for a target, for entity removal.
//
// Cluster scope is rejected: there is no "delete a cluster" operation to hook, and a
// cluster override with no matching services is dormant rather than stale - services may
// be added to that cluster later, and the override should apply again when they are.
func DeleteThresholdOverridesForTarget(q *reform.Querier, scope ThresholdScope, target string) error {
	err := scope.Validate()
	if err != nil {
		return err
	}

	if scope == ThresholdScopeCluster {
		return status.Error(codes.InvalidArgument, "Cluster-scoped threshold overrides are not deleted by target removal.")
	}

	if target == "" {
		return status.Error(codes.InvalidArgument, "Empty target.")
	}

	tail := whereAllEqual(q, "scope", "target")
	_, err = q.DeleteFrom(AlertRuleThresholdOverrideTable, tail, string(scope), target)
	if err != nil {
		return fmt.Errorf("failed to delete threshold overrides: %w", err)
	}

	return nil
}

// DeleteAlertRule removes a rule registry row. Its overrides go with it through the
// foreign key's ON DELETE CASCADE.
func DeleteAlertRule(q *reform.Querier, ruleID string) error {
	_, err := FindAlertRuleByID(q, ruleID)
	if err != nil {
		return err
	}

	err = q.Delete(&AlertRule{RuleID: ruleID})
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	return nil
}
