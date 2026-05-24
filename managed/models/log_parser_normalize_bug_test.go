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
