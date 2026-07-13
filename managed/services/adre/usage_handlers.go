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
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultUsageDays = 30

type usageTotals struct {
	TotalTokens  int64   `json:"total_tokens"`
	CachedTokens int64   `json:"cached_tokens"`
	TotalCost    float64 `json:"total_cost"`
	CallCount    int64   `json:"call_count"`
}

type usageBucket struct {
	Bucket       string  `json:"bucket"`
	Feature      string  `json:"feature,omitempty"`
	Model        string  `json:"model,omitempty"`
	TotalTokens  int64   `json:"total_tokens"`
	CachedTokens int64   `json:"cached_tokens"`
	TotalCost    float64 `json:"total_cost"`
	CallCount    int64   `json:"call_count"`
}

func parseUsageTimeRange(r *http.Request) (from, to time.Time, err error) { //nolint:nonamedreturns
	now := time.Now().UTC()
	to = now
	from = now.AddDate(0, 0, -defaultUsageDays)
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		to, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to: %w", err)
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		from, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from: %w", err)
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from must be before to")
	}
	return from, to, nil
}

func usageFilterClause(from, to time.Time, r *http.Request) (where string, args []any) { //nolint:nonamedreturns
	where = " WHERE created_at >= $1 AND created_at <= $2"
	args = append(args, from, to)
	idx := 3
	if v := strings.TrimSpace(r.URL.Query().Get("feature")); v != "" {
		where += fmt.Sprintf(" AND feature = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.TrimSpace(r.URL.Query().Get("model")); v != "" {
		where += fmt.Sprintf(" AND model = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.TrimSpace(r.URL.Query().Get("triggered_by")); v != "" {
		where += fmt.Sprintf(" AND triggered_by = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.TrimSpace(r.URL.Query().Get("investigation_id")); v != "" {
		where += fmt.Sprintf(" AND investigation_id = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.TrimSpace(r.URL.Query().Get("conversation_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil { //nolint:noinlineerr
			where += fmt.Sprintf(" AND adre_conversation_id = $%d", idx)
			args = append(args, n)
		}
	}
	return where, args
}

func (h *Handlers) queryUsageTotals(where string, args []any) (usageTotals, error) {
	var t usageTotals
	row := h.db.QueryRow(`
		SELECT COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cached_tokens), 0),
		       COALESCE(SUM(total_cost), 0), COUNT(*)
		FROM holmes_usage_events`+where, args...)
	err := row.Scan(&t.TotalTokens, &t.CachedTokens, &t.TotalCost, &t.CallCount)
	return t, err
}

// GetUsageSummary handles GET /v1/adre/usage/summary.
func (h *Handlers) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.checkAdreEnabled(w); !ok {
		return
	}
	from, to, err := parseUsageTimeRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	where, args := usageFilterClause(from, to, r)
	totals, err := h.queryUsageTotals(where, args)
	if err != nil {
		h.l.Errorf("usage totals: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage summary")
		return
	}

	byFeature, err := h.queryUsageGroup(where, args, "feature", "")
	if err != nil {
		h.l.Errorf("usage by feature: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage summary")
		return
	}
	byModel, err := h.queryUsageGroup(where, args, "model", "")
	if err != nil {
		h.l.Errorf("usage by model: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage summary")
		return
	}

	groupBy := strings.TrimSpace(r.URL.Query().Get("group_by"))
	var series []usageBucket
	switch groupBy {
	case "day":
		series, err = h.queryUsageSeriesDay(where, args, "", "")
	case "feature":
		series, err = h.queryUsageGroup(where, args, "feature", "")
	case "model":
		series, err = h.queryUsageGroup(where, args, "model", "")
	case "feature,model", "model,feature":
		series, err = h.queryUsageGroup(where, args, "feature", "model")
	default:
		series, err = h.queryUsageSeriesDay(where, args, "feature", "")
	}
	if err != nil {
		h.l.Errorf("usage series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage summary")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"from":       from.Format(time.RFC3339),
		"to":         to.Format(time.RFC3339),
		"totals":     totals,
		"series":     series,
		"by_feature": byFeature,
		"by_model":   byModel,
	})
}

func (h *Handlers) queryUsageGroup(where string, args []any, col1, col2 string) ([]usageBucket, error) {
	selectCols := "COALESCE(SUM(total_tokens),0), COALESCE(SUM(cached_tokens),0), COALESCE(SUM(total_cost),0), COUNT(*)"
	groupCols := col1
	selectList := col1
	if col2 != "" {
		selectList = col1 + ", " + col2
		groupCols = col1 + ", " + col2
	}
	q := fmt.Sprintf(`SELECT %s, %s FROM holmes_usage_events%s GROUP BY %s ORDER BY COALESCE(SUM(total_cost),0) DESC`,
		selectList, selectCols, where, groupCols)
	rows, err := h.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []usageBucket
	for rows.Next() {
		var b usageBucket
		if col2 != "" { //nolint:gocritic
			err = rows.Scan(&b.Feature, &b.Model, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		} else if col1 == "model" {
			err = rows.Scan(&b.Model, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		} else {
			err = rows.Scan(&b.Feature, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (h *Handlers) queryUsageSeriesDay(where string, args []any, extraCol1, extraCol2 string) ([]usageBucket, error) {
	bucketExpr := "to_char(date_trunc('day', created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD')"
	selectList := bucketExpr + " AS bucket"
	groupBy := bucketExpr
	if extraCol1 != "" {
		selectList += ", " + extraCol1
		groupBy += ", " + extraCol1
	}
	if extraCol2 != "" {
		selectList += ", " + extraCol2
		groupBy += ", " + extraCol2
	}
	q := fmt.Sprintf(`SELECT %s, COALESCE(SUM(total_tokens),0), COALESCE(SUM(cached_tokens),0),
		COALESCE(SUM(total_cost),0), COUNT(*) FROM holmes_usage_events%s GROUP BY %s ORDER BY %s`,
		selectList, where, groupBy, bucketExpr)
	rows, err := h.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []usageBucket
	for rows.Next() {
		var b usageBucket
		switch {
		case extraCol1 != "" && extraCol2 != "":
			err = rows.Scan(&b.Bucket, &b.Feature, &b.Model, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		case extraCol1 != "":
			err = rows.Scan(&b.Bucket, &b.Feature, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		default:
			err = rows.Scan(&b.Bucket, &b.TotalTokens, &b.CachedTokens, &b.TotalCost, &b.CallCount)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetUsageEvents handles GET /v1/adre/usage/events.
func (h *Handlers) GetUsageEvents(w http.ResponseWriter, r *http.Request) { //nolint:gocognit
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.checkAdreEnabled(w); !ok {
		return
	}
	from, to, err := parseUsageTimeRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	where, args := usageFilterClause(from, to, r)
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 { //nolint:noinlineerr
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 { //nolint:noinlineerr
			offset = n
		}
	}
	argsWithPage := append(append([]any{}, args...), limit, offset)
	pageIdx := len(args) + 1
	q := fmt.Sprintf(`SELECT id, created_at, feature, feature_ref, adre_conversation_id, investigation_id, model,
		prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost, latency_ms, triggered_by, stream
		FROM holmes_usage_events%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, pageIdx, pageIdx+1)
	rows, err := h.db.Query(q, argsWithPage...)
	if err != nil {
		h.l.Errorf("usage events: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage events")
		return
	}
	defer func() { _ = rows.Close() }()

	type eventRow struct {
		ID                 int64    `json:"id"`
		CreatedAt          string   `json:"created_at"`
		Feature            string   `json:"feature"`
		FeatureRef         string   `json:"feature_ref"`
		AdreConversationID *int64   `json:"adre_conversation_id,omitempty"`
		InvestigationID    string   `json:"investigation_id,omitempty"`
		Model              string   `json:"model"`
		PromptTokens       *int32   `json:"prompt_tokens,omitempty"`
		CompletionTokens   *int32   `json:"completion_tokens,omitempty"`
		TotalTokens        *int32   `json:"total_tokens,omitempty"`
		CachedTokens       *int32   `json:"cached_tokens,omitempty"`
		TotalCost          *float64 `json:"total_cost,omitempty"`
		LatencyMs          *int32   `json:"latency_ms,omitempty"`
		TriggeredBy        string   `json:"triggered_by,omitempty"`
		Stream             bool     `json:"stream"`
	}

	var events []eventRow
	for rows.Next() {
		var e eventRow
		var createdAt time.Time
		var convID *int64
		var invID string
		err := rows.Scan(
			&e.ID, &createdAt, &e.Feature, &e.FeatureRef, &convID, &invID, &e.Model,
			&e.PromptTokens, &e.CompletionTokens, &e.TotalTokens, &e.CachedTokens, &e.TotalCost,
			&e.LatencyMs, &e.TriggeredBy, &e.Stream,
		)
		if err != nil {
			h.l.Errorf("usage event scan: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to load usage events")
			return
		}
		e.CreatedAt = createdAt.Format(time.RFC3339)
		e.AdreConversationID = convID
		if invID != "" {
			e.InvestigationID = invID
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil { //nolint:noinlineerr
		h.l.Errorf("usage events rows: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load usage events")
		return
	}

	if strings.EqualFold(r.URL.Query().Get("format"), "csv") || strings.Contains(r.Header.Get("Accept"), "text/csv") {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="holmes-usage.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"time", "feature", "feature_ref", "model", "total_tokens", "cached_tokens", "total_cost", "triggered_by"})
		for _, e := range events {
			tokens := ""
			if e.TotalTokens != nil {
				tokens = strconv.FormatInt(int64(*e.TotalTokens), 10)
			}
			cached := ""
			if e.CachedTokens != nil {
				cached = strconv.FormatInt(int64(*e.CachedTokens), 10)
			}
			cost := ""
			if e.TotalCost != nil {
				cost = fmt.Sprintf("%.8f", *e.TotalCost)
			}
			_ = cw.Write([]string{e.CreatedAt, e.Feature, e.FeatureRef, e.Model, tokens, cached, cost, e.TriggeredBy})
		}
		cw.Flush()
		return
	}

	if events == nil {
		events = []eventRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// GetInvestigationUsage handles GET /v1/adre/usage/investigations/:id.
func (h *Handlers) GetInvestigationUsage(w http.ResponseWriter, r *http.Request, investigationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	investigationID = strings.TrimSpace(investigationID)
	if investigationID == "" {
		writeJSONError(w, http.StatusBadRequest, "investigation id is required")
		return
	}
	events, err := QueryInvestigationUsageEvents(h.db, investigationID)
	if err != nil {
		h.l.Errorf("investigation usage: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load investigation usage")
		return
	}
	writeInvestigationUsageResponse(w, investigationID, events)
}

// WriteInvestigationUsageResponse writes the standard investigation usage JSON payload.
func WriteInvestigationUsageResponse(w http.ResponseWriter, investigationID string, events []InvestigationUsageEvent) {
	writeInvestigationUsageResponse(w, investigationID, events)
}

func writeInvestigationUsageResponse(w http.ResponseWriter, investigationID string, events []InvestigationUsageEvent) {
	type step struct {
		ID           int64    `json:"id"`
		CreatedAt    string   `json:"created_at"`
		Feature      string   `json:"feature"`
		Model        string   `json:"model"`
		TotalTokens  *int32   `json:"total_tokens,omitempty"`
		CachedTokens *int32   `json:"cached_tokens,omitempty"`
		TotalCost    *float64 `json:"total_cost,omitempty"`
		LatencyMs    *int32   `json:"latency_ms,omitempty"`
	}
	steps := make([]step, 0, len(events))
	for _, ev := range events {
		steps = append(steps, step{
			ID:           ev.ID,
			CreatedAt:    ev.CreatedAt.Format(time.RFC3339),
			Feature:      ev.Feature,
			Model:        ev.Model,
			TotalTokens:  ev.TotalTokens,
			CachedTokens: ev.CachedTokens,
			TotalCost:    ev.TotalCost,
			LatencyMs:    ev.LatencyMs,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"investigation_id": investigationID, "events": steps})
}

// ServeUsageSubroutes handles /v1/adre/usage/...
func (h *Handlers) ServeUsageSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/adre/usage/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSONError(w, http.StatusNotFound, "Not found")
		return
	}
	if after, ok := strings.CutPrefix(path, "investigations/"); ok {
		id := after
		if id == "" || strings.Contains(id, "/") {
			writeJSONError(w, http.StatusNotFound, "Not found")
			return
		}
		h.GetInvestigationUsage(w, r, id)
		return
	}
	writeJSONError(w, http.StatusNotFound, "Not found")
}
