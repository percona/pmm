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
	"regexp"
	"slices"
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
	queryIntervalMs          = 1000
	maxDataPoints            = 43200

	// thresholdMetricName is the custom metric emitted by pmm-managed
	// (see managed/services/alerting/threshold_metrics.go) that carries the
	// effective per-node threshold value for each overridable parameter.
	thresholdMetricName = "pmm_alert_threshold"
	// thresholdJoinLabel is the label the injected threshold query is reduced to
	// so it matches the observed query in the Grafana math expression. v1 is
	// node-scoped, see the dynamic-thresholds plan.
	thresholdJoinLabel = "node_name"
	// thresholdRefIDPrefix prefixes the ref IDs of injected threshold queries.
	thresholdRefIDPrefix = "T_"
)

var refIDSanitizeRegexp = regexp.MustCompile(`[^A-Za-z0-9_]`)

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
	ruleID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	if template.UsesMultipleExpressions() {
		return buildMultiExpressionRuleData(template, metricsDatasourceUID, ruleID, params, filters)
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
	ruleID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	// Assign each overridable parameter a dedicated ref ID whose query resolves
	// the per-node threshold from the pmm_alert_threshold custom metric. The
	// expression steps then reference $<refID> instead of the baked-in literal.
	overridableRefs := allocateThresholdRefIDs(template)

	data := make([]services.Data, 0, len(template.Queries)+len(overridableRefs)+len(template.Expressions))

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

	// Inject one threshold query per overridable parameter. These carry no
	// template tokens and are intentionally not wrapped in the alert filters:
	// the query must return a value for every node so the math expression can
	// compare every observed series (unmatched nodes would silently never fire).
	for _, paramName := range sortedRefKeys(overridableRefs) {
		refID := overridableRefs[paramName]
		item, err := newPromQueryData(metricsDatasourceUID, refID, thresholdQueryExpr(ruleID, paramName))
		if err != nil {
			return nil, "", err
		}

		data = append(data, item)
	}

	for _, expression := range template.Expressions {
		// Swap [[ .param ]] tokens of overridable params for their $refID before
		// substituting the remaining (non-overridable) params. The default value
		// is therefore never baked into the expression; it is emitted, per node,
		// by the collector instead.
		body := expression.Expression
		for paramName, refID := range overridableRefs {
			body = swapOverridableToken(body, paramName, refID)
		}

		expr, err := fillExprWithParams(body, params)
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

// allocateThresholdRefIDs assigns a collision-free ref ID to every overridable
// parameter of the template. Ref IDs are sanitized to Grafana-safe identifiers
// and de-duplicated against the template's own query/expression ref IDs.
func allocateThresholdRefIDs(template *alert.Template) map[string]string {
	used := make(map[string]struct{}, len(template.Queries)+len(template.Expressions))
	for _, query := range template.Queries {
		used[query.RefID] = struct{}{}
	}
	for _, expression := range template.Expressions {
		used[expression.RefID] = struct{}{}
	}

	refs := make(map[string]string)
	for _, param := range template.Params {
		if !param.Overridable {
			continue
		}

		base := thresholdRefIDPrefix + refIDSanitizeRegexp.ReplaceAllString(param.Name, "_")
		refID := base
		for i := 1; ; i++ {
			if _, ok := used[refID]; !ok {
				break
			}
			refID = fmt.Sprintf("%s_%d", base, i)
		}
		used[refID] = struct{}{}
		refs[param.Name] = refID
	}

	return refs
}

// thresholdQueryExpr returns the PromQL that resolves the effective per-node
// threshold for the given rule/param. It is reduced to the join label only so it
// matches the observed query in the Grafana math expression (the raw metric also
// carries rule_id/param/job/instance labels that would otherwise break matching).
func thresholdQueryExpr(ruleID, paramName string) string {
	return fmt.Sprintf(`max by (%s) (%s{rule_id=%q, param=%q})`, thresholdJoinLabel, thresholdMetricName, ruleID, paramName)
}

// swapOverridableToken replaces every `[[ .name ]]` token (with flexible
// whitespace) in expr with the given Grafana ref reference `$refID`.
func swapOverridableToken(expr, paramName, refID string) string {
	re := regexp.MustCompile(`\[\[\s*\.` + regexp.QuoteMeta(paramName) + `\s*\]\]`)
	// ReplaceAllLiteralString avoids `$`-expansion (e.g. `$T_x` being read as a
	// capture-group reference) in the Grafana ref replacement.
	return re.ReplaceAllLiteralString(expr, "$"+refID)
}

// sortedRefKeys returns the map keys sorted, for deterministic query ordering.
func sortedRefKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
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
