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
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Supported template expression types.
const (
	ExpressionMath   = "math"
	ExpressionReduce = "reduce"
)

// reducers are the Grafana reduce functions a reduce expression may use.
var reducers = []string{"sum", "mean", "min", "max", "count", "last", "median"}

// A step ref ID, restricted so it can be referenced as `$A` in expressions and `$values.A` in templates.
const refIDPattern = `[A-Za-z_]\w*`

var (
	refIDRegexp = regexp.MustCompile(`^` + refIDPattern + `$`)
	// A step reference in an expression, e.g. `$A` or `${A}`.
	stepRefRegexp = regexp.MustCompile(`\$\{?(` + refIDPattern + `)\}?`)
	// A reduce input, which must be exactly one step reference.
	singleStepRefRegexp = regexp.MustCompile(`^\s*` + stepRefRegexp.String() + `\s*$`)
	// A `$values.<ref>` lookup in an annotation or label template.
	valuesRefRegexp = regexp.MustCompile(`\$values\.(` + refIDPattern + `)`)
)

// TemplateQuery represents a query step in an alert template.
type TemplateQuery struct {
	RefID      string     `yaml:"ref_id"`
	Expr       string     `yaml:"expr"`
	Datasource Datasource `yaml:"datasource,omitempty"`
}

// TemplateExpression represents a Grafana server-side expression step in an alert template.
type TemplateExpression struct {
	RefID      string `yaml:"ref_id"`
	Type       string `yaml:"type"`
	Expression string `yaml:"expression"`
	Reducer    string `yaml:"reducer,omitempty"`
}

// UsesDatasource reports whether any template query runs on the datasource, counting an unset
// datasource, and a single-expression template, as Metrics.
func (r *Template) UsesDatasource(datasource Datasource) bool {
	if !r.UsesMultipleExpressions() {
		return datasource == Metrics
	}

	return slices.ContainsFunc(r.Queries, func(q TemplateQuery) bool {
		return q.Datasource == datasource || (q.Datasource == "" && datasource == Metrics)
	})
}

// UsesMultipleExpressions reports whether the template defines queries and a condition.
func (r *Template) UsesMultipleExpressions() bool {
	return len(r.Queries) > 0
}

// StoredExpr returns the expression stored in the database and API summary field.
func (r *Template) StoredExpr() string {
	if r.Expr != "" {
		return r.Expr
	}

	if len(r.Queries) > 0 {
		return r.Queries[0].Expr
	}

	return ""
}

func (r *Template) validateSteps() error {
	if r.UsesMultipleExpressions() {
		return r.validateMultipleExpressionSteps()
	}

	return r.validateSingleExpressionSteps()
}

func (r *Template) validateMultipleExpressionSteps() error {
	if strings.TrimSpace(r.Expr) != "" {
		return errors.New("template expr should be empty for multi-expression templates")
	}

	if r.Condition == "" {
		return errors.New("template condition is empty")
	}

	if len(r.Queries) == 0 {
		return errors.New("template queries are empty")
	}

	// Maps every step ref ID to its datasource, which is empty for expressions.
	refs := make(map[string]Datasource, len(r.Queries)+len(r.Expressions))
	err := validateQueries(r.Queries, refs)
	if err != nil {
		return err
	}

	err = validateExpressions(r.Expressions, refs)
	if err != nil {
		return err
	}

	datasource, ok := refs[r.Condition]
	if !ok {
		return fmt.Errorf("template condition %q does not match any query or expression ref_id", r.Condition)
	}
	if datasource == ClickHouse {
		return fmt.Errorf("template condition %s is a ClickHouse query; add a reduce expression over it and use that instead", r.Condition)
	}

	// Checked once every ref is known, as a reduce may read a step declared after it.
	for _, expression := range r.Expressions {
		input := expression.ReduceInput()
		if input == "" {
			continue
		}

		if _, ok := refs[input]; !ok {
			return fmt.Errorf("template expression %s reduces $%s, which does not match any query or expression ref_id", expression.RefID, input)
		}
	}

	return r.validateClickHouseQueries(refs)
}

// validateClickHouseQueries checks that ClickHouse queries take only float params and are not read
// through $values, which holds a time series only once it is reduced.
func (r *Template) validateClickHouseQueries(refs map[string]Datasource) error {
	for _, param := range r.Params {
		if param.Type == Float {
			continue
		}

		token := ParamTokenRegexp(param.Name)
		for _, query := range r.Queries {
			if query.Datasource == ClickHouse && token.MatchString(query.Expr) {
				return fmt.Errorf("template query %s uses %s parameter '%s'; ClickHouse queries take only float parameters", query.RefID, param.Type, param.Name)
			}
		}
	}

	for _, texts := range []map[string]string{r.Annotations, r.Labels} {
		for name, text := range texts {
			for _, match := range valuesRefRegexp.FindAllStringSubmatch(text, -1) {
				if refs[match[1]] == ClickHouse {
					return fmt.Errorf(
						"'%s' reads $values.%s, but ClickHouse query results are not in $values; read the reduce expression over it instead",
						name, match[1],
					)
				}
			}
		}
	}

	return nil
}

// ReduceInput returns the ref ID a reduce expression reads, or "" for any other expression.
func (e TemplateExpression) ReduceInput() string {
	if e.Type != ExpressionReduce {
		return ""
	}

	match := singleStepRefRegexp.FindStringSubmatch(e.Expression)
	if match == nil {
		return ""
	}

	return match[1]
}

func validateRefID(kind, refID string) error {
	if refID == "" {
		return fmt.Errorf("template %s ref_id is empty", kind)
	}

	if !refIDRegexp.MatchString(refID) {
		return fmt.Errorf("template %s ref_id '%s' must start with a letter or underscore and contain only letters, digits and underscores", kind, refID)
	}

	return nil
}

func validateQueries(queries []TemplateQuery, refs map[string]Datasource) error {
	for _, query := range queries {
		err := validateRefID("query", query.RefID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(query.Expr) == "" {
			return fmt.Errorf("template query %s expression is empty", query.RefID)
		}
		err = query.Datasource.Validate()
		if err != nil {
			return fmt.Errorf("template query %s: %w", query.RefID, err)
		}
		if _, ok := refs[query.RefID]; ok {
			return fmt.Errorf("duplicate template query ref_id %q", query.RefID)
		}
		refs[query.RefID] = query.Datasource
	}

	return nil
}

// validateExpressions expects refs to hold every query already.
func validateExpressions(expressions []TemplateExpression, refs map[string]Datasource) error {
	for _, expression := range expressions {
		err := validateRefID("expression", expression.RefID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(expression.Expression) == "" {
			return fmt.Errorf("template expression %s is empty", expression.RefID)
		}
		switch expression.Type {
		case ExpressionMath:
			// Grafana alerts on a time series only once it is reduced.
			for _, match := range stepRefRegexp.FindAllStringSubmatch(expression.Expression, -1) {
				if refs[match[1]] == ClickHouse {
					return fmt.Errorf("template expression %s uses ClickHouse query $%s directly; reduce it first", expression.RefID, match[1])
				}
			}
		case ExpressionReduce:
			if !slices.Contains(reducers, expression.Reducer) {
				return fmt.Errorf(
					"template expression %s has unsupported reducer '%s', want one of %s",
					expression.RefID, expression.Reducer, strings.Join(reducers, ", "),
				)
			}
			if expression.ReduceInput() == "" {
				return fmt.Errorf("template expression %s must reduce a single step, e.g. `$A`", expression.RefID)
			}
		default:
			return fmt.Errorf("template expression %s has unsupported type %q", expression.RefID, expression.Type)
		}
		if _, ok := refs[expression.RefID]; ok {
			return fmt.Errorf("duplicate template expression ref_id %q", expression.RefID)
		}
		refs[expression.RefID] = ""
	}

	return nil
}

func (r *Template) validateSingleExpressionSteps() error {
	if len(r.Expressions) > 0 || r.Condition != "" {
		return errors.New("template queries are required when expressions or condition are set")
	}

	if strings.TrimSpace(r.Expr) == "" {
		return errors.New("template expression is empty")
	}

	return nil
}
