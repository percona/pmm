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

package investigations

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

const chatMaxTurns = 5

// PostInvestigationChat handles POST /v1/investigations/:id/chat. Runs one round of orchestrator (single-round Q&A).
func (h *Handlers) PostInvestigationChat(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	if settings.Adre.OrchestratorLLMURL == "" {
		writeJSONError(w, http.StatusBadRequest, "Orchestrator LLM is not configured. Set Orchestrator URL in Settings.")
		return
	}
	provider := orchestrator.NewOllamaProvider(settings.Adre.OrchestratorLLMURL, settings.Adre.OrchestratorLLMModel)
	tools := orchestrator.DefaultToolRegistry(false) // no holmes_investigate until phase3

	// Load last messages (newest first from DB; we'll reverse for context)
	msgs, err := models.GetInvestigationMessages(h.db, id, 20, 0)
	if err != nil {
		h.l.Errorf("GetInvestigationMessages: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load messages")
		return
	}
	// Build conversation: system + context, then history (oldest first), then user message
	ctxStr := buildInvestigationContext(inv)
	systemContent := orchestrator.DefaultSystemPrompt + "\n\nCurrent investigation context:\n" + ctxStr
	messages := []orchestrator.Message{{Role: "system", Content: systemContent}}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		role := m.Role
		if role == "tool" {
			messages = append(messages, orchestrator.Message{Role: "tool", Content: string(m.Content), Name: m.ToolName})
		} else {
			messages = append(messages, orchestrator.Message{Role: role, Content: m.Content})
		}
	}
	messages = append(messages, orchestrator.Message{Role: "user", Content: body.Message})

	// Persist user message
	userMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID: id,
		Role:            "user",
		Content:         body.Message,
	}
	if err := models.CreateInvestigationMessage(h.db, userMsg); err != nil {
		h.l.Errorf("CreateInvestigationMessage: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save message")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Orchestration loop: call LLM, execute tools, append results, repeat until no tool calls or max turns
	turn := 0
	var lastContent string
	for turn < chatMaxTurns {
		turn++
		result, err := provider.Complete(ctx, messages, tools)
		if err != nil {
			h.l.Errorf("Orchestrator Complete: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Orchestrator failed: "+err.Error())
			return
		}
		lastContent = result.Content

		// Persist assistant message
		assistantMsg := &models.InvestigationMessage{
			ID:              models.NewInvestigationID(),
			InvestigationID: id,
			Role:            "assistant",
			Content:         result.Content,
		}
		if err := models.CreateInvestigationMessage(h.db, assistantMsg); err != nil {
			h.l.Warnf("CreateInvestigationMessage assistant: %v", err)
		}
		messages = append(messages, orchestrator.Message{Role: "assistant", Content: result.Content})

		if len(result.ToolCalls) == 0 {
			break
		}

		// Execute tool calls and append tool messages
		for _, tc := range result.ToolCalls {
			toolResult := executeTool(h.db, h.l, id, inv, tc)
			messages = append(messages, orchestrator.Message{Role: "tool", Content: toolResult, Name: tc.Name})
			toolMsg := &models.InvestigationMessage{
				ID:              models.NewInvestigationID(),
				InvestigationID: id,
				Role:            "tool",
				Content:         toolResult,
				ToolName:        tc.Name,
			}
			if err := models.CreateInvestigationMessage(h.db, toolMsg); err != nil {
				h.l.Warnf("CreateInvestigationMessage tool: %v", err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"content": lastContent})
}

// PostInvestigationRun handles POST /v1/investigations/:id/run. Runs the multi-turn orchestration loop to build the report.
func (h *Handlers) PostInvestigationRun(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}

	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	if settings.Adre.OrchestratorLLMURL == "" {
		writeJSONError(w, http.StatusBadRequest, "Orchestrator LLM is not configured. Set Orchestrator URL in Settings.")
		return
	}
	provider := orchestrator.NewOllamaProvider(settings.Adre.OrchestratorLLMURL, settings.Adre.OrchestratorLLMModel)
	tools := orchestrator.DefaultToolRegistry(false)

	ctxStr := buildInvestigationContext(inv)
	systemContent := orchestrator.RunReportSystemPrompt + "\n\nCurrent investigation context:\n" + ctxStr
	messages := []orchestrator.Message{
		{Role: "system", Content: systemContent},
		{Role: "user", Content: "Generate the full investigation report based on the context above. Use get_investigation_context first, then append_block to add summary and finding blocks."},
	}

	// Persist the synthetic user message
	userMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID:  id,
		Role:            "user",
		Content:         "Generate the full investigation report.",
	}
	if err := models.CreateInvestigationMessage(h.db, userMsg); err != nil {
		h.l.Warnf("CreateInvestigationMessage run user: %v", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	turn := 0
	var lastContent string
	for turn < chatMaxTurns {
		turn++
		result, err := provider.Complete(ctx, messages, tools)
		if err != nil {
			h.l.Errorf("Orchestrator Complete run: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Orchestrator failed: "+err.Error())
			return
		}
		lastContent = result.Content

		assistantMsg := &models.InvestigationMessage{
			ID:              models.NewInvestigationID(),
			InvestigationID:  id,
			Role:            "assistant",
			Content:         result.Content,
		}
		_ = models.CreateInvestigationMessage(h.db, assistantMsg)
		messages = append(messages, orchestrator.Message{Role: "assistant", Content: result.Content})

		if len(result.ToolCalls) == 0 {
			break
		}

		for _, tc := range result.ToolCalls {
			toolResult := executeTool(h.db, h.l, id, inv, tc)
			messages = append(messages, orchestrator.Message{Role: "tool", Content: toolResult, Name: tc.Name})
			toolMsg := &models.InvestigationMessage{
				ID:              models.NewInvestigationID(),
				InvestigationID: id,
				Role:            "tool",
				Content:         toolResult,
				ToolName:        tc.Name,
			}
			_ = models.CreateInvestigationMessage(h.db, toolMsg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"content": lastContent})
}

func buildInvestigationContext(inv *models.Investigation) string {
	return fmt.Sprintf("Title: %s\nStatus: %s\nTime range: %s — %s\nSummary: %s",
		inv.Title, inv.Status,
		inv.TimeFrom.Format(time.RFC3339), inv.TimeTo.Format(time.RFC3339),
		inv.Summary)
}

func executeTool(db *reform.DB, l *logrus.Entry, investigationID string, inv *models.Investigation, tc orchestrator.ToolCall) string {
	switch tc.Name {
	case "get_investigation_context":
		ctx := buildInvestigationContext(inv)
		blocks, _ := models.GetInvestigationBlocks(db, investigationID)
		if len(blocks) > 0 {
			ctx += "\nBlocks: " + fmt.Sprintf("%d", len(blocks))
		}
		return ctx
	case "append_block":
		var args struct {
			Type     string                 `json:"type"`
			Title    string                 `json:"title"`
			Position int                    `json:"position"`
			DataJSON map[string]interface{} `json:"data_json"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			l.Warnf("append_block unmarshal: %v", err)
			return `{"error": "invalid arguments"}`
		}
		if args.Type == "" {
			return `{"error": "type is required"}`
		}
		dataJSON, _ := json.Marshal(args.DataJSON)
		block := &models.InvestigationBlock{
			ID:             models.NewInvestigationID(),
			InvestigationID: investigationID,
			Type:           args.Type,
			Title:          args.Title,
			Position:       args.Position,
			DataJSON:       dataJSON,
		}
		if err := models.CreateInvestigationBlock(db, block); err != nil {
			l.Warnf("CreateInvestigationBlock: %v", err)
			return `{"error": "failed to create block"}`
		}
		return fmt.Sprintf(`{"ok": true, "block_id": "%s"}`, block.ID)
	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Name)
	}
}
