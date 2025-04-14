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

// Package filters contains utility functions for manipulating filters.
package clickhouse

import (
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
)

func MatchersToClickHouse(matchers []*labels.Matcher) (string, error) {
	var conditions []string

	for _, m := range matchers {
		var condition string

		// TODO: implement processing for custom labels
		// if !analytics.IsDimension(m.Name) {
		// 	switch m.Type {
		// 	case labels.MatchEqual:
		// 		condition = fmt.Sprintf("label.key = '%s' AND label.value = '%s'", m.Name, escapeValue(m.Value))
		// 	case labels.MatchNotEqual:
		// 		condition = fmt.Sprintf("label.key != '%s' AND label.value != '%s'", m.Name, escapeValue(m.Value))
		// 	case labels.MatchRegexp:
		// 		condition = fmt.Sprintf("match(%s, '%s')", m.Name, clickhouseRegex(m.Value))
		// 	case labels.MatchNotRegexp:
		// 		condition = fmt.Sprintf("NOT match(%s, '%s')", m.Name, clickhouseRegex(m.Value))
		// 	default:
		// 		return "", fmt.Errorf("unsupported matcher type: %v", m.Type)
		// 	}
		// }
		switch m.Type {
		case labels.MatchEqual:
			condition = fmt.Sprintf("%s = '%s'", m.Name, escapeValue(m.Value))
		case labels.MatchNotEqual:
			condition = fmt.Sprintf("%s != '%s'", m.Name, escapeValue(m.Value))
		case labels.MatchRegexp:
			condition = fmt.Sprintf("match(%s, '%s')", m.Name, clickhouseRegex(m.Value))
		case labels.MatchNotRegexp:
			condition = fmt.Sprintf("NOT match(%s, '%s')", m.Name, clickhouseRegex(m.Value))
		default:
			return "", fmt.Errorf("unsupported matcher type: %v", m.Type)
		}

		conditions = append(conditions, condition)
	}

	return strings.Join(conditions, " AND "), nil
}

func escapeValue(value string) string {
	// Escape single quotes to counter SQL injection
	escaped := strings.ReplaceAll(value, "'", "''")

	// ClickHouse requires escaping these for LIKE/ILIKE:
	escaped = strings.ReplaceAll(escaped, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "%", `\%`)
	escaped = strings.ReplaceAll(escaped, "_", `\_`)

	return escaped
}

func clickhouseRegex(regex string) string {
	// Make quantifiers non-greedy
	return strings.ReplaceAll(regex, ".*", ".*?")
}
