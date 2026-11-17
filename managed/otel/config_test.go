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

package otel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIndentYAML(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, IndentYAML("", "  "))
	})
	t.Run("single_line", func(t *testing.T) {
		assert.Equal(t, "    - type: foo\n", IndentYAML("- type: foo", "    "))
	})
	t.Run("multi_line", func(t *testing.T) {
		block := "- type: key_value_parser\n  parse_from: body\n"
		got := IndentYAML(block, "      ")
		assert.True(t, strings.HasPrefix(got, "      - type: key_value_parser"))
		assert.Contains(t, got, "        parse_from: body")
	})
}

func TestFilelogMultilineYAML(t *testing.T) {
	t.Parallel()
	assert.Empty(t, FilelogMultilineYAML("", "    "))

	pattern := `^\d{4}-\d{2}-\d{2} \S+ \[\d+\] it's`
	block := FilelogMultilineYAML(pattern, "    ")
	assert.True(t, strings.HasPrefix(block, "    multiline:\n      line_start_pattern: '"), block)
	var parsed struct {
		Multiline struct {
			LineStartPattern string `yaml:"line_start_pattern"`
		} `yaml:"multiline"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(block), &parsed))
	assert.Equal(t, pattern, parsed.Multiline.LineStartPattern)
}
