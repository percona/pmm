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
	// The grafanaExprDatasourceUID sentinel is the UID/type of Grafana's built-in
	// server-side expression datasource. For expression queries Grafana requires
	// both the datasource "type" and "uid" to be this literal "__expr__" value.
	grafanaExprDatasourceUID = "__expr__"
	queryRelativeFromSeconds = 600
	expressionTypeMath       = "math"
	expressionTypeReduce     = "reduce"
	expressionTypeThreshold  = "threshold"
	queryIntervalMs          = 1000
	maxDataPoints            = 43200

	clickhouseDatasourceType = "grafana-clickhouse-datasource"
	// Relative time window in seconds over which the ClickHouse log query is evaluated.
	clickhouseRelativeFromSeconds = 300
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

type clickhouseQueryModel struct {
	RefID      string            `json:"refId"`
	Datasource map[string]string `json:"datasource"`
	RawSQL     string            `json:"rawSql"`
	QueryType  string            `json:"queryType"`
}

type reduceExpressionModel struct {
	RefID      string            `json:"refId"`
	Datasource map[string]string `json:"datasource"`
	Type       string            `json:"type"`
	Expression string            `json:"expression"`
	Reducer    string            `json:"reducer"`
}

type thresholdExpressionModel struct {
	RefID      string               `json:"refId"`
	Datasource map[string]string    `json:"datasource"`
	Type       string               `json:"type"`
	Expression string               `json:"expression"`
	Conditions []thresholdCondition `json:"conditions"`
}

type thresholdCondition struct {
	Evaluator thresholdEvaluator `json:"evaluator"`
}

type thresholdEvaluator struct {
	Type   string    `json:"type"`
	Params []float64 `json:"params"`
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
			// Preserve series that don't carry the label (e.g. constant/threshold queries)
			filledExpr = fmt.Sprintf(`label_match(%s, "%s", "(%s)|")`, filledExpr, filter.Label, filter.Regexp)
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
		// Grafana's expression datasource identifies itself by the same sentinel
		// for both type and uid.
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

func newClickHouseQueryData(chUID, refID, rawSQL string) (services.Data, error) {
	model, err := json.Marshal(clickhouseQueryModel{
		RefID: refID,
		Datasource: map[string]string{
			"type": clickhouseDatasourceType,
			"uid":  chUID,
		},
		RawSQL:    rawSQL,
		QueryType: "table",
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal ClickHouse query model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: chUID,
		RelativeTimeRange: services.RelativeTimeRange{
			From: clickhouseRelativeFromSeconds,
			To:   0,
		},
		Model: model,
	}, nil
}

func newReduceExpressionData(refID, expression string) (services.Data, error) {
	model, err := json.Marshal(reduceExpressionModel{
		RefID:      refID,
		Datasource: exprDatasourceRef(),
		Type:       expressionTypeReduce,
		Expression: expression,
		Reducer:    "last",
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal reduce expression model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: grafanaExprDatasourceUID,
		Model:         model,
	}, nil
}

func newThresholdExpressionData(refID, expression string, threshold float64) (services.Data, error) {
	model, err := json.Marshal(thresholdExpressionModel{
		RefID:      refID,
		Datasource: exprDatasourceRef(),
		Type:       expressionTypeThreshold,
		Expression: expression,
		Conditions: []thresholdCondition{
			{Evaluator: thresholdEvaluator{Type: "gt", Params: []float64{threshold}}},
		},
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal threshold expression model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: grafanaExprDatasourceUID,
		Model:         model,
	}, nil
}

// exprDatasourceRef returns the datasource reference Grafana expects for server-side expression nodes.
func exprDatasourceRef() map[string]string {
	return map[string]string{
		"type": grafanaExprDatasourceUID,
		"uid":  grafanaExprDatasourceUID,
	}
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
