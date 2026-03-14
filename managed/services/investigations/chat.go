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
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

const investigationRunTimeout = 5 * time.Minute
const investigationChatTimeout = 2 * time.Minute

func (h *Handlers) requireHolmesURL(w http.ResponseWriter, settings *models.Settings) bool {
	if settings.GetAdreURL() == "" {
		writeJSONError(w, http.StatusBadRequest, "HolmesGPT is not configured. Set HolmesGPT URL in AI Assistant Settings.")
		return false
	}
	return true
}

func (h *Handlers) requireValidChatBackend(w http.ResponseWriter, settings *models.Settings) bool {
	cb := settings.Adre.ChatBackend
	if cb == "" {
		cb = "holmesgpt"
	}
	if cb != "holmesgpt" && cb != "holmes_agent" {
		writeJSONError(w, http.StatusBadRequest, "Chat backend must be PMM Agent or Holmes Agent. Configure it in AI Assistant Settings.")
		return false
	}
	return true
}

// PostInvestigationChat handles POST /v1/investigations/:id/chat. Uses Holmes (PMM Agent or Holmes Agent) for one round.
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
	if !h.requireHolmesURL(w, settings) || !h.requireValidChatBackend(w, settings) {
		return
	}

	// Load existing messages before persisting the new user message so conversation_history does not duplicate it.
	msgs, err := models.GetInvestigationMessages(h.db, id, 20, 0)
	if err != nil {
		h.l.Errorf("GetInvestigationMessages: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load messages")
		return
	}

	// Persist user message
	userMsg := &models.InvestigationMessage{
		ID:             models.NewInvestigationID(),
		InvestigationID: id,
		Role:           "user",
		Content:        body.Message,
	}
	if err := models.CreateInvestigationMessage(h.db, userMsg); err != nil {
		h.l.Errorf("CreateInvestigationMessage: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save message")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), investigationChatTimeout)
	defer cancel()

	client := adre.NewClient(settings.GetAdreURL())
	ctxStr := buildInvestigationContext(inv)
	investigationPrompt := settings.Adre.InvestigationPrompt
	if investigationPrompt == "" {
		investigationPrompt = adre.DefaultInvestigationPrompt
	}
	systemWithContext := investigationPrompt + "\n\nCurrent investigation context:\n" + ctxStr

	// Build history from existing messages only (oldest first); new user message is sent as Ask.
	var history []interface{}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == "tool" {
			history = append(history, map[string]interface{}{"role": "tool", "content": m.Content, "name": m.ToolName})
		} else {
			history = append(history, map[string]interface{}{"role": m.Role, "content": m.Content})
		}
	}

	// HolmesGPT requires the first item in conversation_history to be role "system" when history is non-empty.
	historyForHolmes := history
	if len(historyForHolmes) > 0 {
		withSystem := make([]interface{}, 0, len(historyForHolmes)+1)
		withSystem = append(withSystem, map[string]interface{}{"role": "system", "content": systemWithContext})
		withSystem = append(withSystem, historyForHolmes...)
		historyForHolmes = withSystem
	}

	var lastContent string
	cb := settings.Adre.ChatBackend
	if cb == "" {
		cb = "holmesgpt"
	}
	switch cb {
	case "holmesgpt":
		req := &adre.ChatRequest{
			Ask:                    body.Message,
			ConversationHistory:    historyForHolmes,
			AdditionalSystemPrompt: systemWithContext,
			Stream:                 false,
		}
		resp, err := client.Chat(ctx, req)
		if err != nil {
			h.l.Errorf("Holmes Chat: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Chat failed: "+err.Error())
			return
		}
		lastContent = resp.Analysis
	case "holmes_agent":
		historyWithToolID := make([]interface{}, 0, len(history))
		for _, v := range history {
			m, _ := v.(map[string]interface{})
			if m != nil && m["role"] == "tool" {
				// Include tool_call_id for tool messages (PMM Agent sync may expect it).
				withID := make(map[string]interface{}, len(m)+1)
				for k, val := range m {
					withID[k] = val
				}
				if _, has := withID["tool_call_id"]; !has {
					withID["tool_call_id"] = ""
				}
				historyWithToolID = append(historyWithToolID, withID)
			} else {
				historyWithToolID = append(historyWithToolID, v)
			}
		}
		content, err := adre.RunPMMAgentChatSync(ctx, h.db, h.l, settings, body.Message, historyWithToolID, investigationChatTimeout)
		if err != nil {
			h.l.Errorf("PMM Agent Chat: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Chat failed: "+err.Error())
			return
		}
		lastContent = content
	default:
		writeJSONError(w, http.StatusBadRequest, "Chat backend must be PMM Agent or Holmes Agent.")
		return
	}

	assistantMsg := &models.InvestigationMessage{
		ID:             models.NewInvestigationID(),
		InvestigationID: id,
		Role:           "assistant",
		Content:        lastContent,
	}
	_ = models.CreateInvestigationMessage(h.db, assistantMsg)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"content": lastContent})
}

// PostInvestigationRun handles POST /v1/investigations/:id/run. Uses Holmes (Investigate or PMM Agent sync) to generate the report.
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
	if !h.requireHolmesURL(w, settings) || !h.requireValidChatBackend(w, settings) {
		return
	}

	ctxStr := buildInvestigationContext(inv)
	userMsg := &models.InvestigationMessage{
		ID:             models.NewInvestigationID(),
		InvestigationID: id,
		Role:           "user",
		Content:        "Generate the full investigation report.",
	}
	if err := models.CreateInvestigationMessage(h.db, userMsg); err != nil {
		h.l.Warnf("CreateInvestigationMessage run user: %v", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), investigationRunTimeout)
	defer cancel()

	adreURL := settings.GetAdreURL()
	var lastContent string
	runBackend := settings.Adre.ChatBackend
	if runBackend == "" {
		runBackend = "holmesgpt"
	}
	switch runBackend {
	case "holmesgpt":
		invPrompt := settings.Adre.InvestigationPrompt
		if invPrompt == "" {
			invPrompt = adre.DefaultInvestigationPrompt
		}
		client := adre.NewClient(adreURL)
		// Prefer /api/investigate when available; many HolmesGPT instances only have /api/chat.
		invReq := &adre.InvestigateRequest{
			Source:                 "pmm-investigation",
			Title:                  inv.Title,
			Description:            "Generate the full investigation report based on the context below.",
			Subject:                map[string]interface{}{}, // required by HolmesGPT InvestigateRequest
			Context:                map[string]interface{}{"investigation_id": id, "time_from": inv.TimeFrom.Format(time.RFC3339), "time_to": inv.TimeTo.Format(time.RFC3339), "summary": inv.Summary, "context_text": ctxStr},
			AdditionalSystemPrompt: invPrompt,
		}
		resp, err := client.Investigate(ctx, invReq)
		if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found")) {
			// Fallback: use /api/chat (supported by all HolmesGPT deployments).
			ask := "Generate the full investigation report for this incident.\n\nContext:\n" + ctxStr
			chatReq := &adre.ChatRequest{
				Ask:                    ask,
				ConversationHistory:    nil,
				AdditionalSystemPrompt: invPrompt + "\n\nCurrent investigation context:\n" + ctxStr,
				Stream:                 false,
			}
			var chatResp *adre.ChatResponse
			chatResp, err = client.Chat(ctx, chatReq)
			if err != nil {
				h.l.Errorf("Holmes Chat (investigation fallback): %v", err)
				writeJSONError(w, http.StatusBadGateway, "Run failed: "+err.Error())
				return
			}
			lastContent = chatResp.Analysis
		} else if err != nil {
			h.l.Errorf("Holmes Investigate: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Run failed: "+err.Error())
			return
		} else {
			// Use only Analysis; it already contains the full report. Appending Sections would duplicate content.
			lastContent = resp.Analysis
		}
	case "holmes_agent":
		ask := "Run the full investigation now. Use ask_holmes to gather logs, metrics, and alerts for this incident, then call generate_investigation_report with the gathered context. Do not ask for confirmation—execute the investigation immediately.\n\nContext:\n" + ctxStr
		content, err := adre.RunPMMAgentChatSync(ctx, nil, h.l, settings, ask, nil, investigationRunTimeout)
		if err != nil {
			h.l.Errorf("PMM Agent run: %v", err)
			writeJSONError(w, http.StatusBadGateway, "Run failed: "+err.Error())
			return
		}
		lastContent = content
	default:
		writeJSONError(w, http.StatusBadRequest, "Chat backend must be PMM Agent or Holmes Agent.")
		return
	}

	// Pass 2: format raw report into structured JSON; update investigation and create blocks on success
	client := adre.NewClient(adreURL)
	formattedJSON, err := FormatInvestigationReport(ctx, client, lastContent)
	if err == nil {
		report, parseErr := ParseFormattedReport(formattedJSON)
		if parseErr == nil {
			inv.Summary = report.Summary
			inv.SummaryDetailed = report.SummaryDetailed
			inv.RootCauseSummary = report.RootCauseSummary
			inv.ResolutionSummary = report.ResolutionSummary
			if err := models.UpdateInvestigation(h.db, inv); err != nil {
				h.l.Warnf("UpdateInvestigation after format: %v", err)
			}
			if err := models.DeleteInvestigationBlocksForInvestigation(h.db, id); err != nil {
				h.l.Warnf("DeleteInvestigationBlocksForInvestigation: %v", err)
			}
			if err := models.DeleteInvestigationTimelineEventsForInvestigation(h.db, id); err != nil {
				h.l.Warnf("DeleteInvestigationTimelineEventsForInvestigation: %v", err)
			}
			for pos, sec := range report.Sections {
				blockType := sec.Type
				if blockType != BlockTypeMarkdown && blockType != BlockTypeFinding && blockType != BlockTypeRemediationSteps {
					blockType = BlockTypeMarkdown
				}
				dataJSON := buildBlockDataJSON(blockType, sec.Content)
				block := &models.InvestigationBlock{
					ID:              models.NewInvestigationID(),
					InvestigationID: id,
					Type:            blockType,
					Title:           sec.Title,
					Position:        pos,
					DataJSON:        dataJSON,
				}
				if err := models.CreateInvestigationBlock(h.db, block); err != nil {
					h.l.Warnf("CreateInvestigationBlock: %v", err)
				}
			}
			for _, te := range report.TimelineEvents {
				if te.EventTime == "" || te.Title == "" {
					continue
				}
				eventTime, err := time.Parse(time.RFC3339, te.EventTime)
				if err != nil {
					h.l.Warnf("Parse timeline event_time %q: %v", te.EventTime, err)
					continue
				}
				event := &models.InvestigationTimelineEvent{
					ID:              models.NewInvestigationID(),
					InvestigationID: id,
					EventTime:       eventTime,
					Type:            te.Type,
					Title:           te.Title,
					Description:     te.Description,
					Source:          "format",
				}
				if err := models.CreateInvestigationTimelineEvent(h.db, event); err != nil {
					h.l.Warnf("CreateInvestigationTimelineEvent: %v", err)
				}
			}
		} else {
			h.l.Warnf("ParseFormattedReport: %v", parseErr)
		}
	} else {
		h.l.Warnf("FormatInvestigationReport: %v (fallback: raw report only)", err)
	}

	assistantMsg := &models.InvestigationMessage{
		ID:             models.NewInvestigationID(),
		InvestigationID: id,
		Role:           "assistant",
		Content:        lastContent,
	}
	_ = models.CreateInvestigationMessage(h.db, assistantMsg)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"content": lastContent})
}

func buildInvestigationContext(inv *models.Investigation) string {
	return fmt.Sprintf("Title: %s\nStatus: %s\nTime range: %s — %s\nSummary: %s",
		inv.Title, inv.Status,
		inv.TimeFrom.Format(time.RFC3339), inv.TimeTo.Format(time.RFC3339),
		inv.Summary)
}
