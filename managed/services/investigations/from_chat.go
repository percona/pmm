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

package investigations

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

const (
	chatSourceType = "chat"
	// Transcript blocks follow the sections the formatter adds at positions from 0.
	chatBlockPosition = 1000
	// Bounds what one formatting call reads from a long chat.
	chatTranscriptMaxBytes = 60_000
	chatTitleMaxRunes      = 100
	chatSummaryMaxRunes    = 600
)

// The Assistant's own bookkeeping, not evidence.
var chatInternalTools = []string{"TodoWrite", "fetch_skill"}

// chatTool is one tool call behind an answer.
type chatTool struct {
	Name   string
	Params map[string]any
	Failed bool
}

// chatTurn is one question of an Assistant chat with its answer and the tool calls behind it.
type chatTurn struct {
	Question string
	Answer   string
	Tools    []chatTool
}

func toolFromRow(m models.AdreMessage) (chatTool, bool) {
	var payload struct {
		ToolName string `json:"tool_name"`
		Result   struct {
			Status string         `json:"status"`
			Params map[string]any `json:"params"`
		} `json:"result"`
	}
	_ = json.Unmarshal(m.ToolResultJSON, &payload)
	name := m.ToolName
	if name == "" {
		name = payload.ToolName
	}
	if name == "" || slices.Contains(chatInternalTools, name) {
		return chatTool{}, false
	}
	return chatTool{Name: name, Params: payload.Result.Params, Failed: payload.Result.Status == "error"}, true
}

// chatTurns groups a chat's rows into turns; tool rows join the turn they were saved in, before or after its answer.
func chatTurns(msgs []models.AdreMessage) []chatTurn {
	var turns []chatTurn
	current := func() *chatTurn {
		if len(turns) == 0 {
			turns = append(turns, chatTurn{})
		}
		return &turns[len(turns)-1]
	}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			turns = append(turns, chatTurn{Question: strings.TrimSpace(m.Content)})
		case "assistant":
			t := current()
			t.Answer = strings.TrimSpace(strings.Join(slices.DeleteFunc([]string{t.Answer, m.Content}, func(s string) bool {
				return strings.TrimSpace(s) == ""
			}), "\n\n"))
		case "tool":
			tool, ok := toolFromRow(m)
			if ok {
				t := current()
				t.Tools = append(t.Tools, tool)
			}
		}
	}
	return turns
}

func paramString(params map[string]any, key string) string {
	switch v := params[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// evidence is how a tool call reads in an investigation: its name, then the exact query or its parameters.
func (c chatTool) evidence() string {
	var b strings.Builder
	b.WriteString("`" + c.Name + "`")
	if d := paramString(c.Params, "description"); d != "" {
		b.WriteString(" — " + d)
	}
	if c.Failed {
		b.WriteString(" (failed)")
	}
	query := paramString(c.Params, "query")
	switch {
	case c.Name == "pmm_clickhouse_query" && query != "":
		b.WriteString("\n\n```sql\n" + query + "\n```")
	case strings.HasPrefix(c.Name, "execute_prometheus") || c.Name == "pmm_metrics_snapshot":
		b.WriteString("\n\n```promql\n" + query + "\n```")
	default:
		shown := map[string]any{}
		for k, v := range c.Params {
			if k != "description" && v != nil && v != "" {
				shown[k] = v
			}
		}
		if len(shown) > 0 {
			params, _ := json.MarshalIndent(shown, "", "  ") //nolint:errchkjson // decoded from JSON, so it encodes
			b.WriteString("\n\n```json\n" + string(params) + "\n```")
		}
	}
	return b.String()
}

// markdown is the answer followed by the evidence behind it.
func (t chatTurn) markdown() string {
	if len(t.Tools) == 0 {
		return t.Answer
	}
	parts := []string{t.Answer, "#### Evidence"}
	for _, tool := range t.Tools {
		parts = append(parts, tool.evidence())
	}
	return strings.Join(parts, "\n\n")
}

func truncateRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n-1]) + "…"
}

// lead is an answer's first plain paragraph, skipping headings, lists, tables, quotes and code.
func lead(answer string) string {
	for p := range strings.SplitSeq(answer, "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" || strings.ContainsAny(p[:1], "#-*|>`0123456789") {
			continue
		}
		return truncateRunes(strings.Join(strings.Fields(p), " "), chatSummaryMaxRunes)
	}
	return ""
}

// toolWindow is the time range the chat's tools looked at.
func toolWindow(turns []chatTurn) (time.Time, time.Time) {
	var from, to time.Time
	for _, t := range turns {
		for _, tool := range t.Tools {
			for _, key := range []string{"start", "end", "from", "to", "time_from", "time_to"} {
				ts, err := time.Parse(time.RFC3339, paramString(tool.Params, key))
				if err != nil {
					continue
				}
				if from.IsZero() || ts.Before(from) {
					from = ts
				}
				if ts.After(to) {
					to = ts
				}
			}
		}
	}
	return from, to
}

// chatInvestigation is the investigation a chat becomes, before its blocks.
func chatInvestigation(conv *models.AdreConversation, msgs []models.AdreMessage, turns []chatTurn, login string) *models.Investigation {
	var question, summary, longest string
	for _, t := range turns {
		if question == "" {
			question = t.Question
		}
		if len(t.Answer) > len(longest) {
			longest = t.Answer
		}
	}
	summary = lead(longest)
	title := strings.TrimSpace(conv.Title)
	if title == "" {
		title = question
	}
	from, to := toolWindow(turns)
	if from.IsZero() || !to.After(from) {
		from, to = msgs[0].CreatedAt.Add(-time.Hour), msgs[len(msgs)-1].CreatedAt
	}
	cfg, _ := json.Marshal(map[string]string{ //nolint:errchkjson // a map of strings always encodes
		configKeyUserRequest: question,
		"conversation_id":    strconv.FormatInt(conv.ID, 10),
	})
	return &models.Investigation{
		ID:         models.NewInvestigationID(),
		Title:      "Chat: " + truncateRunes(strings.Join(strings.Fields(title), " "), chatTitleMaxRunes),
		Status:     "open",
		CreatedBy:  login,
		TimeFrom:   from.UTC(),
		TimeTo:     to.UTC(),
		Summary:    summary,
		SourceType: chatSourceType,
		SourceRef:  strconv.FormatInt(conv.ID, 10),
		Config:     cfg,
	}
}

// chatTranscript is the chat for the report formatter; a long one keeps its latest turns.
func chatTranscript(turns []chatTurn, maxBytes int) string {
	parts := make([]string, 0, len(turns))
	for _, t := range turns {
		parts = append(parts, "## Question: "+t.Question+"\n\n"+t.markdown())
	}
	s := strings.Join(parts, "\n\n")
	if len(s) <= maxBytes {
		return s
	}
	cut := []rune(s[len(s)-maxBytes:])
	// Drop a rune split by the byte cut.
	return string(cut[1:])
}

// InvestigationFromChat turns an Assistant chat into an investigation with its answers, evidence and messages,
// then formats it into a report in the background; a chat already turned into one returns that one.
func (h *Handlers) InvestigationFromChat(
	ctx context.Context, conv *models.AdreConversation, msgs []models.AdreMessage, login string,
) (string, bool, error) {
	existing, err := models.FindInvestigationBySource(h.db, chatSourceType, strconv.FormatInt(conv.ID, 10))
	if err != nil {
		return "", false, fmt.Errorf("find the chat's investigation: %w", err)
	}
	if existing != nil {
		return existing.ID, true, nil
	}
	turns := chatTurns(msgs)
	inv := chatInvestigation(conv, msgs, turns, login)
	settings, err := models.GetSettings(h.db)
	if err != nil {
		return "", false, fmt.Errorf("get settings: %w", err)
	}
	format := settings.GetAdreURL() != ""
	if format {
		inv.Status = "running"
	}
	err = models.CreateInvestigation(h.db, inv)
	if err != nil {
		return "", false, fmt.Errorf("create investigation: %w", err)
	}
	l := h.l.WithField("investigation_id", inv.ID)
	for i, t := range turns {
		data, _ := json.Marshal(map[string]string{"content": t.markdown()}) //nolint:errchkjson // a map of strings always encodes
		title := truncateRunes(t.Question, chatTitleMaxRunes)
		if title == "" {
			title = "Answer"
		}
		err := models.CreateInvestigationBlock(h.db, &models.InvestigationBlock{
			ID:              models.NewInvestigationID(),
			InvestigationID: inv.ID,
			Type:            BlockTypeMarkdown,
			Title:           title,
			Position:        chatBlockPosition + i,
			DataJSON:        data,
			CreatedBy:       login,
		})
		if err != nil {
			l.Warnf("CreateInvestigationBlock: %v", err)
		}
	}
	// The chat carries on in the investigation's own chat.
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		err := models.CreateInvestigationMessage(h.db, &models.InvestigationMessage{
			ID:              models.NewInvestigationID(),
			InvestigationID: inv.ID,
			Role:            m.Role,
			Content:         m.Content,
			Model:           m.Model,
			CreatedAt:       m.CreatedAt,
		})
		if err != nil {
			l.Warnf("CreateInvestigationMessage: %v", err)
		}
	}
	if format {
		// Formatting outlives the request.
		go h.formatChatInvestigation(context.WithoutCancel(ctx), inv.ID, chatTranscript(turns, chatTranscriptMaxBytes), settings)
	}
	return inv.ID, false, nil
}

// formatChatInvestigation fills a chat's investigation with a formatted report: summary, root cause,
// resolution, sections and timeline. Without one the investigation stays open with the chat alone.
func (h *Handlers) formatChatInvestigation(ctx context.Context, id, transcript string, settings *models.Settings) {
	ctx, cancel := context.WithTimeout(ctx, investigationRunTimeout)
	defer cancel()
	l := h.l.WithField("investigation_id", id)
	client := adre.NewClientFromSettings(settings, adre.ResolveHolmesAPIKey(h.db, h.l))
	start := time.Now()
	formatted, metadata, err := FormatInvestigationReport(ctx, client, settings, transcript)
	inv, getErr := models.GetInvestigationByID(h.db, id)
	if getErr != nil || inv == nil {
		l.Errorf("Reload investigation: %v", getErr)
		return
	}
	var report *FormattedReport
	if err == nil {
		_, _ = adre.RecordHolmesUsage(ctx, adre.UsageRecordInput{
			DB:              h.db,
			Feature:         adre.HolmesFeatureInvestigationFormat,
			FeatureRef:      id,
			InvestigationID: id,
			Metadata:        metadata,
			TriggeredBy:     inv.CreatedBy,
			LatencyMs:       int(time.Since(start).Milliseconds()),
		})
		report, err = ParseFormattedReport(formatted)
	}
	inv.Status = "open"
	if err != nil {
		l.Warnf("Format chat investigation: %v", err)
	} else {
		h.applyFormattedReport(inv, report, false)
		inv.Status = "completed"
	}
	err = models.UpdateInvestigation(h.db, inv)
	if err != nil {
		l.Errorf("UpdateInvestigation: %v", err)
	}
}
