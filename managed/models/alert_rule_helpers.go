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
			return nil, status.Errorf(codes.NotFound, "Alert rule with ID %q not found.", ruleID)
		}

		return nil, err
	}

	return rule, nil
}

// FindAllThresholdOverrides returns every threshold override row, tombstones included.
// The collector needs the tombstones: they are what keeps a cleared target's series
// alive at the rule's default.
func FindAllThresholdOverrides(q *reform.Querier) ([]*AlertRuleThresholdOverride, error) {
	return selectThresholdOverrides(q, "")
}

// FindThresholdOverridesByRule returns every override row for one rule, tombstones included.
func FindThresholdOverridesByRule(q *reform.Querier, ruleID string) ([]*AlertRuleThresholdOverride, error) {
	if ruleID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty rule ID.")
	}

	return selectThresholdOverrides(q, "WHERE rule_id = "+q.Placeholder(1), ruleID)
}

// FindThresholdOverridesByTarget returns every override row for one target, tombstones included.
func FindThresholdOverridesByTarget(q *reform.Querier, scope ThresholdScope, target string) ([]*AlertRuleThresholdOverride, error) {
	err := scope.Validate()
	if err != nil {
		return nil, err
	}

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty target.")
	}

	tail := fmt.Sprintf("WHERE scope = %s AND target = %s", q.Placeholder(1), q.Placeholder(2))

	return selectThresholdOverrides(q, tail, string(scope), target)
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

func findThresholdOverride(q *reform.Querier, ruleID, paramName string, scope ThresholdScope, target string) (*AlertRuleThresholdOverride, error) {
	tail := fmt.Sprintf("WHERE rule_id = %s AND param_name = %s AND scope = %s AND target = %s",
		q.Placeholder(1), q.Placeholder(2), q.Placeholder(3), q.Placeholder(4))

	override := &AlertRuleThresholdOverride{}
	err := q.SelectOneTo(override, tail, ruleID, paramName, string(scope), target)
	if err != nil {
		return nil, err
	}

	return override, nil
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

// ChangeAlertRuleGrafanaUID stores the Grafana rule UID for an already-registered rule.
// The UID is a cached handle, not the identity, so it is set after Grafana has accepted
// the rule rather than being required up front.
func ChangeAlertRuleGrafanaUID(q *reform.Querier, ruleID, grafanaRuleUID string) (*AlertRule, error) {
	rule, err := FindAlertRuleByID(q, ruleID)
	if err != nil {
		return nil, err
	}

	if grafanaRuleUID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Grafana rule UID.")
	}

	rule.GrafanaRuleUID = &grafanaRuleUID
	err = q.Update(rule)
	if err != nil {
		return nil, fmt.Errorf("failed to update alert rule: %w", err)
	}

	return rule, nil
}

// UpsertThresholdOverride sets the override for one parameter of one rule at one target,
// creating the row if it does not exist. Writing to a tombstoned row revives it.
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

	override, err := findThresholdOverride(q, ruleID, paramName, scope, target)
	switch {
	case err == nil:
		override.Value = value
		override.ClearedAt = nil
		err = q.Update(override)
		if err != nil {
			return nil, fmt.Errorf("failed to update threshold override: %w", err)
		}

		return override, nil

	case errors.Is(err, reform.ErrNoRows):
		override = &AlertRuleThresholdOverride{
			ID:        uuid.New().String(),
			RuleID:    ruleID,
			ParamName: paramName,
			Scope:     scope,
			Target:    target,
			Value:     value,
		}
		err = q.Insert(override)
		if err != nil {
			return nil, fmt.Errorf("failed to create threshold override: %w", err)
		}

		return override, nil

	default:
		return nil, fmt.Errorf("failed to look up threshold override: %w", err)
	}
}

// ClearThresholdOverride tombstones an override instead of deleting it, so the emitted
// series keeps existing and merely changes value. Deleting the row would signal the
// clear by absence, which takes a full VictoriaMetrics lookbehind to become visible.
func ClearThresholdOverride(q *reform.Querier, ruleID, paramName string, scope ThresholdScope, target string) error {
	err := checkThresholdOverrideKey(ruleID, paramName, scope, target)
	if err != nil {
		return err
	}

	override, err := findThresholdOverride(q, ruleID, paramName, scope, target)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return status.Errorf(codes.NotFound, "Threshold override for rule %q parameter %q not found.", ruleID, paramName)
		}

		return fmt.Errorf("failed to look up threshold override: %w", err)
	}

	if override.IsCleared() {
		return nil
	}

	override.ClearedAt = new(Now())
	err = q.Update(override)
	if err != nil {
		return fmt.Errorf("failed to clear threshold override: %w", err)
	}

	return nil
}

// DeleteThresholdOverridesForTarget hard-deletes every override for a target, and is for
// entity removal only. A user clearing an override tombstones it (the target still
// exists and its series must keep resolving); a removed node or service has no target
// left to emit for, so a tombstone there would be pure residue.
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

	tail := fmt.Sprintf("WHERE scope = %s AND target = %s", q.Placeholder(1), q.Placeholder(2))
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
