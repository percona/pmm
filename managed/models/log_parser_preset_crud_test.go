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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mysqlErrorPresetYAML = `- type: regex_parser
  regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z) (?P<thread_id>\d+) \[(?P<subsystem>[^\]]+)\] \[(?P<code>[^\]]+)\] \[(?P<component>[^\]]+)\] (?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02T15:04:05.000000Z'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.subsystem
  preset: none
  mapping:
    System: info
    Warning: warn
    Error: error
- type: move
  from: attributes.message
  to: body`

func TestValidateLogParserOperatorYAML(t *testing.T) {
	t.Parallel()

	t.Run("valid mysql_error preset", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateLogParserOperatorYAML(mysqlErrorPresetYAML))
	})

	t.Run("default single operator", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateLogParserOperatorYAML(`- type: regex_parser
  regex: '^(?P<message>.*)$'
  parse_from: body
`))
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		err := ValidateLogParserOperatorYAML("  ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("missing type", func(t *testing.T) {
		t.Parallel()
		err := ValidateLogParserOperatorYAML(`- regex: 'foo'`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid type")
	})
}

func TestNormalizeLogParserOperatorYAML(t *testing.T) {
	t.Parallel()

	t.Run("parse_from on same line as regex", func(t *testing.T) {
		t.Parallel()
		collapsed := strings.Join([]string{
			"- type: regex_parser",
			"  regex: '^(?P<message>.*)$' parse_from: body",
			"  parse_to: attributes",
		}, "\n")
		normalized := NormalizeLogParserOperatorYAML(collapsed)
		require.NoError(t, ValidateLogParserOperatorYAML(normalized))
		assert.Contains(t, normalized, "'\n  parse_from: body")
	})

	t.Run("literal backslash-n sequences", func(t *testing.T) {
		t.Parallel()
		collapsed := `- type: regex_parser\n  regex: 'foo'\n  parse_from: body`
		normalized := NormalizeLogParserOperatorYAML(collapsed)
		require.NoError(t, ValidateLogParserOperatorYAML(normalized))
		assert.Contains(t, normalized, "\n  regex:")
	})

	t.Run("double-quoted regex run into parse_from", func(t *testing.T) {
		t.Parallel()
		collapsed := strings.Join([]string{
			"- type: regex_parser",
			`  regex: "^(?P<message>.*)$" parse_from: body`,
			"  parse_to: attributes",
		}, "\n")
		normalized := NormalizeLogParserOperatorYAML(collapsed)
		require.NoError(t, ValidateLogParserOperatorYAML(normalized))
		assert.Contains(t, normalized, "\"\n  parse_from: body")
	})

	t.Run("unquoted regex with colons", func(t *testing.T) {
		t.Parallel()
		collapsed := strings.Join([]string{
			"- type: regex_parser",
			"  regex: ^(?P<timestamp>\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d+Z) (?P<message>.*)$",
			"  parse_from: body",
			"  parse_to: attributes",
		}, "\n")
		normalized := NormalizeLogParserOperatorYAML(collapsed)
		require.NoError(t, ValidateLogParserOperatorYAML(normalized))
		assert.Contains(t, normalized, "regex: '^(?P<timestamp>")
	})
}
