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
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	alerting "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/models"
)

// ListNodeThresholds returns every overridable threshold applicable to a node,
// including the template default and the effective value (the per-node override
// when set, otherwise the default).
func (s *Service) ListNodeThresholds(ctx context.Context, req *alerting.ListNodeThresholdsRequest) (*alerting.ListNodeThresholdsResponse, error) {
	var thresholds []*alerting.NodeThreshold

	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if _, err := models.FindNodeByID(tx.Querier, req.NodeId); err != nil {
			return err
		}

		rules, err := models.FindAlertRules(tx.Querier)
		if err != nil {
			return err
		}

		overrides, err := models.FindThresholdOverridesByNode(tx.Querier, req.NodeId)
		if err != nil {
			return err
		}

		// rule_id -> param -> override value
		byRuleParam := make(map[string]map[string]float64, len(overrides))
		for _, o := range overrides {
			m, ok := byRuleParam[o.RuleID]
			if !ok {
				m = make(map[string]float64)
				byRuleParam[o.RuleID] = m
			}
			m[o.ParamName] = o.Value
		}

		for _, rule := range rules {
			for paramName, defaultValue := range rule.DefaultParams {
				effective := defaultValue
				overridden := false
				if params, ok := byRuleParam[rule.RuleID]; ok {
					if v, ok := params[paramName]; ok {
						effective = v
						overridden = true
					}
				}

				thresholds = append(thresholds, s.buildNodeThreshold(rule, paramName, defaultValue, effective, overridden))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sortNodeThresholds(thresholds)

	return &alerting.ListNodeThresholdsResponse{Thresholds: thresholds}, nil
}

// SetNodeThreshold creates or updates a per-node override for an overridable
// threshold parameter of a rule.
func (s *Service) SetNodeThreshold(ctx context.Context, req *alerting.SetNodeThresholdRequest) (*alerting.SetNodeThresholdResponse, error) {
	var result *alerting.NodeThreshold

	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if _, err := models.FindNodeByID(tx.Querier, req.NodeId); err != nil {
			return err
		}

		rule, err := models.FindAlertRuleByID(tx.Querier, req.RuleId)
		if err != nil {
			return err
		}

		defaultValue, ok := rule.DefaultParams[req.ParamName]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "Parameter %q is not an overridable threshold of rule %q.", req.ParamName, req.RuleId)
		}

		if err := s.validateThresholdRange(rule, req.ParamName, req.Value); err != nil {
			return err
		}

		if _, err := models.UpsertThresholdOverride(tx.Querier, req.RuleId, req.ParamName, req.NodeId, req.Value); err != nil {
			return err
		}

		result = s.buildNodeThreshold(rule, req.ParamName, defaultValue, req.Value, true)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &alerting.SetNodeThresholdResponse{Threshold: result}, nil
}

// DeleteNodeThreshold removes a per-node override, reverting the node to the
// rule default.
func (s *Service) DeleteNodeThreshold(ctx context.Context, req *alerting.DeleteNodeThresholdRequest) (*alerting.DeleteNodeThresholdResponse, error) {
	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if _, err := models.FindNodeByID(tx.Querier, req.NodeId); err != nil {
			return err
		}

		if _, err := models.FindAlertRuleByID(tx.Querier, req.RuleId); err != nil {
			return err
		}

		return models.DeleteThresholdOverride(tx.Querier, req.RuleId, req.ParamName, req.NodeId)
	})
	if err != nil {
		return nil, err
	}

	return &alerting.DeleteNodeThresholdResponse{}, nil
}

// buildNodeThreshold assembles the API representation, enriching the summary and
// unit from the current template when it is still available.
func (s *Service) buildNodeThreshold(rule *models.AlertRule, paramName string, defaultValue, effectiveValue float64, overridden bool) *alerting.NodeThreshold {
	nt := &alerting.NodeThreshold{
		RuleId:         rule.RuleID,
		RuleTitle:      rule.RuleTitle,
		TemplateName:   rule.TemplateName,
		ParamName:      paramName,
		DefaultValue:   defaultValue,
		EffectiveValue: effectiveValue,
		IsOverridden:   overridden,
	}

	if def, ok := s.paramDefinition(rule.TemplateName, paramName); ok {
		nt.Summary = def.Summary
		nt.Unit = convertParamUnit(def.Unit)
	}

	return nt
}

// validateThresholdRange checks the value against the parameter's float range
// from the current template. If the template or its range is unavailable the
// value is accepted (the rule default itself was validated at creation).
func (s *Service) validateThresholdRange(rule *models.AlertRule, paramName string, value float64) error {
	def, ok := s.paramDefinition(rule.TemplateName, paramName)
	if !ok || def.FloatParam == nil {
		return nil
	}

	if def.FloatParam.Min != nil && value < *def.FloatParam.Min {
		return status.Errorf(codes.InvalidArgument, "Value %v is less than the minimum %v for parameter %q.", value, *def.FloatParam.Min, paramName)
	}
	if def.FloatParam.Max != nil && value > *def.FloatParam.Max {
		return status.Errorf(codes.InvalidArgument, "Value %v is greater than the maximum %v for parameter %q.", value, *def.FloatParam.Max, paramName)
	}

	return nil
}

// paramDefinition looks up a parameter definition of a currently-collected template.
func (s *Service) paramDefinition(templateName, paramName string) (models.AlertExprParamDefinition, bool) {
	tmpl, ok := s.GetTemplates()[templateName]
	if !ok {
		return models.AlertExprParamDefinition{}, false
	}

	for _, p := range tmpl.Params {
		if p.Name == paramName {
			return p, true
		}
	}

	return models.AlertExprParamDefinition{}, false
}

func sortNodeThresholds(thresholds []*alerting.NodeThreshold) {
	sort.Slice(thresholds, func(i, j int) bool {
		if thresholds[i].RuleTitle != thresholds[j].RuleTitle {
			return thresholds[i].RuleTitle < thresholds[j].RuleTitle
		}
		if thresholds[i].RuleId != thresholds[j].RuleId {
			return thresholds[i].RuleId < thresholds[j].RuleId
		}
		return thresholds[i].ParamName < thresholds[j].ParamName
	})
}
