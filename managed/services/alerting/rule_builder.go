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
	"encoding/json"
	"fmt"
	"strings"

	alertingv1 "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/pi/alert"
	"github.com/percona/pmm/managed/services"
)

const (
	grafanaExprDatasourceUID = "__expr__"
	queryRelativeFromSeconds = 600
	expressionTypeMath       = "math"
	queryIntervalMs          = 1000
	maxDataPoints            = 43200
)

type promQueryModel struct {
	Expr          string `json:"expr"`
	RefID         string `json:"refId"`
	Instant       bool   `json:"instant"`
	Hide          bool   `json:"hide"`
	IntervalMs    int    `json:"intervalMs"`
	MaxDataPoints int    `json:"maxDataPoints"`
}

type mathExpressionModel struct {
	Type          string            `json:"type"`
	Expression    string            `json:"expression"`
	RefID         string            `json:"refId"`
	Datasource    map[string]string `json:"datasource"`
	Hide          bool              `json:"hide"`
	IntervalMs    int               `json:"intervalMs"`
	MaxDataPoints int               `json:"maxDataPoints"`
}

func buildGrafanaRuleData(
	template *alert.Template,
	metricsDatasourceUID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	if template.UsesMultipleExpressions() {
		return buildMultiExpressionRuleData(template, metricsDatasourceUID, params, filters)
	}

	expr, err := fillAndFilterExpr(template.Expr, params, filters)
	if err != nil {
		return nil, "", err
	}

	data, err := newPromQueryData(metricsDatasourceUID, "A", expr)
	if err != nil {
		return nil, "", err
	}

	return []services.Data{data}, "A", nil
}

func buildMultiExpressionRuleData(
	template *alert.Template,
	metricsDatasourceUID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	data := make([]services.Data, 0, len(template.Queries)+len(template.Expressions))

	for _, query := range template.Queries {
		expr, err := fillAndFilterExpr(query.Expr, params, filters)
		if err != nil {
			return nil, "", fmt.Errorf("failed to fill query %s: %w", query.RefID, err)
		}

		item, err := newPromQueryData(metricsDatasourceUID, query.RefID, expr)
		if err != nil {
			return nil, "", err
		}

		data = append(data, item)
	}

	for _, expression := range template.Expressions {
		expr, err := fillExprWithParams(expression.Expression, params)
		if err != nil {
			return nil, "", fmt.Errorf("failed to fill expression %s: %w", expression.RefID, err)
		}

		item, err := newMathExpressionData(expression.RefID, expr)
		if err != nil {
			return nil, "", err
		}

		data = append(data, item)
	}

	return data, template.Condition, nil
}

func fillAndFilterExpr(expr string, params map[string]string, filters []*alertingv1.Filter) (string, error) {
	filledExpr, err := fillExprWithParams(expr, params)
	if err != nil {
		return "", err
	}

	for _, filter := range filters {
		switch filter.Type {
		case alertingv1.FilterType_FILTER_TYPE_MATCH:
			filledExpr = fmt.Sprintf(`label_match(%s, "%s", "%s")`, filledExpr, filter.Label, filter.Regexp)
		case alertingv1.FilterType_FILTER_TYPE_MISMATCH:
			filledExpr = fmt.Sprintf(`label_mismatch(%s, "%s", "%s")`, filledExpr, filter.Label, filter.Regexp)
		default:
			return "", fmt.Errorf("unknown filter type: %T", filter)
		}
	}

	return filledExpr, nil
}

func newPromQueryData(metricsDatasourceUID, refID, expr string) (services.Data, error) {
	model, err := json.Marshal(promQueryModel{
		Expr:          expr,
		RefID:         refID,
		Instant:       true,
		Hide:          false,
		IntervalMs:    queryIntervalMs,
		MaxDataPoints: maxDataPoints,
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal prom query model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: metricsDatasourceUID,
		RelativeTimeRange: services.RelativeTimeRange{
			From: queryRelativeFromSeconds,
			To:   0,
		},
		Model: model,
	}, nil
}

func newMathExpressionData(refID, expression string) (services.Data, error) {
	model, err := json.Marshal(mathExpressionModel{
		Type:       expressionTypeMath,
		Expression: expression,
		RefID:      refID,
		Datasource: map[string]string{
			"type": grafanaExprDatasourceUID,
			"uid":  grafanaExprDatasourceUID,
		},
		Hide:          false,
		IntervalMs:    queryIntervalMs,
		MaxDataPoints: maxDataPoints,
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal math expression model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: grafanaExprDatasourceUID,
		RelativeTimeRange: services.RelativeTimeRange{
			From: 0,
			To:   0,
		},
		Model: model,
	}, nil
}

func parseAlertTemplate(yamlContent string) (*alert.Template, error) {
	templates, err := alert.Parse(strings.NewReader(yamlContent), &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse alert template: %w", err)
	}

	if len(templates) != 1 {
		return nil, fmt.Errorf("expected exactly one template, got %d", len(templates))
	}

	return &templates[0], nil
}
