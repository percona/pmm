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

// validateOverridableParams checks the constraints that depend on the template's shape,
// rather than on the parameter alone.
func (r *Template) validateOverridableParams() error {
	for _, param := range r.Params {
		if !param.Overridable {
			continue
		}

		// A single-expression template bakes its threshold into the RHS of a PromQL
		// comparison, which has to be split apart before a threshold step can be
		// injected. That is deliberately not supported yet, so reject it here instead
		// of silently ignoring the flag and shipping a rule that never overrides.
		if !r.UsesMultipleExpressions() {
			return fmt.Errorf("parameter %q cannot be overridable: only multi-expression templates support overridable parameters", param.Name)
		}

		// The threshold is injected as a separate query step and referenced from the
		// expression, so a parameter that no expression mentions has nothing to override.
		if !r.ParamReferencedInExpressions(param.Name) {
			return fmt.Errorf("overridable parameter %q must be referenced by an expression step", param.Name)
		}
	}

	return nil
}
