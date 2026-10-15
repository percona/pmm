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

// Package annotations represents node- and service-scoped Grafana annotations as VictoriaMetrics series.
package annotations

import (
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/percona/pmm/managed/models"
)

const (
	// MetricName is the name of the series written for every scoped annotation.
	MetricName = "pmm_annotation_event"
	// MaxTextBytes is the longest annotation text, in bytes, kept in the series.
	MaxTextBytes = 4096
	// MaxLabels is the most labels with non-empty values VictoriaMetrics accepts on a series.
	MaxLabels = 40
)

// ErrTooManyLabels is returned for an event whose series has more than MaxLabels labels.
var ErrTooManyLabels = errors.New("too many labels")

// Event is a Grafana annotation scoped to a node or a service.
type Event struct {
	AnnotationID int64
	Time         time.Time
	Text         string
	Tags         []string
	// Labels are the labels of the object the annotation is scoped to.
	Labels map[string]string
}

// ObjectLabels returns the labels of the annotated node, or of the service on that node; service is nil for a node.
func ObjectLabels(node *models.Node, service *models.Service) (map[string]string, error) {
	labels, err := models.MergeLabels(node, service, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to merge object labels: %w", err)
	}

	return labels, nil
}

// SeriesLabels returns the labels of the event's series, without empty values.
func (e Event) SeriesLabels() map[string]string {
	res := make(map[string]string, len(e.Labels))
	maps.Copy(res, e.Labels)

	res["__name__"] = MetricName
	res["annotation_id"] = strconv.FormatInt(e.AnnotationID, 10)
	res["text"] = truncateText(e.Text)
	res["tags"] = strings.Join(e.Tags, ",")

	maps.DeleteFunc(res, func(_, value string) bool { return value == "" })

	return res
}

// Validate returns ErrTooManyLabels if VictoriaMetrics would drop the event's series.
func (e Event) Validate() error {
	n := len(e.SeriesLabels())
	if n > MaxLabels {
		return fmt.Errorf("%w: %d labels, at most %d are allowed", ErrTooManyLabels, n, MaxLabels)
	}

	return nil
}

// truncateText cuts s to at most MaxTextBytes bytes without splitting a UTF-8 sequence.
func truncateText(s string) string {
	if len(s) <= MaxTextBytes {
		return s
	}

	end := MaxTextBytes
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}

	return s[:end]
}
