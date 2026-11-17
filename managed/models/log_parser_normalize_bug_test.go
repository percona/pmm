// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDoesNotBreakQuotedMysqlRegex(t *testing.T) {
	t.Parallel()
	indented := `    - type: regex_parser
      regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z) (?P<thread_id>\d+) \[(?P<subsystem>[^\]]+)\] \[(?P<code>[^\]]+)\] \[(?P<component>[^\]]+)\] (?P<message>.*)$'
      parse_from: body
      parse_to: attributes
    - type: time_parser
      parse_from: attributes.timestamp
      layout: '2006-01-02T15:04:05.000000Z'
      layout_type: gotime`
	normalized := NormalizeLogParserOperatorYAML(indented)
	require.NoError(t, ValidateLogParserOperatorYAML(indented))
	require.NoError(t, ValidateLogParserOperatorYAML(normalized))
	require.True(t, strings.HasPrefix(normalized, "- type:"))
}
