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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryValidate(t *testing.T) {
	t.Parallel()

	valid := []Category{
		"", // optional
		CategoryUnknown,
		CategoryPMM,
		CategoryMongoDB,
		CategoryMySQL,
		CategoryNode,
		CategoryPostgreSQL,
		CategoryProxySQL,
		CategoryValkey,
		CategoryHAProxy,
	}
	for _, c := range valid {
		require.NoError(t, c.Validate(), "category %q should be valid", c)
	}

	invalid := []Category{"MySQL", "Mysql", "mongo", "redis", "external", "garbage"}
	for _, c := range invalid {
		assert.Error(t, c.Validate(), "category %q should be invalid", c)
	}
}

func TestCategoryOrDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CategoryUnknown, Category("").OrDefault())
	assert.Equal(t, CategoryUnknown, CategoryUnknown.OrDefault())
	assert.Equal(t, CategoryMySQL, CategoryMySQL.OrDefault())
}

func templateYAML(categoryLine string) string {
	return `---
templates:
  - name: test_template
    version: 1
    summary: Test template
` + categoryLine + `    expr: 'up == 0'
    for: 1m
    severity: warning
    annotations:
      summary: Test
      description: Test
`
}

func TestParseCategory(t *testing.T) {
	t.Parallel()

	strict := &ParseParams{DisallowUnknownFields: true, DisallowInvalidTemplates: true}

	t.Run("declared category is parsed", func(t *testing.T) {
		t.Parallel()

		templates, err := Parse(strings.NewReader(templateYAML("    category: mysql\n")), strict)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, CategoryMySQL, templates[0].Category)
	})

	t.Run("absent category stays empty", func(t *testing.T) {
		t.Parallel()

		templates, err := Parse(strings.NewReader(templateYAML("")), strict)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Empty(t, templates[0].Category)
	})

	t.Run("invalid category is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := Parse(strings.NewReader(templateYAML("    category: MySQL\n")), strict)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unhandled template category")
	})
}

// TestCategoryYAMLRoundTrip guards the invariant that parsing must not materialize a
// default category. ConvertTemplate stores ToYAML output in the alert_rule_templates
// `yaml` column, which is served back to the UI verbatim, so an absent category must
// stay absent through a parse/serialize round-trip.
func TestCategoryYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	strict := &ParseParams{DisallowUnknownFields: true, DisallowInvalidTemplates: true}

	t.Run("absent category is not written back", func(t *testing.T) {
		t.Parallel()

		templates, err := Parse(strings.NewReader(templateYAML("")), strict)
		require.NoError(t, err)

		out, err := ToYAML(templates)
		require.NoError(t, err)
		assert.NotContains(t, out, "category")
	})

	t.Run("declared category survives", func(t *testing.T) {
		t.Parallel()

		templates, err := Parse(strings.NewReader(templateYAML("    category: mysql\n")), strict)
		require.NoError(t, err)

		out, err := ToYAML(templates)
		require.NoError(t, err)
		assert.Contains(t, out, "category: mysql")

		reparsed, err := Parse(strings.NewReader(out), strict)
		require.NoError(t, err)
		require.Len(t, reparsed, 1)
		assert.Equal(t, CategoryMySQL, reparsed[0].Category)
	})
}
