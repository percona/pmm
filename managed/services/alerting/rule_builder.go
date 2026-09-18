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

	// Prefixes the ref ID of each injected threshold query.
	thresholdRefIDPrefix = "T_"

	// Ref IDs used when a single-expression template is desugared. A multi-expression
	// template names its own steps; a desugared one has none to inherit.
	desugaredQueryRefID     = "A"
	desugaredConditionRefID = "C"

	// The label the injected threshold query joins the observed query on. It follows
	// from the scope: an override targets a node by node_name, and a service - whether
	// named directly or reached through its cluster - by service_name.
	nodeJoinLabel    = "node_name"
	serviceJoinLabel = "service_name"
)

// thresholdRefIDSanitizer strips anything a Grafana ref ID cannot carry, so a parameter
// name with punctuation still yields a usable ref ID.
var thresholdRefIDSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)

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

	overridable := template.OverridableParams()
	if ruleID != "" && len(overridable) != 0 {
		return buildDesugaredRuleData(template, metricsDatasourceUID, ruleID, overridable[0], params, filters)
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

// buildDesugaredRuleData turns a single-expression template into the same three steps a
// multi-expression template produces, so its threshold can be overridden per target:
// the observed query, an injected threshold, and a math comparison between them.
func buildDesugaredRuleData(
	template *alert.Template,
	metricsDatasourceUID string,
	ruleID string,
	param alert.Parameter,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	split, err := alert.SplitSingleExpr(template.Expr, param.Name)
	if err != nil {
		return nil, "", fmt.Errorf("failed to split expression for parameter %q: %w", param.Name, err)
	}

	joinLabel, err := joinLabelForScopes(param.GetOverrideScopes())
	if err != nil {
		return nil, "", fmt.Errorf("parameter %q: %w", param.Name, err)
	}

	defaultValue, ok := params[param.Name]
	if !ok {
		return nil, "", fmt.Errorf("no value supplied for overridable parameter %q", param.Name)
	}

	// The observed query carries the alert's filters; the threshold deliberately does not,
	// since a filtered threshold would leave the targets the filter excludes with none.
	observed, err := fillAndFilterExpr(split.LHS, params, filters)
	if err != nil {
		return nil, "", err
	}

	fanOut, err := fillExprWithParams(split.LHS, params)
	if err != nil {
		return nil, "", err
	}

	// A and C are fixed here, unlike the multi-expression path where the template chooses
	// its own ref IDs, so only those two can be collided with.
	taken := map[string]struct{}{
		desugaredQueryRefID:     {},
		desugaredConditionRefID: {},
	}
	thresholdRefID := allocateThresholdRefID(param.Name, taken)

	query, err := newPromQueryData(metricsDatasourceUID, desugaredQueryRefID, observed)
	if err != nil {
		return nil, "", err
	}

	threshold, err := newPromQueryData(metricsDatasourceUID, thresholdRefID,
		thresholdQueryExpr(ruleID, param.Name, joinLabel, fanOut, defaultValue))
	if err != nil {
		return nil, "", err
	}

	// The template's `bool` modifier, if any, is dropped: Grafana math comparisons already
	// yield 0/1, so carrying it across would be redundant.
	condition, err := newMathExpressionData(desugaredConditionRefID,
		fmt.Sprintf("$%s %s $%s", desugaredQueryRefID, split.Operator, thresholdRefID))
	if err != nil {
		return nil, "", err
	}

	return []services.Data{query, threshold, condition}, desugaredConditionRefID, nil
}

func buildMultiExpressionRuleData(
	template *alert.Template,
	metricsDatasourceUID string,
	ruleID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) ([]services.Data, string, error) {
	injections, err := planThresholdInjections(template, ruleID, params)
	if err != nil {
		return nil, "", err
	}

	data := make([]services.Data, 0, len(template.Queries)+len(template.Expressions)+len(injections))

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

	for _, injection := range injections {
		item, err := newPromQueryData(metricsDatasourceUID, injection.refID, injection.expr)
		if err != nil {
			return nil, "", err
		}

		data = append(data, item)
	}

	for _, expression := range template.Expressions {
		// Swap the parameter tokens for their threshold ref IDs before filling, so the
		// default is never baked into the rule.
		body := swapOverridableTokens(expression.Expression, injections)

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

// thresholdInjection is one generated threshold query step: the ref ID the expression
// will reference, and the PromQL that resolves the effective threshold per target.
type thresholdInjection struct {
	paramName string
	refID     string
	expr      string
}

// planThresholdInjections builds one threshold query per overridable parameter. It
// returns nothing when the rule has no PMM-minted ID, which is how rules created before
// this feature - and rules with no overridable parameters - keep their previous shape.
func planThresholdInjections(template *alert.Template, ruleID string, params map[string]string) ([]thresholdInjection, error) {
	overridable := template.OverridableParams()
	if ruleID == "" || len(overridable) == 0 {
		return nil, nil
	}

	taken := make(map[string]struct{}, len(template.Queries)+len(template.Expressions))
	for _, query := range template.Queries {
		taken[query.RefID] = struct{}{}
	}
	for _, expression := range template.Expressions {
		taken[expression.RefID] = struct{}{}
	}

	injections := make([]thresholdInjection, 0, len(overridable))
	for _, param := range overridable {
		joinLabel, err := joinLabelForScopes(param.GetOverrideScopes())
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", param.Name, err)
		}

		observed, err := observedQueryForParam(template, param.Name)
		if err != nil {
			return nil, err
		}

		// The fan-out reuses the observed query with its parameters filled but its
		// filters left off: a filtered threshold would leave the targets the filter
		// excludes with no threshold at all.
		observedExpr, err := fillExprWithParams(observed.Expr, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fill query %s for parameter %q: %w", observed.RefID, param.Name, err)
		}

		defaultValue, ok := params[param.Name]
		if !ok {
			return nil, fmt.Errorf("no value supplied for overridable parameter %q", param.Name)
		}

		refID := allocateThresholdRefID(param.Name, taken)
		injections = append(injections, thresholdInjection{
			paramName: param.Name,
			refID:     refID,
			expr:      thresholdQueryExpr(ruleID, param.Name, joinLabel, observedExpr, defaultValue),
		})
	}

	return injections, nil
}

// thresholdQueryExpr renders the injected threshold step.
//
// The first clause carries the overrides, with label_replace mapping the collector's
// generic target label onto whichever label this rule joins on - without it the two
// operands of the `or` have different label sets and `or` returns both instead of
// preferring the left.
//
// The second clause manufactures the default for every target the observed query
// reports, by reusing that query and discarding its value with `* 0`. Fanning out over
// the observed query rather than over an inventory metric is what makes the threshold
// share the observed data's fate: it cannot go missing while the data it guards is still
// arriving, so a rule cannot silently stop evaluating.
//
// `max by` is load-bearing in both clauses. It strips instance and job - the threshold is
// scraped from pmm-managed, which can never match an observed series - reduces both
// operands to identical label sets so `or` prefers the left, and collapses the duplicate
// series an HA cluster emits.
func thresholdQueryExpr(ruleID, paramName, joinLabel, observedExpr, defaultValue string) string {
	return fmt.Sprintf(
		`max by (%s) (label_replace(%s{%s=%q, %s=%q}, %q, "$1", %q, "(.*)")) or (max by (%s) (%s) * 0 + %s)`,
		joinLabel,
		thresholdMetricName, thresholdRuleIDLabel, ruleID, thresholdParamLabel, paramName,
		joinLabel, thresholdTargetLabel,
		joinLabel, observedExpr, defaultValue,
	)
}

// joinLabelForScopes derives the join label from the scopes a parameter may be overridden
// at. Node overrides resolve to a node_name while service and cluster overrides both
// resolve to a service_name, so a parameter cannot mix node with the other two: a rule
// joins on one label, and overrides landing in the other namespace would never match.
func joinLabelForScopes(scopes []string) (string, error) {
	var node, service bool
	for _, scope := range scopes {
		switch scope {
		case alert.OverrideScopeNode:
			node = true
		case alert.OverrideScopeService, alert.OverrideScopeCluster:
			service = true
		default:
			return "", fmt.Errorf("unknown override scope %q", scope)
		}
	}

	if node && service {
		return "", fmt.Errorf("override scopes %v mix node with service or cluster, which join on different labels", scopes)
	}

	if node {
		return nodeJoinLabel, nil
	}

	return serviceJoinLabel, nil
}

// observedQueryForParam returns the query a parameter is compared against, which is the
// one the default clause fans out over. It is the nearest query reference to the left of
// the parameter's token, so a template comparing several queries in one expression -
// `$A > [[ .a ]] && $B > [[ .b ]]` - pairs each parameter with its own query.
func observedQueryForParam(template *alert.Template, paramName string) (alert.TemplateQuery, error) {
	token := alert.ParamTokenRegexp(paramName)

	for _, expression := range template.Expressions {
		loc := token.FindStringIndex(expression.Expression)
		if loc == nil {
			continue
		}

		preceding := expression.Expression[:loc[0]]

		var (
			found alert.TemplateQuery
			at    = -1
		)

		for _, query := range template.Queries {
			ref := regexp.MustCompile(`\$` + regexp.QuoteMeta(query.RefID) + `\b`)

			matches := ref.FindAllStringIndex(preceding, -1)
			if len(matches) == 0 {
				continue
			}

			last := matches[len(matches)-1][0]
			if last > at {
				at, found = last, query
			}
		}

		if at < 0 {
			return alert.TemplateQuery{}, fmt.Errorf(
				"overridable parameter %q is not compared against any query in expression %s", paramName, expression.RefID,
			)
		}

		return found, nil
	}

	return alert.TemplateQuery{}, fmt.Errorf("overridable parameter %q is not referenced by any expression", paramName)
}

// allocateThresholdRefID derives a ref ID for a parameter's threshold query, suffixing it
// if the template already uses that ref ID.
func allocateThresholdRefID(paramName string, taken map[string]struct{}) string {
	base := thresholdRefIDPrefix + thresholdRefIDSanitizer.ReplaceAllString(paramName, "_")

	refID := base
	for i := 1; ; i++ {
		_, clash := taken[refID]
		if !clash {
			break
		}

		refID = fmt.Sprintf("%s_%d", base, i)
	}

	taken[refID] = struct{}{}

	return refID
}

// swapOverridableTokens rewrites each overridable parameter's token to its threshold ref
// ID. Replacement is literal so that a `$` in the ref ID is never treated as an expansion.
func swapOverridableTokens(expression string, injections []thresholdInjection) string {
	for _, injection := range injections {
		expression = alert.ParamTokenRegexp(injection.paramName).
			ReplaceAllLiteralString(expression, "$"+injection.refID)
	}

	return expression
}

// desugaredValueRegexp matches Grafana's `$value` variable, but not `$values`, whose name
// starts with it. A plain string replacement would turn `$values.A` into nonsense.
var desugaredValueRegexp = regexp.MustCompile(`\$value\b`)

// desugaredBareValueRegexp matches a whole action that is nothing but `$value`, which is the
// case worth formatting rather than only renaming.
var desugaredBareValueRegexp = regexp.MustCompile(`\{\{\s*\$value\s*\}\}`)

// isDesugaredRule reports whether this rule is built by splitting a single expression apart.
func isDesugaredRule(template *alert.Template, ruleID string) bool {
	return ruleID != "" && !template.UsesMultipleExpressions() && len(template.OverridableParams()) != 0
}

// rewriteDesugaredAnnotations repoints Grafana's `$value` at the observed query.
//
// `$value` is only a single scalar when a rule has one step. A desugared rule has three, so
// the variable stops resolving and the alert text ships broken - which is why this runs for
// every desugared rule rather than only where it looks necessary.
//
// A bare `{{ $value }}` also gains formatting, since an unformatted float renders every
// digit it has. An action that pipes the value, such as `{{ $value | humanizeDuration }}`,
// keeps its pipeline and only has the variable renamed.
func rewriteDesugaredAnnotations(annotations map[string]string) {
	for key, text := range annotations {
		// Literal replacement throughout: `$values` would otherwise be read as a capture
		// group reference and silently dropped.
		rewritten := desugaredBareValueRegexp.ReplaceAllLiteralString(text,
			`{{ printf "%.2f" $values.`+desugaredQueryRefID+`.Value }}`)
		rewritten = desugaredValueRegexp.ReplaceAllLiteralString(rewritten,
			`$values.`+desugaredQueryRefID+`.Value`)

		annotations[key] = rewritten
	}
}
