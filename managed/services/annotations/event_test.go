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

package annotations

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestObjectLabels(t *testing.T) {
	t.Parallel()

	node := &models.Node{
		NodeID:   "node-1",
		NodeName: "db-host",
		NodeType: models.GenericNodeType,
	}
	require.NoError(t, node.SetCustomLabels(map[string]string{"team": "a", "region": "eu"}))

	service := &models.Service{
		ServiceID:   "service-1",
		ServiceName: "mysql-1",
		ServiceType: models.MySQLServiceType,
		Environment: "prod",
	}
	require.NoError(t, service.SetCustomLabels(map[string]string{"team": "b"}))

	t.Run("node", func(t *testing.T) {
		t.Parallel()

		labels, err := ObjectLabels(node, nil)
		require.NoError(t, err)

		assert.Equal(t, "db-host", labels["node_name"])
		assert.Equal(t, "eu", labels["region"])
		assert.Equal(t, "a", labels["team"])
		assert.NotContains(t, labels, "service_name")
	})

	t.Run("service", func(t *testing.T) {
		t.Parallel()

		labels, err := ObjectLabels(node, service)
		require.NoError(t, err)

		assert.Equal(t, "db-host", labels["node_name"])
		assert.Equal(t, "mysql-1", labels["service_name"])
		assert.Equal(t, "prod", labels["environment"])
		assert.Equal(t, "b", labels["team"], "service label wins over node label")
		assert.NotContains(t, labels, "agent_id")
		assert.NotContains(t, labels, "instance")
	})

	t.Run("invalid custom labels", func(t *testing.T) {
		t.Parallel()

		broken := &models.Node{NodeID: "node-2", CustomLabels: []byte("not json")}

		_, err := ObjectLabels(broken, nil)
		require.Error(t, err)
	})
}

func TestSeriesLabels(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		event    Event
		expected map[string]string
	}{
		"reserved labels": {
			event: Event{
				AnnotationID: 42,
				Text:         "deployed v2",
				Tags:         []string{"deploy", "db"},
				Labels:       map[string]string{"node_name": "db-host"},
			},
			expected: map[string]string{
				"__name__":            MetricName,
				"pmm_annotation_id":   "42",
				"pmm_annotation_text": "deployed v2",
				"pmm_annotation_tags": "deploy,db",
				"node_name":           "db-host",
			},
		},
		"reserved labels win over object labels": {
			event: Event{
				AnnotationID: 7,
				Text:         "event text",
				Tags:         []string{"a"},
				Labels: map[string]string{
					"pmm_annotation_text": "custom",
					"pmm_annotation_tags": "custom",
					"pmm_annotation_id":   "custom",
					"__name__":            "custom",
				},
			},
			expected: map[string]string{
				"__name__":            MetricName,
				"pmm_annotation_id":   "7",
				"pmm_annotation_text": "event text",
				"pmm_annotation_tags": "a",
			},
		},
		"custom labels named text, tags and annotation_id are kept": {
			event: Event{
				AnnotationID: 7,
				Text:         "event text",
				Tags:         []string{"a"},
				Labels: map[string]string{
					"text":          "custom text",
					"tags":          "custom tags",
					"annotation_id": "custom id",
				},
			},
			expected: map[string]string{
				"__name__":            MetricName,
				"pmm_annotation_id":   "7",
				"pmm_annotation_text": "event text",
				"pmm_annotation_tags": "a",
				"text":                "custom text",
				"tags":                "custom tags",
				"annotation_id":       "custom id",
			},
		},
		"empty tags and empty values are omitted": {
			event: Event{
				AnnotationID: 1,
				Text:         "t",
				Labels:       map[string]string{"cluster": "", "pmm_annotation_tags": "custom"},
			},
			expected: map[string]string{
				"__name__":            MetricName,
				"pmm_annotation_id":   "1",
				"pmm_annotation_text": "t",
			},
		},
		"empty text is omitted": {
			event: Event{AnnotationID: 1, Tags: []string{"x"}},
			expected: map[string]string{
				"__name__":            MetricName,
				"pmm_annotation_id":   "1",
				"pmm_annotation_tags": "x",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.event.SeriesLabels())
		})
	}

	t.Run("does not modify event labels", func(t *testing.T) {
		t.Parallel()

		event := Event{AnnotationID: 1, Text: "t", Labels: map[string]string{"node_name": "n"}}
		event.SeriesLabels()

		assert.Equal(t, map[string]string{"node_name": "n"}, event.Labels)
	})

	t.Run("truncates text", func(t *testing.T) {
		t.Parallel()

		event := Event{Text: strings.Repeat("x", MaxTextBytes+1)}

		assert.Len(t, event.SeriesLabels()["pmm_annotation_text"], MaxTextBytes)
	})
}

func TestTruncateText(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		input    string
		expected string
	}{
		"over the limit": {
			input:    strings.Repeat("x", 5000),
			expected: strings.Repeat("x", MaxTextBytes),
		},
		"at the limit": {
			input:    strings.Repeat("x", MaxTextBytes),
			expected: strings.Repeat("x", MaxTextBytes),
		},
		"multibyte rune straddles the limit": {
			input:    strings.Repeat("x", MaxTextBytes-1) + "é",
			expected: strings.Repeat("x", MaxTextBytes-1),
		},
		"multibyte rune ends at the limit": {
			input:    strings.Repeat("x", MaxTextBytes-2) + "é" + "y",
			expected: strings.Repeat("x", MaxTextBytes-2) + "é",
		},
		"short": {
			input:    "hello",
			expected: "hello",
		},
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := truncateText(tc.input)

			assert.Equal(t, tc.expected, actual)
			assert.LessOrEqual(t, len(actual), MaxTextBytes)
			assert.True(t, utf8.ValidString(actual))
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	// __name__, pmm_annotation_id and pmm_annotation_text come on top of the object labels.
	const reserved = 3

	objectLabels := func(n int) map[string]string {
		labels := make(map[string]string, n)
		for i := range n {
			labels[fmt.Sprintf("label_%02d", i)] = "v"
		}
		return labels
	}

	for name, tc := range map[string]struct {
		objectLabels int
		valid        bool
	}{
		"at the limit":   {objectLabels: MaxLabels - reserved, valid: true},
		"over the limit": {objectLabels: MaxLabels - reserved + 1, valid: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := Event{AnnotationID: 1, Text: "t", Labels: objectLabels(tc.objectLabels)}

			err := event.Validate()
			if tc.valid {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, ErrTooManyLabels)
			assert.ErrorContains(t, err, strconv.Itoa(tc.objectLabels+reserved))
		})
	}
}

func TestValidateLabelValueLength(t *testing.T) {
	t.Parallel()

	// tagsOf returns n tags of ten bytes each, which join to n*11-1 bytes.
	tagsOf := func(n int) []string {
		tags := make([]string, n)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag-%06d", i)
		}
		return tags
	}

	for name, tc := range map[string]struct {
		event Event
		label string
		bytes int
	}{
		"joined tags over the limit": {
			// 373 tags join to 4,102 bytes.
			event: Event{AnnotationID: 1, Text: "t", Tags: tagsOf(373)},
			label: "pmm_annotation_tags",
			bytes: 4102,
		},
		"custom label value over the limit": {
			event: Event{AnnotationID: 1, Text: "t", Labels: map[string]string{"team": strings.Repeat("x", MaxTextBytes+1)}},
			label: "team",
			bytes: MaxTextBytes + 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.event.Validate()
			require.ErrorIs(t, err, ErrLabelValueTooLong)
			require.ErrorContains(t, err, tc.label)
			require.ErrorContains(t, err, strconv.Itoa(tc.bytes))
		})
	}

	for name, event := range map[string]Event{
		"joined tags at the limit": {
			AnnotationID: 1,
			Text:         "t",
			Tags:         append(tagsOf(372), strings.Repeat("y", MaxTextBytes-372*11)),
		},
		"custom label value at the limit": {
			AnnotationID: 1,
			Text:         "t",
			Labels:       map[string]string{"team": strings.Repeat("x", MaxTextBytes)},
		},
		"text over the limit is truncated, not rejected": {
			AnnotationID: 1,
			Text:         strings.Repeat("x", MaxTextBytes+1),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			longest := 0
			for _, value := range event.SeriesLabels() {
				longest = max(longest, len(value))
			}
			assert.Equal(t, MaxTextBytes, longest)

			require.NoError(t, event.Validate())
		})
	}
}
