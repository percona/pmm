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

package models

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/percona/pmm/qan-api2/utils/logger"
)

func setup(t *testing.T, filter string) context.Context {
	t.Helper()
	encoded := base64.StdEncoding.EncodeToString([]byte(filter))
	md := metadata.Pairs(LBACHeaderName, encoded)
	ctx := metadata.NewIncomingContext(context.TODO(), md)
	return logger.SetEntry(ctx, logrus.WithField("test", t.Name()))
}

func TestParseFilters(t *testing.T) {
	t.Run("empty filters", func(t *testing.T) {
		result, err := parseFilters(nil)
		require.NoError(t, err)
		require.Nil(t, result)

		result, err = parseFilters([]string{})
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("valid filters", func(t *testing.T) {
		filters := []string{"abc", "def"}
		encoded := base64.StdEncoding.EncodeToString([]byte(`["abc", "def"]`))
		result, err := parseFilters([]string{encoded})
		require.NoError(t, err)
		require.Equal(t, filters, result)
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := parseFilters([]string{"invalid-base64"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode filters")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("invalid-json"))
		_, err := parseFilters([]string{encoded})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse JSON")
	})
}

func TestHeadersToLbacFilter(t *testing.T) {
	// Selector example: `[{service_type=~"mysql|mongodb", environment!~"prod"}, {service_type="postgresql", az!="us-east-1"}]`
	t.Run("no metadata in context", func(t *testing.T) {
		filter, err := headersToLbacFilter(context.TODO())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to extract metadata from context")
		require.Empty(t, filter)
	})

	t.Run("empty filter", func(t *testing.T) {
		md := metadata.New(map[string]string{})
		ctx := metadata.NewIncomingContext(context.TODO(), md)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Empty(t, filter)
	})

	t.Run("invalid base64 in filter", func(t *testing.T) {
		md := metadata.New(map[string]string{
			LBACHeaderName: "invalid-base64",
		})
		ctx := metadata.NewIncomingContext(context.TODO(), md)
		ctx = logger.SetEntry(ctx, logrus.WithField("test", t.Name()))
		filter, err := headersToLbacFilter(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode filters")
		require.Empty(t, filter)
	})

	t.Run("invalid JSON after decoding", func(t *testing.T) {
		invalidJSON := base64.StdEncoding.EncodeToString([]byte("invalid-json"))
		md := metadata.New(map[string]string{
			LBACHeaderName: invalidJSON,
		})
		ctx := metadata.NewIncomingContext(context.TODO(), md)
		ctx = logger.SetEntry(ctx, logrus.WithField("test", t.Name()))
		filter, err := headersToLbacFilter(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse JSON")
		require.Empty(t, filter)
	})

	t.Run("single dimension filter", func(t *testing.T) {
		ctx := setup(t, `["{service_type=\"mysql\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, "service_type = 'mysql'", filter)
	})

	t.Run("multiple dimension filters with OR", func(t *testing.T) {
		ctx := setup(t, `["{service_type=\"mysql\"}", "{service_type=\"postgresql\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, "service_type = 'mysql' OR service_type = 'postgresql'", filter)
	})

	t.Run("complex filter with multiple conditions", func(t *testing.T) {
		ctx := setup(t, `["{service_type=\"mysql\", environment!=\"dev\"}", "{service_type=\"postgresql\", environment!=\"prod\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, "(service_type = 'mysql' AND environment != 'dev') OR (service_type = 'postgresql' AND environment != 'prod')", filter)
	})

	t.Run("regex match", func(t *testing.T) {
		ctx := setup(t, `["{service_type=~\"mysql|postgresql\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, "match(service_type, 'mysql|postgresql')", filter)
	})

	t.Run("custom label", func(t *testing.T) {
		ctx := setup(t, `["{custom_label=\"value\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, "(hasAny(labels.key, ['custom_label']) AND hasAny(labels.value, ['value']))", filter)
	})

	t.Run("complex filter with custom label and dimension", func(t *testing.T) {
		ctx := setup(t, `["{custom_label=\"value\",service_type=\"mysql\"}"]`)
		filter, err := headersToLbacFilter(ctx)
		require.NoError(t, err)
		require.Equal(t, filter, "((hasAny(labels.key, ['custom_label']) AND hasAny(labels.value, ['value'])) AND service_type = 'mysql')")
	})
}

func TestMatchersToSQL(t *testing.T) {
	t.Run("standard dimension matchers", func(t *testing.T) {
		testCases := []struct {
			name     string
			matchers []*labels.Matcher
			expected string
		}{
			{
				name: "equal matcher",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchEqual, "service_type", "mysql"),
				},
				expected: "service_type = 'mysql'",
			},
			{
				name: "not equal matcher",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchNotEqual, "environment", "dev"),
				},
				expected: "environment != 'dev'",
			},
			{
				name: "regex matcher",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchRegexp, "service_type", "mysql|postgresql"),
				},
				expected: "match(service_type, 'mysql|postgresql')",
			},
			{
				name: "not regex matcher",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchNotRegexp, "node_name", "db-.*"),
				},
				expected: "NOT match(node_name, 'db-.*')",
			},
			{
				name: "multiple matchers",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchEqual, "service_type", "mysql"),
					labels.MustNewMatcher(labels.MatchNotEqual, "environment", "prod"),
				},
				expected: "service_type = 'mysql' AND environment != 'prod'",
			},
			{
				name: "escaped value",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchEqual, "database", "my'db"),
				},
				expected: "database = 'my''db'",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := matchersToSQL(tc.matchers)
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("custom label matchers", func(t *testing.T) {
		testCases := []struct {
			name     string
			matchers []*labels.Matcher
			expected string
		}{
			{
				name: "equal matcher for custom label",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchEqual, "custom_label", "custom_value"),
				},
				expected: "(hasAny(labels.key, ['custom_label']) AND hasAny(labels.value, ['custom_value']))",
			},
			{
				name: "not equal matcher for custom label",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchNotEqual, "community", "pmm-supporters"),
				},
				expected: "NOT (hasAny(labels.key, ['community']) AND hasAny(labels.value, ['pmm-supporters']))",
			},
			{
				name: "regex matcher for custom label",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchRegexp, "project", "pmm.*"),
				},
				expected: "(hasAny(labels.key, ['project']) AND arrayExists(x -> match(x, 'pmm.*'), labels.value))",
			},
			{
				name: "not regex matcher for custom label",
				matchers: []*labels.Matcher{
					labels.MustNewMatcher(labels.MatchNotRegexp, "team", "dev.*"),
				},
				expected: "NOT (hasAny(labels.key, ['team']) AND arrayExists(x -> match(x, 'dev.*'), labels.value))",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := matchersToSQL(tc.matchers)
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("mixed dimension and custom label matchers", func(t *testing.T) {
		matchers := []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "service_type", "mysql"),
			labels.MustNewMatcher(labels.MatchEqual, "custom_label", "value"),
		}
		result, err := matchersToSQL(matchers)
		require.NoError(t, err)
		require.Equal(t, "service_type = 'mysql' AND (hasAny(labels.key, ['custom_label']) AND hasAny(labels.value, ['value']))", result)
	})

	t.Run("unsupported matcher type", func(t *testing.T) {
		// Create a custom matcher with an invalid type (5)
		invalidMatcher := &labels.Matcher{
			Type:  5, // Invalid type
			Name:  "service_type",
			Value: "mysql",
		}
		_, err := matchersToSQL([]*labels.Matcher{invalidMatcher})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported matcher type")
	})
}
