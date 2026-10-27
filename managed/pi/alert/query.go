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
	"strings"
)

// TemplateQuery represents a PromQL query step in an alert template.
type TemplateQuery struct {
	RefID string `yaml:"ref_id"`
	Expr  string `yaml:"expr"`
}

// TemplateExpression represents a Grafana server-side expression step in an alert template.
type TemplateExpression struct {
	RefID      string `yaml:"ref_id"`
	Type       string `yaml:"type"`
	Expression string `yaml:"expression"`
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

	refs := make(map[string]struct{}, len(r.Queries)+len(r.Expressions))
	err := validateQueries(r.Queries, refs)
	if err != nil {
		return err
	}

	err = validateExpressions(r.Expressions, refs)
	if err != nil {
		return err
	}

	if _, ok := refs[r.Condition]; !ok {
		return fmt.Errorf("template condition %q does not match any query or expression ref_id", r.Condition)
	}

	return nil
}

func validateQueries(queries []TemplateQuery, refs map[string]struct{}) error {
	for _, query := range queries {
		if query.RefID == "" {
			return errors.New("template query ref_id is empty")
		}
		if strings.TrimSpace(query.Expr) == "" {
			return fmt.Errorf("template query %s expression is empty", query.RefID)
		}
		if _, ok := refs[query.RefID]; ok {
			return fmt.Errorf("duplicate template query ref_id %q", query.RefID)
		}
		refs[query.RefID] = struct{}{}
	}

	return nil
}

func validateExpressions(expressions []TemplateExpression, refs map[string]struct{}) error {
	for _, expression := range expressions {
		if expression.RefID == "" {
			return errors.New("template expression ref_id is empty")
		}
		if expression.Type != "math" {
			return fmt.Errorf("template expression %s has unsupported type %q", expression.RefID, expression.Type)
		}
		if strings.TrimSpace(expression.Expression) == "" {
			return fmt.Errorf("template expression %s is empty", expression.RefID)
		}
		if _, ok := refs[expression.RefID]; ok {
			return fmt.Errorf("duplicate template expression ref_id %q", expression.RefID)
		}
		refs[expression.RefID] = struct{}{}
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
