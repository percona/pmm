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

// SingleExprThresholdRef locates an overridable parameter's [[ .name ]] token
// inside a single-expression template's Expr.
type SingleExprThresholdRef struct {
	LHS      string // everything left of the comparison operator, verbatim
	Operator string // one of ==, !=, >, >=, <, <=
	Bool     bool   // the `bool` modifier was present
}

// paramTokenRegexp builds a regexp matching the `[[ .name ]]` template token with
// flexible whitespace, e.g. `[[.name]]` or `[[  .name  ]]`.
func paramTokenRegexp(name string) *regexp.Regexp {
	return regexp.MustCompile(`\[\[\s*\.` + regexp.QuoteMeta(name) + `\s*\]\]`)
}

// FindSingleExprThresholdRef locates paramName as the right-hand side of expr's
// final comparison. The token must occur exactly once.
func FindSingleExprThresholdRef(expr, paramName string) (SingleExprThresholdRef, error) {
	tokenRe := paramTokenRegexp(paramName)
	matches := tokenRe.FindAllStringIndex(expr, -1)
	switch len(matches) {
	case 0:
		return SingleExprThresholdRef{}, fmt.Errorf(
			"overridable parameter '%s' is not referenced as a [[ .%s ]] token in the expression",
			paramName, paramName,
		)
	case 1:
	default:
		return SingleExprThresholdRef{}, fmt.Errorf(
			"overridable parameter '%s' is referenced %d times in the expression; it must be referenced exactly once",
			paramName, len(matches),
		)
	}

	// Two-character operators are listed first because Go's regexp is leftmost-first,
	// not leftmost-longest.
	finalCmpRe := regexp.MustCompile(`(?s)(==|!=|>=|<=|>|<)\s*(bool\b\s*)?\[\[\s*\.` +
		regexp.QuoteMeta(paramName) + `\s*\]\]\s*\z`)
	loc := finalCmpRe.FindStringSubmatchIndex(expr)
	if loc == nil {
		return SingleExprThresholdRef{}, fmt.Errorf(
			"overridable parameter '%s' must be the right-hand side of the expression's final comparison, e.g. \"<query> > bool [[ .%s ]]\"",
			paramName, paramName,
		)
	}

	boolPresent := loc[4] >= 0 && loc[4] < loc[5]

	return SingleExprThresholdRef{
		LHS:      expr[:loc[2]],
		Operator: expr[loc[2]:loc[3]],
		Bool:     boolPresent,
	}, nil
}
