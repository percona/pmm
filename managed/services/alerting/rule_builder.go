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

	"github.com/prometheus/common/model"

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
	queryIntervalMs          = 1000
	maxDataPoints            = 43200

	// A ClickHouse query covers one QAN bucket plus the time QAN data takes to arrive, and
	// buckets by QAN's own minute so a finer `$__timeInterval` never splits it.
	clickhouseRelativeFromSeconds = 300
	clickhouseIntervalMs          = 60000

	// Model fields of the ClickHouse plugin's time series SQL query.
	clickhouseQueryType        = "timeseries"
	clickhouseFormatTimeSeries = 0
	clickhouseEditorType       = "sql"

	// Strict reduce mode turns a single gap into NaN, and Grafana treats a NaN condition
	// as firing.
	reduceModeDropNonNumbers = "dropNN"

	// Prefixes the ref ID of each injected threshold query.
	thresholdRefIDPrefix = "T_"

	// How long the threshold keeps an override while the overrides are not being published.
	thresholdOutageBridge = "5m"

	// The label the injected threshold query joins the observed query on. It follows
	// from the scope: an override targets a node by node_name, and a service - whether
	// named directly or reached through its cluster - by service_name.
	nodeJoinLabel    = "node_name"
	serviceJoinLabel = "service_name"
)

// thresholdRefIDSanitizer strips anything a Grafana ref ID cannot carry, so a parameter
// name with punctuation still yields a usable ref ID.
var thresholdRefIDSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)

// sqlStringEscaper escapes text for a single-quoted ClickHouse string literal.
var sqlStringEscaper = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

type promQueryModel struct {
	Expr          string `json:"expr"`
	RefID         string `json:"refId"`
	Instant       bool   `json:"instant"`
	Hide          bool   `json:"hide"`
	IntervalMs    int    `json:"intervalMs"`
	MaxDataPoints int    `json:"maxDataPoints"`
}

type clickhouseQueryModel struct {
	RefID      string `json:"refId"`
	EditorType string `json:"editorType"`
	QueryType  string `json:"queryType"`
	Format     int    `json:"format"`
	RawSQL     string `json:"rawSql"`
	IntervalMs int    `json:"intervalMs"`
}

type expressionModel struct {
	Type          string            `json:"type"`
	Expression    string            `json:"expression"`
	Reducer       string            `json:"reducer,omitempty"`
	Settings      reduceSettings    `json:"settings,omitzero"`
	RefID         string            `json:"refId"`
	Datasource    map[string]string `json:"datasource"`
	Hide          bool              `json:"hide"`
	IntervalMs    int               `json:"intervalMs"`
	MaxDataPoints int               `json:"maxDataPoints"`
}

type reduceSettings struct {
	Mode string `json:"mode"`
}

// grafanaRuleData is what the builder produced: the steps, the condition ref ID, and the
// injected threshold step for each overridable parameter.
type grafanaRuleData struct {
	data          []services.Data
	condition     string
	thresholdRefs map[string]string
}

func buildGrafanaRuleData(
	template *alert.Template,
	datasourceUIDs map[alert.Datasource]string,
	ruleID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) (grafanaRuleData, error) {
	if template.UsesMultipleExpressions() {
		return buildMultiExpressionRuleData(template, datasourceUIDs, ruleID, params, filters)
	}

	data, err := newQueryData(alert.TemplateQuery{RefID: "A", Expr: template.Expr}, datasourceUIDs, params, filters)
	if err != nil {
		return grafanaRuleData{}, err
	}

	return grafanaRuleData{data: []services.Data{data}, condition: "A"}, nil
}

func buildMultiExpressionRuleData(
	template *alert.Template,
	datasourceUIDs map[alert.Datasource]string,
	ruleID string,
	params map[string]string,
	filters []*alertingv1.Filter,
) (grafanaRuleData, error) {
	injections, err := planThresholdInjections(template, ruleID, params)
	if err != nil {
		return grafanaRuleData{}, err
	}

	data := make([]services.Data, 0, len(template.Queries)+len(template.Expressions)+len(injections))

	for _, query := range template.Queries {
		item, err := newQueryData(query, datasourceUIDs, params, filters)
		if err != nil {
			return grafanaRuleData{}, err
		}

		data = append(data, item)
	}

	for _, injection := range injections {
		metricsUID, err := datasourceUIDFor(datasourceUIDs, alert.Metrics)
		if err != nil {
			return grafanaRuleData{}, err
		}

		item, err := newPromQueryData(metricsUID, injection.refID, injection.expr)
		if err != nil {
			return grafanaRuleData{}, err
		}

		data = append(data, item)
	}

	for _, expression := range template.Expressions {
		item, err := newTemplateExpressionData(expression, injections, params)
		if err != nil {
			return grafanaRuleData{}, err
		}

		data = append(data, item)
	}

	thresholdRefs := make(map[string]string, len(injections))
	for _, injection := range injections {
		thresholdRefs[injection.paramName] = injection.refID
	}

	return grafanaRuleData{data: data, condition: template.Condition, thresholdRefs: thresholdRefs}, nil
}

// newQueryData builds a template query for the datasource it runs against.
func newQueryData(
	query alert.TemplateQuery,
	datasourceUIDs map[alert.Datasource]string,
	params map[string]string,
	filters []*alertingv1.Filter,
) (services.Data, error) {
	switch query.Datasource {
	case "", alert.Metrics:
		uid, err := datasourceUIDFor(datasourceUIDs, alert.Metrics)
		if err != nil {
			return services.Data{}, err
		}

		expr, err := fillAndFilterExpr(query.Expr, params, filters)
		if err != nil {
			return services.Data{}, fmt.Errorf("failed to fill query %s: %w", query.RefID, err)
		}

		return newPromQueryData(uid, query.RefID, expr)

	case alert.ClickHouse:
		uid, err := datasourceUIDFor(datasourceUIDs, alert.ClickHouse)
		if err != nil {
			return services.Data{}, err
		}

		sql, err := fillExprWithParams(query.Expr, params)
		if err != nil {
			return services.Data{}, fmt.Errorf("failed to fill query %s: %w", query.RefID, err)
		}

		sql, err = filterClickHouseSQL(sql, filters)
		if err != nil {
			return services.Data{}, fmt.Errorf("failed to filter query %s: %w", query.RefID, err)
		}

		return newClickHouseQueryData(uid, query.RefID, sql)
	}

	// do not add `default:` to make exhaustive linter do its job

	return services.Data{}, fmt.Errorf("query %s: unhandled datasource '%s'", query.RefID, query.Datasource)
}

// newTemplateExpressionData builds a template expression, pointing overridable parameter
// tokens at their injected threshold steps.
func newTemplateExpressionData(
	expression alert.TemplateExpression,
	injections []thresholdInjection,
	params map[string]string,
) (services.Data, error) {
	if expression.Type == alert.ExpressionReduce {
		// Grafana takes the input as a bare ref ID; it strips only a leading `$`, not `${...}`.
		return newExpressionData(expressionModel{
			Type:       alert.ExpressionReduce,
			Expression: expression.ReduceInput(),
			Reducer:    expression.Reducer,
			Settings:   reduceSettings{Mode: reduceModeDropNonNumbers},
			RefID:      expression.RefID,
		})
	}

	// Swap the parameter tokens for their threshold ref IDs before filling, so the
	// default is never baked into the rule.
	body := swapOverridableTokens(expression.Expression, injections)

	expr, err := fillExprWithParams(body, params)
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to fill expression %s: %w", expression.RefID, err)
	}

	return newExpressionData(expressionModel{
		Type:       alert.ExpressionMath,
		Expression: expr,
		RefID:      expression.RefID,
	})
}

// datasourceUIDFor returns the resolved UID of a datasource the rule needs.
func datasourceUIDFor(datasourceUIDs map[alert.Datasource]string, datasource alert.Datasource) (string, error) {
	uid := datasourceUIDs[datasource]
	if uid == "" {
		return "", fmt.Errorf("the %s datasource UID is not resolved", datasource)
	}

	return uid, nil
}

// validateFilters rejects filters no datasource can apply: unknown types, invalid label names and non-RE2 regexps.
func validateFilters(filters []*alertingv1.Filter) error {
	for _, filter := range filters {
		if filter.Type != alertingv1.FilterType_FILTER_TYPE_MATCH && filter.Type != alertingv1.FilterType_FILTER_TYPE_MISMATCH {
			return fmt.Errorf("filter type %s is not supported", filter.Type)
		}

		if !model.LegacyValidation.IsValidLabelName(filter.Label) {
			return fmt.Errorf("filter label '%s' is not a valid label name", filter.Label)
		}

		_, err := regexp.Compile(filter.Regexp)
		if err != nil {
			return fmt.Errorf("filter regexp for label '%s' is invalid: %w", filter.Label, err)
		}
	}

	return nil
}

// filterClickHouseSQL keeps the rows whose label columns pass the filters, matching like label_match and label_mismatch.
func filterClickHouseSQL(sql string, filters []*alertingv1.Filter) (string, error) {
	if len(filters) == 0 {
		return sql, nil
	}

	conditions := make([]string, 0, len(filters))
	for _, filter := range filters {
		// The label goes into the SQL as a quoted column name, unescaped.
		if !model.LegacyValidation.IsValidLabelName(filter.Label) {
			return "", fmt.Errorf("filter label '%s' is not a valid label name", filter.Label)
		}

		// Compared as text, with NULL as the empty label MetricsQL sees for a missing one.
		column := fmt.Sprintf("ifNull(toString(`%s`), '')", filter.Label)

		var condition string
		switch filter.Type {
		case alertingv1.FilterType_FILTER_TYPE_MATCH:
			// Like label_match's `(re)|`, an empty label passes too.
			condition = fmt.Sprintf("match(%s, '%s')", column, sqlStringEscaper.Replace("^(?:(?:"+filter.Regexp+")|)$"))
		case alertingv1.FilterType_FILTER_TYPE_MISMATCH:
			condition = fmt.Sprintf("NOT match(%s, '%s')", column, sqlStringEscaper.Replace("^(?:"+filter.Regexp+")$"))
		default:
			return "", fmt.Errorf("unknown filter type: %s", filter.Type)
		}

		conditions = append(conditions, condition)
	}

	// The newline before `)` keeps a trailing `-- comment` from swallowing it.
	body := strings.TrimRight(sql, "; \t\r\n")

	return fmt.Sprintf("SELECT * FROM (\n%s\n) WHERE %s", body, strings.Join(conditions, " AND ")), nil //nolint:unqueryvet // keeps the template's own columns
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
			return "", fmt.Errorf("unknown filter type: %s", filter.Type)
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

func newClickHouseQueryData(clickhouseDatasourceUID, refID, sql string) (services.Data, error) {
	model, err := json.Marshal(clickhouseQueryModel{
		RefID:      refID,
		EditorType: clickhouseEditorType,
		QueryType:  clickhouseQueryType,
		Format:     clickhouseFormatTimeSeries,
		RawSQL:     sql,
		IntervalMs: clickhouseIntervalMs,
	})
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal ClickHouse query model: %w", err)
	}

	return services.Data{
		RefID:         refID,
		DatasourceUID: clickhouseDatasourceUID,
		QueryType:     clickhouseQueryType,
		RelativeTimeRange: services.RelativeTimeRange{
			From: clickhouseRelativeFromSeconds,
			To:   0,
		},
		Model: model,
	}, nil
}

// newExpressionData completes a server-side expression step with the fields every step shares.
func newExpressionData(model expressionModel) (services.Data, error) {
	// Grafana's expression datasource identifies itself by the same sentinel
	// for both type and uid.
	model.Datasource = map[string]string{
		"type": grafanaExprDatasourceUID,
		"uid":  grafanaExprDatasourceUID,
	}
	model.IntervalMs = queryIntervalMs
	model.MaxDataPoints = maxDataPoints

	b, err := json.Marshal(model)
	if err != nil {
		return services.Data{}, fmt.Errorf("failed to marshal %s expression model: %w", model.Type, err)
	}

	return services.Data{
		RefID:         model.RefID,
		DatasourceUID: grafanaExprDatasourceUID,
		RelativeTimeRange: services.RelativeTimeRange{
			From: 0,
			To:   0,
		},
		Model: b,
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
		joinLabel, err := joinLabelForParam(param)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}

		observed, err := template.ObservedQueryForParam(param.Name)
		if err != nil {
			return nil, err
		}

		// The fan-out reuses the observed query with its parameters filled but its
		// filters left off: a filtered threshold would leave the targets the filter
		// excludes with no threshold at all.
		observedExpr, err := fillExprWithParams(observed.Expr, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fill query %s for parameter '%s': %w", observed.RefID, param.Name, err)
		}

		defaultValue, ok := params[param.Name]
		if !ok {
			return nil, fmt.Errorf("no value supplied for overridable parameter '%s'", param.Name)
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

// thresholdQueryExpr renders the injected threshold step: the override, else the last
// override seen while they are not being published, else the default fanned out over the
// observed query. Every clause is reduced to the join label so `or` prefers the left.
func thresholdQueryExpr(ruleID, paramName, joinLabel, observedExpr, defaultValue string) string {
	override := fmt.Sprintf(`%s{%s=%q, %s=%q}`,
		thresholdMetricName, thresholdRuleIDLabel, ruleID, thresholdParamLabel, paramName)

	byJoinLabel := func(expr string) string {
		return fmt.Sprintf(`max by (%s) (label_replace(%s, %q, "$1", %q, "(.*)"))`,
			joinLabel, expr, joinLabel, thresholdTargetLabel)
	}

	return fmt.Sprintf(
		`%s or (%s unless on() (%s == 1)) or (group by (%s) (%s) * %s)`,
		byJoinLabel(override),
		byJoinLabel(fmt.Sprintf("last_over_time(%s[%s])", override, thresholdOutageBridge)),
		thresholdCollectSuccessMetricName,
		joinLabel, observedExpr, defaultValue,
	)
}

// joinLabelForParam derives the join label from the scopes a parameter may be overridden at.
func joinLabelForParam(param alert.Parameter) (string, error) {
	node, err := param.OverrideJoinsOnNode()
	if err != nil {
		return "", err
	}

	if node {
		return nodeJoinLabel, nil
	}

	return serviceJoinLabel, nil
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

// rewriteOverridableAnnotations repoints an overridable parameter's placeholder at the
// threshold step that resolves it, so the alert text reports the value the rule actually
// fired on rather than the template default.
func rewriteOverridableAnnotations(annotations, thresholdRefs map[string]string) {
	for key, text := range annotations {
		for paramName, refID := range thresholdRefs {
			// `%g` renders 80 as 80 and still carries a fractional threshold's decimals.
			text = alert.ParamTokenRegexp(paramName).ReplaceAllLiteralString(text,
				`{{ printf "%g" $values.`+refID+`.Value }}`)
		}

		annotations[key] = text
	}
}
