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

package alert

import (
	"fmt"
	"regexp"
	"slices"
)

// ParamTokenRegexp returns a regexp matching a parameter's placeholder token, tolerating
// the optional whitespace the template syntax allows, e.g. both `[[ .threshold ]]` and
// `[[.threshold]]`. The name is quoted, so any parameter name is safe to pass.
func ParamTokenRegexp(name string) *regexp.Regexp {
	return regexp.MustCompile(`\[\[\s*\.` + regexp.QuoteMeta(name) + `\s*\]\]`)
}

// OverridableParams returns the template's overridable parameters, in declaration order.
func (r *Template) OverridableParams() []Parameter {
	var params []Parameter
	for _, param := range r.Params {
		if param.Overridable {
			params = append(params, param)
		}
	}

	return params
}

// ObservedQueryForParam returns the query a parameter is compared against, which is the one
// the default clause fans out over. Every occurrence must be the bare right-hand side of a
// comparison against the same query, because one threshold step serves all of them.
func (r *Template) ObservedQueryForParam(paramName string) (TemplateQuery, error) {
	token := ParamTokenRegexp(paramName)

	var found *TemplateQuery

	for _, expression := range r.Expressions {
		text := expression.Expression

		for _, loc := range token.FindAllStringIndex(text, -1) {
			if !isBareComparand(text[:loc[0]], text[loc[1]:]) {
				return TemplateQuery{}, fmt.Errorf(
					"overridable parameter '%s' in expression %s must be the whole right-hand side of a comparison, e.g. `$A > [[ .%s ]]`",
					paramName, expression.RefID, paramName,
				)
			}

			ref, ok := r.nearestStepRef(text[:loc[0]])
			if !ok {
				return TemplateQuery{}, fmt.Errorf(
					"overridable parameter '%s' is not compared against any query in expression %s", paramName, expression.RefID,
				)
			}

			query, err := r.observedQuery(paramName, expression.RefID, ref)
			if err != nil {
				return TemplateQuery{}, err
			}

			if found != nil && query.RefID != found.RefID {
				return TemplateQuery{}, fmt.Errorf(
					"overridable parameter '%s' is compared against both $%s and $%s, but can only be compared against one query",
					paramName, found.RefID, query.RefID,
				)
			}

			found = &query
		}
	}

	if found == nil {
		return TemplateQuery{}, fmt.Errorf("overridable parameter '%s' is not referenced by any expression", paramName)
	}

	return *found, nil
}

// isBareComparand reports whether the text around a token makes it the whole right-hand
// operand of a comparison, so the rule compares against exactly the value the API reports.
func isBareComparand(before, after string) bool {
	return comparisonSuffix.MatchString(before) && operandEndPrefix.MatchString(after)
}

var (
	comparisonSuffix = regexp.MustCompile(`(?:[<>]=?|[=!]=)\s*$`)
	operandEndPrefix = regexp.MustCompile(`^\s*(?:$|\)|&&|\|\|)`)
)

// nearestStepRef returns the query or expression referenced last in text, which pairs each
// parameter with its own step in an expression like `$A > [[ .a ]] && $B > [[ .b ]]`.
func (r *Template) nearestStepRef(text string) (string, bool) {
	for _, match := range slices.Backward(stepRefRegexp.FindAllStringSubmatch(text, -1)) {
		ref := match[1]
		if slices.ContainsFunc(r.Queries, func(q TemplateQuery) bool { return q.RefID == ref }) ||
			slices.ContainsFunc(r.Expressions, func(e TemplateExpression) bool { return e.RefID == ref }) {
			return ref, true
		}
	}

	return "", false
}

// observedQuery returns the query a parameter's comparand ref resolves to, which must be a
// query because its threshold fans out over it; validation keeps ClickHouse queries out of math.
func (r *Template) observedQuery(paramName, expressionRefID, ref string) (TemplateQuery, error) {
	idx := slices.IndexFunc(r.Queries, func(q TemplateQuery) bool { return q.RefID == ref })
	if idx < 0 {
		return TemplateQuery{}, fmt.Errorf(
			"overridable parameter '%s' in expression %s is compared against expression $%s, but can only be compared against a MetricsQL query",
			paramName, expressionRefID, ref,
		)
	}

	return r.Queries[idx], nil
}

// validateOverridableParams checks the constraints that depend on the template's shape,
// rather than on the parameter alone.
func (r *Template) validateOverridableParams() error {
	overridable := r.OverridableParams()
	if len(overridable) != 0 && !r.UsesMultipleExpressions() {
		return fmt.Errorf(
			"overridable parameter '%s' requires the queries and expressions template form", overridable[0].Name,
		)
	}

	for _, param := range overridable {
		// Must resolve to a query, not merely be mentioned.
		_, err := r.ObservedQueryForParam(param.Name)
		if err != nil {
			return err
		}

		// A token inside a query is filled with the default, so overrides would never reach it.
		token := ParamTokenRegexp(param.Name)
		for _, query := range r.Queries {
			if token.MatchString(query.Expr) {
				return fmt.Errorf(
					"overridable parameter '%s' is used in query %s, but can only be used in expressions", param.Name, query.RefID,
				)
			}
		}
	}

	return nil
}
