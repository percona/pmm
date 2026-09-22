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

package alerting

import (
	"context"
	"math"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	alerting "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// thresholdScopeFromAPI converts a scope from the wire, defaulting to node.
//
// Only node scope resolves in this increment. Service and cluster are already carried by
// the schema, the resolver and the proto, so enabling them later is a validation change
// rather than an API change - which is why they are rejected as unimplemented rather than
// as invalid.
func thresholdScopeFromAPI(scope alerting.ThresholdScope) (models.ThresholdScope, error) {
	switch scope {
	case alerting.ThresholdScope_THRESHOLD_SCOPE_UNSPECIFIED, alerting.ThresholdScope_THRESHOLD_SCOPE_NODE:
		return models.ThresholdScopeNode, nil
	case alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE:
		return "", status.Error(codes.Unimplemented, "Service-scoped threshold overrides are not supported yet.")
	case alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER:
		return "", status.Error(codes.Unimplemented, "Cluster-scoped threshold overrides are not supported yet.")
	}

	// do not add `default:` to make exhaustive linter do its job

	return "", status.Errorf(codes.InvalidArgument, "Unknown threshold scope '%s'.", scope.String())
}

func thresholdScopeToAPI(scope models.ThresholdScope) alerting.ThresholdScope {
	switch scope {
	case models.ThresholdScopeNode:
		return alerting.ThresholdScope_THRESHOLD_SCOPE_NODE
	case models.ThresholdScopeService:
		return alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE
	case models.ThresholdScopeCluster:
		return alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER
	}

	// do not add `default:` to make exhaustive linter do its job

	return alerting.ThresholdScope_THRESHOLD_SCOPE_UNSPECIFIED
}

// checkThresholdTargetExists rejects an override aimed at something that is not there.
// A cluster is a label value rather than an inventory entity, so its existence cannot be
// checked - and should not be, since a cluster override may legitimately precede the
// services that will join it.
func checkThresholdTargetExists(q *reform.Querier, scope models.ThresholdScope, target string) error {
	switch scope {
	case models.ThresholdScopeNode:
		_, err := models.FindNodeByID(q, target)

		return err
	case models.ThresholdScopeService:
		_, err := models.FindServiceByID(q, target)

		return err
	case models.ThresholdScopeCluster:
		return nil
	}

	// do not add `default:` to make exhaustive linter do its job

	return nil
}

// resolveThresholdRequest validates one set or clear against the rule registry, and
// returns the rule parameter it addresses. Checks run cheapest-first, so a malformed
// request never reaches the database.
func resolveThresholdRequest(
	q *reform.Querier,
	scope models.ThresholdScope,
	target, ruleID, paramName string,
	value *float64,
) (models.AlertRuleParam, error) {
	var zero models.AlertRuleParam

	rule, err := models.FindAlertRuleByID(q, ruleID)
	if err != nil {
		return zero, err
	}

	// The registry only holds parameters that were overridable when the rule was
	// created, so a parameter missing here is either unknown or not overridable.
	param, ok := rule.Params[paramName]
	if !ok {
		return zero, status.Errorf(codes.NotFound,
			"Rule '%s' has no overridable parameter '%s'.", ruleID, paramName)
	}

	if !param.OverridableAt(scope) {
		return zero, status.Errorf(codes.InvalidArgument,
			"Parameter '%s' cannot be overridden at '%s' scope.", paramName, scope)
	}

	if value != nil {
		err = checkThresholdValue(paramName, param, *value)
		if err != nil {
			return zero, err
		}
	}

	err = checkThresholdTargetExists(q, scope, target)
	if err != nil {
		return zero, err
	}

	return param, nil
}

// checkThresholdValue guards the value at the API as well as the database. The column's
// CHECK rejects non-finite values too, but reaching it would surface as an opaque
// internal error rather than a bad request.
func checkThresholdValue(paramName string, param models.AlertRuleParam, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return status.Errorf(codes.InvalidArgument, "Threshold for '%s' must be a finite number.", paramName)
	}

	if param.Min != nil && value < *param.Min {
		return status.Errorf(codes.InvalidArgument,
			"Threshold for '%s' must be at least %v.", paramName, *param.Min)
	}

	if param.Max != nil && value > *param.Max {
		return status.Errorf(codes.InvalidArgument,
			"Threshold for '%s' must be at most %v.", paramName, *param.Max)
	}

	return nil
}

// thresholdFromResolved builds the API view of one parameter as it applies to one target.
func thresholdFromResolved(ruleID, paramName string, param models.AlertRuleParam, resolved models.ResolvedThreshold) *alerting.Threshold {
	threshold := &alerting.Threshold{
		RuleId:         ruleID,
		ParamName:      paramName,
		Summary:        param.Summary,
		Unit:           convertParamUnit(models.ParamUnit(param.Unit)),
		DefaultValue:   param.Default,
		EffectiveValue: resolved.Value,
		IsOverridden:   resolved.IsOverridden(),
	}

	if resolved.Source != nil {
		threshold.Scope = thresholdScopeToAPI(resolved.Source.Scope)
		threshold.Target = resolved.Source.Target
	}

	return threshold
}

// ListThresholds returns per-target threshold overrides.
//
// As a side effect, throttled to once per reconcileInterval, this also triggers an async
// sweep that reaps alert-rule registry rows whose Grafana rule no longer exists.
func (s *Service) ListThresholds(ctx context.Context, req *alerting.ListThresholdsRequest) (*alerting.ListThresholdsResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	// Converted before the transaction opens: it only reads the request, and a bad scope
	// should be rejected without taking a connection.
	scope, err := thresholdScopeFromAPI(req.Scope)
	if err != nil {
		return nil, err
	}

	var thresholds []*alerting.Threshold

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		rules, err := s.thresholdRules(tx.Querier, req.RuleId)
		if err != nil {
			return err
		}

		overrides, err := thresholdOverridesFor(tx.Querier, req.RuleId)
		if err != nil {
			return err
		}

		inv, err := loadThresholdInventory(tx.Querier, overrides)
		if err != nil {
			return err
		}

		targetName, err := s.thresholdTargetName(tx.Querier, scope, req.Target, &inv)
		if err != nil {
			return err
		}

		byRule := make(map[string][]*models.AlertRuleThresholdOverride, len(rules))
		for _, override := range overrides {
			byRule[override.RuleID] = append(byRule[override.RuleID], override)
		}

		for _, rule := range rules {
			thresholds = append(thresholds,
				thresholdsForRule(rule, byRule[rule.RuleID], inv, targetName, scope)...)
		}

		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	sortThresholds(thresholds)

	s.maybeReconcile(ctx)

	return &alerting.ListThresholdsResponse{Thresholds: thresholds}, nil
}

// thresholdOverridesFor loads one rule's override rows, or every rule's when none is named.
func thresholdOverridesFor(q *reform.Querier, ruleID string) ([]*models.AlertRuleThresholdOverride, error) {
	if ruleID != "" {
		return models.FindThresholdOverridesByRule(q, ruleID)
	}

	return models.FindAllThresholdOverrides(q)
}

// thresholdsForRule reports one registry row's thresholds. With no target it reports only
// what has actually been overridden; with a target it reports every parameter of the rule
// that can be overridden at the requested scope, falling back to that rule's own default
// where nothing overrides it.
func thresholdsForRule(
	rule *models.AlertRule,
	overrides []*models.AlertRuleThresholdOverride,
	inv models.ThresholdInventory,
	targetName string,
	scope models.ThresholdScope,
) []*alerting.Threshold {
	var thresholds []*alerting.Threshold

	for paramName, param := range rule.Params {
		// Listing a parameter the write path rejects puts a field in the UI that cannot
		// be saved, and one rejected row rolls back the whole batch.
		if targetName != "" && !param.OverridableAt(scope) {
			continue
		}

		resolved := models.ResolveThresholdsDetailed(
			filterOverridesByParam(overrides, paramName), param.Default, inv,
		)

		if targetName == "" {
			// With no target there is no bounded set of targets to enumerate, so
			// only what has actually been overridden is reported.
			for _, entry := range resolved {
				if entry.IsOverridden() {
					thresholds = append(thresholds, thresholdFromResolved(rule.RuleID, paramName, param, entry))
				}
			}

			continue
		}

		entry, ok := resolved[targetName]
		if !ok {
			entry = models.ResolvedThreshold{Value: param.Default}
		}

		thresholds = append(thresholds, thresholdFromResolved(rule.RuleID, paramName, param, entry))
	}

	return thresholds
}

// thresholdRules returns the registry rows to report on, honouring an optional filter.
func (s *Service) thresholdRules(q *reform.Querier, ruleID string) ([]*models.AlertRule, error) {
	if ruleID != "" {
		rule, err := models.FindAlertRuleByID(q, ruleID)
		if err != nil {
			return nil, err
		}

		return []*models.AlertRule{rule}, nil
	}

	return models.FindAlertRules(q)
}

// thresholdTargetName resolves the requested target to its join-label value and makes
// sure the inventory carries it, so a target with no override of its own still resolves.
func (s *Service) thresholdTargetName(
	q *reform.Querier,
	scope models.ThresholdScope,
	target string,
	inv *models.ThresholdInventory,
) (string, error) {
	if target == "" {
		return "", nil
	}

	switch scope {
	case models.ThresholdScopeNode:
		node, err := models.FindNodeByID(q, target)
		if err != nil {
			return "", err
		}

		if inv.NodeNames == nil {
			inv.NodeNames = make(map[string]string, 1)
		}
		inv.NodeNames[target] = node.NodeName

		return node.NodeName, nil

	case models.ThresholdScopeService:
		service, err := models.FindServiceByID(q, target)
		if err != nil {
			return "", err
		}

		if inv.ServiceNames == nil {
			inv.ServiceNames = make(map[string]string, 1)
		}
		inv.ServiceNames[target] = service.ServiceName

		return service.ServiceName, nil

	case models.ThresholdScopeCluster:
		return "", status.Error(codes.InvalidArgument, "Cluster is not a listable target.")
	}

	// do not add `default:` to make exhaustive linter do its job

	return "", nil
}

func filterOverridesByParam(overrides []*models.AlertRuleThresholdOverride, paramName string) []*models.AlertRuleThresholdOverride {
	filtered := make([]*models.AlertRuleThresholdOverride, 0, len(overrides))
	for _, override := range overrides {
		if override.ParamName == paramName {
			filtered = append(filtered, override)
		}
	}

	return filtered
}

// sortThresholds gives the response a stable order, since it is assembled from map
// iteration over a rule's parameters.
func sortThresholds(thresholds []*alerting.Threshold) {
	sort.Slice(thresholds, func(i, j int) bool {
		if thresholds[i].RuleId != thresholds[j].RuleId {
			return thresholds[i].RuleId < thresholds[j].RuleId
		}

		if thresholds[i].ParamName != thresholds[j].ParamName {
			return thresholds[i].ParamName < thresholds[j].ParamName
		}

		return thresholds[i].Target < thresholds[j].Target
	})
}

// SetThreshold overrides one rule parameter for one target.
func (s *Service) SetThreshold(_ context.Context, req *alerting.SetThresholdRequest) (*alerting.SetThresholdResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	var threshold *alerting.Threshold

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		scope, err := thresholdScopeFromAPI(req.Scope)
		if err != nil {
			return err
		}

		param, err := resolveThresholdRequest(tx.Querier, scope, req.Target, req.RuleId, req.ParamName, &req.Value)
		if err != nil {
			return err
		}

		_, err = models.UpsertThresholdOverride(tx.Querier, req.RuleId, req.ParamName, scope, req.Target, req.Value)
		if err != nil {
			return err
		}

		threshold, err = s.readThreshold(tx.Querier, req.RuleId, req.ParamName, param, scope, req.Target)

		return err
	})
	if errTx != nil {
		return nil, errTx
	}

	return &alerting.SetThresholdResponse{Threshold: threshold}, nil
}

// ClearThreshold removes an override so the target falls back to the rule's default, or
// to a broader override still covering it.
func (s *Service) ClearThreshold(_ context.Context, req *alerting.ClearThresholdRequest) (*alerting.ClearThresholdResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		scope, err := thresholdScopeFromAPI(req.Scope)
		if err != nil {
			return err
		}

		_, err = resolveThresholdRequest(tx.Querier, scope, req.Target, req.RuleId, req.ParamName, nil)
		if err != nil {
			return err
		}

		return models.ClearThresholdOverride(tx.Querier, req.RuleId, req.ParamName, scope, req.Target)
	})
	if errTx != nil {
		return nil, errTx
	}

	return &alerting.ClearThresholdResponse{}, nil
}

// BatchUpdateThresholds applies several set and clear operations in one transaction, so
// a client editing many rows at once never lands a partial result it cannot report.
func (s *Service) BatchUpdateThresholds(_ context.Context, req *alerting.BatchUpdateThresholdsRequest) (*alerting.BatchUpdateThresholdsResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	thresholds := make([]*alerting.Threshold, 0, len(req.Updates))

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		thresholds = thresholds[:0]

		for _, update := range req.Updates {
			scope, err := thresholdScopeFromAPI(update.Scope)
			if err != nil {
				return err
			}

			param, err := resolveThresholdRequest(tx.Querier, scope, update.Target, update.RuleId, update.ParamName, update.Value)
			if err != nil {
				return err
			}

			if update.Value == nil {
				err = models.ClearThresholdOverride(tx.Querier, update.RuleId, update.ParamName, scope, update.Target)
				if err != nil {
					return err
				}

				continue
			}

			_, err = models.UpsertThresholdOverride(tx.Querier, update.RuleId, update.ParamName, scope, update.Target, *update.Value)
			if err != nil {
				return err
			}

			threshold, err := s.readThreshold(tx.Querier, update.RuleId, update.ParamName, param, scope, update.Target)
			if err != nil {
				return err
			}

			thresholds = append(thresholds, threshold)
		}

		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	return &alerting.BatchUpdateThresholdsResponse{Thresholds: thresholds}, nil
}

// readThreshold reports a parameter as it stands for one target after a write, resolved
// through the same precedence the collector applies.
func (s *Service) readThreshold(
	q *reform.Querier,
	ruleID, paramName string,
	param models.AlertRuleParam,
	scope models.ThresholdScope,
	target string,
) (*alerting.Threshold, error) {
	overrides, err := models.FindThresholdOverridesByRule(q, ruleID)
	if err != nil {
		return nil, err
	}

	overrides = filterOverridesByParam(overrides, paramName)

	inv, err := loadThresholdInventory(q, overrides)
	if err != nil {
		return nil, err
	}

	targetName, err := s.thresholdTargetName(q, scope, target, &inv)
	if err != nil {
		return nil, err
	}

	resolved := models.ResolveThresholdsDetailed(overrides, param.Default, inv)

	entry, ok := resolved[targetName]
	if !ok {
		entry = models.ResolvedThreshold{Value: param.Default}
	}

	return thresholdFromResolved(ruleID, paramName, param, entry), nil
}
