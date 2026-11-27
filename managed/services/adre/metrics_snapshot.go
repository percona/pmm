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

package adre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	defaultMaxSeries    = 5
	defaultMaxPoints    = 500
	defaultChangePoints = 3
	defaultSnapshotStep = 5 * time.Minute
)

type metricsSnapshotRequest struct {
	Query     string `json:"query"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Step      string `json:"step"`
	MaxSeries int    `json:"max_series"`
}

type metricsSnapshotSeries struct {
	Metric       model.Metric  `json:"metric"`
	Stats        SeriesStats   `json:"stats"`
	ChangePoints []ChangePoint `json:"change_points"`
	Anomalies    []Anomaly     `json:"anomalies"`
	PointCount   int           `json:"point_count"`
}

type metricsSnapshotResponse struct {
	From        string                  `json:"from"`
	To          string                  `json:"to"`
	Step        string                  `json:"step"`
	Series      []metricsSnapshotSeries `json:"series"`
	SeriesCount int                     `json:"series_count"`
	Truncated   bool                    `json:"truncated"`
}

// PostMetricsSnapshot handles POST /v1/adre/metrics/snapshot.
func (h *Handlers) PostMetricsSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := h.checkAdreEnabled(w); !ok {
		return
	}
	if h.vm == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "metrics backend not configured")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) //nolint:mnd
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req metricsSnapshotRequest
	if err = json.Unmarshal(body, &req); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		writeJSONError(w, http.StatusBadRequest, "query is required")
		return
	}

	start, end, err := parseSnapshotBounds(req.Start, req.End)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	step := defaultSnapshotStep
	if strings.TrimSpace(req.Step) != "" {
		step, err = time.ParseDuration(strings.TrimSpace(req.Step))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid step duration")
			return
		}
	}
	maxSeries := req.MaxSeries
	if maxSeries <= 0 {
		maxSeries = defaultMaxSeries
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()

	matrix, warns, err := h.vm.QueryRange(ctx, req.Query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		h.l.Errorf("QueryRange: %v", err)
		writeJSONError(w, http.StatusBadGateway, "failed to query metrics")
		return
	}
	for _, wmsg := range warns {
		h.l.Warn(wmsg)
	}

	if matrix.Type() != model.ValMatrix {
		writeJSONError(w, http.StatusBadRequest, "query did not return a matrix")
		return
	}
	series := matrix.(model.Matrix) //nolint:forcetypeassert
	if len(series) > maxSeries {
		writeJSONError(w, http.StatusBadRequest,
			fmt.Sprintf("query returned %d series; tighten matchers or use topk (max_series=%d)", len(series), maxSeries))
		return
	}

	resp := metricsSnapshotResponse{
		From:        start.UTC().Format(time.RFC3339),
		To:          end.UTC().Format(time.RFC3339),
		Step:        step.String(),
		SeriesCount: len(series),
	}
	out := make([]metricsSnapshotSeries, 0, len(series))
	for _, stream := range series {
		timestamps, values, truncated := sampleStreamPoints(stream, defaultMaxPoints)
		if truncated {
			resp.Truncated = true
		}
		if len(values) == 0 {
			continue
		}
		out = append(out, metricsSnapshotSeries{
			Metric:       stream.Metric,
			Stats:        computeSeriesStats(values),
			ChangePoints: findChangePoints(timestamps, values, defaultChangePoints),
			Anomalies:    findAnomalies(timestamps, values, defaultZScoreThreshold),
			PointCount:   len(values),
		})
	}
	resp.Series = out
	writeJSON(w, http.StatusOK, resp)
}

const snapshotTimeFormatHint = "use RFC3339 UTC (e.g. 2026-05-24T12:00:00Z) or Unix epoch seconds; negative seconds are relative to end (or now if end omitted)"

func parseSnapshotBounds(startRaw, endRaw string) (time.Time, time.Time, error) {
	end := time.Now().UTC()
	if strings.TrimSpace(endRaw) != "" {
		t, err := parseSnapshotTime(strings.TrimSpace(endRaw), end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end: %w", err)
		}
		end = t
	}
	startRaw = strings.TrimSpace(startRaw)
	if startRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("start is required (%s)", snapshotTimeFormatHint)
	}
	start, err := parseSnapshotTime(startRaw, end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start: %w", err)
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, errors.New("start must be before end")
	}
	return start, end, nil
}

func parseSnapshotTime(raw string, relativeTo time.Time) (time.Time, error) {
	if sec, err := strconv.ParseInt(raw, 10, 64); err == nil { //nolint:noinlineerr
		if sec < 0 {
			return relativeTo.Add(time.Duration(sec) * time.Second).UTC(), nil
		}
		return time.Unix(sec, 0).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s; got %q", snapshotTimeFormatHint, raw)
	}
	return t.UTC(), nil
}

func sampleStreamPoints(stream *model.SampleStream, maxPoints int) ([]time.Time, []float64, bool) {
	if stream == nil || len(stream.Values) == 0 {
		return nil, nil, false
	}
	values := stream.Values
	truncated := len(values) > maxPoints
	if truncated {
		values = values[len(values)-maxPoints:]
	}
	timestamps := make([]time.Time, 0, len(values))
	floats := make([]float64, 0, len(values))
	for _, sp := range values {
		v := float64(sp.Value)
		if math.IsNaN(v) {
			continue
		}
		timestamps = append(timestamps, sp.Timestamp.Time())
		floats = append(floats, v)
	}
	return timestamps, floats, truncated
}
