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

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// CreateAlertRuleParams are params for registering an alert rule created from a template.
type CreateAlertRuleParams struct {
	RuleID        string
	TemplateName  string
	FolderUID     string
	RuleGroup     string
	RuleTitle     string
	DefaultParams AlertRuleDefaultParams
}

// CreateAlertRule inserts a registry row for a rule created from a template.
func CreateAlertRule(q *reform.Querier, params *CreateAlertRuleParams) (*AlertRule, error) {
	if params.RuleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	row := &AlertRule{
		RuleID:        params.RuleID,
		TemplateName:  params.TemplateName,
		FolderUID:     params.FolderUID,
		RuleGroup:     params.RuleGroup,
		RuleTitle:     params.RuleTitle,
		DefaultParams: params.DefaultParams,
	}
	if row.DefaultParams == nil {
		row.DefaultParams = AlertRuleDefaultParams{}
	}

	err := q.Insert(row)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert rule registry row: %w", err)
	}

	return row, nil
}

// FindAlertRules returns all registered alert rules.
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

// FindAlertRuleByID returns a registered alert rule by its ID.
func FindAlertRuleByID(q *reform.Querier, ruleID string) (*AlertRule, error) {
	if ruleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	rule := &AlertRule{RuleID: ruleID}
	err := q.Reload(rule)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Alert rule with ID %q not found.", ruleID)
		}
		return nil, err
	}

	return rule, nil
}

// DeleteAlertRule removes a registered alert rule (its overrides cascade away).
func DeleteAlertRule(q *reform.Querier, ruleID string) error {
	if ruleID == "" {
		return status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	err := q.Delete(&AlertRule{RuleID: ruleID})
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to delete alert rule %q: %w", ruleID, err)
	}

	return nil
}

// FindAllThresholdOverrides returns every per-node threshold override.
func FindAllThresholdOverrides(q *reform.Querier) ([]*AlertRuleThresholdOverride, error) {
	return selectThresholdOverrides(q, "")
}

// FindThresholdOverridesByRule returns per-node overrides for a given rule.
func FindThresholdOverridesByRule(q *reform.Querier, ruleID string) ([]*AlertRuleThresholdOverride, error) {
	return selectThresholdOverrides(q, "WHERE rule_id = "+q.Placeholder(1), ruleID)
}

// FindThresholdOverridesByNode returns per-node overrides for a given node.
func FindThresholdOverridesByNode(q *reform.Querier, nodeID string) ([]*AlertRuleThresholdOverride, error) {
	return selectThresholdOverrides(q, "WHERE node_id = "+q.Placeholder(1), nodeID)
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

// UpsertThresholdOverride creates or updates the per-node override for a
// (rule, param, node) triple and returns the resulting row.
func UpsertThresholdOverride(q *reform.Querier, ruleID, paramName, nodeID string, value float64) (*AlertRuleThresholdOverride, error) {
	existing, err := q.SelectOneFrom(AlertRuleThresholdOverrideTable,
		fmt.Sprintf("WHERE rule_id = %s AND param_name = %s AND node_id = %s", q.Placeholder(1), q.Placeholder(2), q.Placeholder(3)),
		ruleID, paramName, nodeID)
	switch {
	case errors.Is(err, reform.ErrNoRows):
		row := &AlertRuleThresholdOverride{
			ID:        uuid.New().String(),
			RuleID:    ruleID,
			ParamName: paramName,
			NodeID:    nodeID,
			Value:     value,
		}
		if err := q.Insert(row); err != nil {
			return nil, fmt.Errorf("failed to create threshold override: %w", err)
		}
		return row, nil
	case err != nil:
		return nil, fmt.Errorf("failed to look up threshold override: %w", err)
	default:
		row := existing.(*AlertRuleThresholdOverride) //nolint:forcetypeassert
		row.Value = value
		if err := q.Update(row); err != nil {
			return nil, fmt.Errorf("failed to update threshold override: %w", err)
		}
		return row, nil
	}
}

// DeleteThresholdOverride removes the per-node override for a (rule, param, node)
// triple, reverting the node to the rule default. Missing rows are a no-op.
func DeleteThresholdOverride(q *reform.Querier, ruleID, paramName, nodeID string) error {
	_, err := q.DeleteFrom(AlertRuleThresholdOverrideTable,
		fmt.Sprintf("WHERE rule_id = %s AND param_name = %s AND node_id = %s", q.Placeholder(1), q.Placeholder(2), q.Placeholder(3)),
		ruleID, paramName, nodeID)
	if err != nil {
		return fmt.Errorf("failed to delete threshold override: %w", err)
	}

	return nil
}
