// Copyright (C) 2025 Percona LLC
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

package adre

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/investigations/orchestrator"
)

const generalChatMaxTurns = 5

// RunGeneralChatStream runs the orchestrator-based general chat (no investigation), emits SSE in the same format as HolmesGPT chat so the frontend works unchanged.
func RunGeneralChatStream(
	w http.ResponseWriter,
	r *http.Request,
	db reform.DBTX,
	l *logrus.Entry,
	settings *models.Settings,
	ask string,
	conversationHistory []interface{},
	streamTimeout time.Duration,
) {
	ctx, cancel := context.WithTimeout(r.Context(), streamTimeout)
	defer cancel()

	messages := buildGeneralMessages(conversationHistory, ask)
	provider := orchestrator.NewOllamaProvider(settings.Adre.OrchestratorLLMURL, settings.Adre.OrchestratorLLMModel)
	includeHolmes := settings.IsAdreEnabled() && settings.GetAdreURL() != ""
	tools := orchestrator.GeneralToolRegistry(includeHolmes)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	writeSSE := func(event, data string) {
		if event != "" {
			fmt.Fprintf(w, "event: %s\n", event)
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	turn := 0
	for turn < generalChatMaxTurns {
		turn++
		result, err := provider.Complete(ctx, messages, tools)
		if err != nil {
			l.Errorf("Orchestrator general chat Complete: %v", err)
			errMsg := "Orchestrator failed: " + err.Error()
			contentJSON, _ := json.Marshal(map[string]string{"content": errMsg})
			writeSSE("", string(contentJSON))
			return
		}

		if result.Content != "" {
			contentJSON, _ := json.Marshal(map[string]string{"content": result.Content})
			writeSSE("", string(contentJSON))
		}

		if len(result.ToolCalls) == 0 {
			return
		}

		for _, tc := range result.ToolCalls {
			startPayload, _ := json.Marshal(map[string]string{
				"id":          tc.ID,
				"tool_name":   tc.Name,
				"description": tc.Name,
			})
			writeSSE("start_tool_calling", string(startPayload))

			toolResult := executeGeneralTool(ctx, db, l, tc, settings.GetAdreURL())
			resultPayload, _ := json.Marshal(map[string]interface{}{
				"tool_call_id": tc.ID,
				"name":         tc.Name,
				"result":       map[string]string{"data": toolResult},
			})
			writeSSE("tool_calling_result", string(resultPayload))

			messages = append(messages, orchestrator.Message{Role: "tool", Content: toolResult, Name: tc.Name})
		}
		messages = append(messages, orchestrator.Message{Role: "assistant", Content: result.Content})
	}
}

func buildGeneralMessages(history []interface{}, ask string) []orchestrator.Message {
	messages := []orchestrator.Message{{Role: "system", Content: orchestrator.GeneralSystemPrompt}}
	for _, h := range history {
		m, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		if role == "" || content == "" {
			continue
		}
		if role == "user" || role == "assistant" {
			messages = append(messages, orchestrator.Message{Role: role, Content: content})
		}
	}
	// Frontend often sends the current user message in both conversation_history (last item) and ask; avoid duplicate.
	if ask != "" {
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			if last.Role == "user" && last.Content == ask {
				return messages
			}
		}
		messages = append(messages, orchestrator.Message{Role: "user", Content: ask})
	}
	return messages
}

func executeGeneralTool(ctx context.Context, db reform.DBTX, l *logrus.Entry, tc orchestrator.ToolCall, adreURL string) string {
	switch tc.Name {
	case "create_investigation":
		var args struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			SourceType  string `json:"source_type"`
			SourceRef   string `json:"source_ref"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			l.Warnf("create_investigation unmarshal: %v", err)
			return `{"error": "invalid arguments"}`
		}
		if args.Title == "" {
			return `{"error": "title is required"}`
		}
		now := time.Now().UTC()
		inv := &models.Investigation{
			ID:         models.NewInvestigationID(),
			Title:      args.Title,
			Status:     "open",
			TimeFrom:   now,
			TimeTo:     now,
			Summary:    args.Description,
			SourceType: args.SourceType,
			SourceRef:  args.SourceRef,
			CreatedBy:  "orchestrator",
		}
		if inv.SourceType == "" {
			inv.SourceType = "manual"
		}
		dbPtr, ok := db.(*reform.DB)
		if !ok {
			l.Warn("create_investigation: db is not *reform.DB")
			return `{"error": "database error"}`
		}
		if err := models.CreateInvestigation(dbPtr, inv); err != nil {
			l.Warnf("CreateInvestigation from orchestrator: %v", err)
			return `{"error": "failed to create investigation"}`
		}
		urlPath := "/investigations/" + inv.ID
		outBytes, _ := json.Marshal(map[string]string{"id": inv.ID, "url": urlPath})
		return string(outBytes)
	case "holmes_investigate":
		if adreURL == "" {
			return `{"error": "HolmesGPT is not configured"}`
		}
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			l.Warnf("holmes_investigate unmarshal: %v", err)
			return `{"error": "invalid arguments"}`
		}
		question := args.Question
		if question == "" {
			question = "Analyze and provide findings."
		}
		client := NewClient(adreURL)
		req := &InvestigateRequest{
			Source:      "pmm-chat",
			Title:       "General investigation",
			Description: question,
		}
		resp, err := client.Investigate(ctx, req)
		if err != nil {
			l.Warnf("HolmesGPT Investigate from general chat: %v", err)
			b, _ := json.Marshal(map[string]string{"error": "HolmesGPT request failed: " + err.Error()})
			return string(b)
		}
		out := resp.Analysis
		if len(resp.Sections) > 0 {
			for k, v := range resp.Sections {
				out += "\n\n## " + k + "\n" + v
			}
		}
		return out
	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Name)
	}
}

