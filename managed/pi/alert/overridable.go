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
	"strings"

	"github.com/prometheus/prometheus/promql/parser"
)

// ParamTokenRegexp returns a regexp matching a parameter's placeholder token, tolerating
// the optional whitespace the template syntax allows, e.g. both `[[ .threshold ]]` and
// `[[.threshold]]`. The name is quoted, so any parameter name is safe to pass.
func ParamTokenRegexp(name string) *regexp.Regexp {
	return regexp.MustCompile(`\[\[\s*\.` + regexp.QuoteMeta(name) + `\s*\]\]`)
}

// ParamReferencedInExpressions reports whether any expression step references the parameter.
func (r *Template) ParamReferencedInExpressions(name string) bool {
	re := ParamTokenRegexp(name)
	for _, expression := range r.Expressions {
		if re.MatchString(expression.Expression) {
			return true
		}
	}

	return false
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

// SingleExprSplit is a single-expression template taken apart so the builder can emit the
// same three steps a multi-expression template produces: the observed query, an injected
// threshold, and a math comparison between them.
type SingleExprSplit struct {
	// LHS is everything left of the comparison, sliced from the original text so the
	// author's formatting survives.
	LHS string
	// Operator is the comparison the template used, e.g. "<" or ">=".
	Operator string
}

// SplitSingleExpr takes a single-expression template apart at its final comparison.
//
// The parameter token is not valid PromQL, so it is replaced by a numeric sentinel padded
// to the token's exact byte length. AST positions then map 1:1 onto the original text,
// which is what lets the left-hand side be sliced out of the original string rather than
// printed back from the AST - preserving line breaks and spacing exactly as written.
//
// Splitting on the AST rather than with a regexp is deliberate. A regexp has to special-case
// parentheses, the `bool` modifier, vector matching and nested comparisons, and it fails
// silently by producing a plausible but wrong left-hand side.
func SplitSingleExpr(expr, paramName string) (SingleExprSplit, error) {
	var zero SingleExprSplit

	token := ParamTokenRegexp(paramName).FindString(expr)
	if token == "" {
		return zero, fmt.Errorf("parameter %q is not referenced in the expression", paramName)
	}

	// "0" padded with spaces to the token's exact byte length: parseable, and leaves every
	// following position unchanged.
	sentinel := "0" + strings.Repeat(" ", len(token)-1)
	probe := strings.Replace(expr, token, sentinel, 1)

	// Default options: templates are ordinary PromQL, so nothing experimental is enabled.
	parsed, err := parser.NewParser(parser.Options{}).ParseExpr(probe)
	if err != nil {
		return zero, fmt.Errorf("failed to parse expression: %w", err)
	}

	binary, ok := parsed.(*parser.BinaryExpr)
	if !ok || !binary.Op.IsComparisonOperator() {
		return zero, errors.New("an overridable parameter must be the right-hand side of the expression's top-level comparison")
	}

	// Anything other than the bare sentinel means the token was used inside a larger
	// expression, e.g. `foo > [[ .threshold ]] * 100`, which cannot become a threshold step.
	number, ok := binary.RHS.(*parser.NumberLiteral)
	if !ok || number.Val != 0 {
		return zero, errors.New("an overridable parameter must be compared directly, not used inside a larger expression")
	}

	// Vector matching on the comparison itself needs no guard: PromQL only allows it
	// between two instant vectors, and the threshold is a scalar, so such an expression
	// fails to parse above with "vector matching only allowed between instant vectors".
	// Matching on an operator *inside* the left-hand side is untouched and stays valid.

	// The `bool` modifier is read and then dropped: Grafana math comparisons already yield
	// 0/1, so carrying it across would be redundant.
	position := binary.LHS.PositionRange()

	return SingleExprSplit{
		LHS:      strings.TrimSpace(expr[position.Start:position.End]),
		Operator: binary.Op.String(),
	}, nil
}

// validateOverridableParams checks the constraints that depend on the template's shape,
// rather than on the parameter alone.
func (r *Template) validateOverridableParams() error {
	overridable := r.OverridableParams()

	for _, param := range overridable {
		if r.UsesMultipleExpressions() {
			// The threshold is injected as a separate query step and referenced from the
			// expression, so a parameter no expression mentions has nothing to override.
			if !r.ParamReferencedInExpressions(param.Name) {
				return fmt.Errorf("overridable parameter %q must be referenced by an expression step", param.Name)
			}

			continue
		}

		// A single-expression template is split apart at build time. Checking that here
		// means an expression that cannot be split fails when the template is parsed, with
		// the reason, rather than producing a rule whose threshold silently never applies.
		if len(overridable) > 1 {
			return fmt.Errorf(
				"a single-expression template supports at most one overridable parameter, got %d", len(overridable))
		}

		_, err := SplitSingleExpr(r.Expr, param.Name)
		if err != nil {
			return fmt.Errorf("overridable parameter %q: %w", param.Name, err)
		}
	}

	return nil
}
