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
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

type confidencePayload struct {
	Band      string          `json:"band"`
	Score     int             `json:"score"`
	Rationale string          `json:"rationale"`
	Evidence  []EvidenceEntry `json:"evidence"`
}

const (
	investigationRunTimeout  = 5 * time.Minute
	investigationChatTimeout = 5 * time.Minute
)

func (h *Handlers) requireHolmesURL(w http.ResponseWriter, settings *models.Settings) bool {
	if settings.GetAdreURL() == "" {
		writeJSONError(w, http.StatusBadRequest, "HolmesGPT is not configured. Set HolmesGPT URL in AI Assistant Settings.")
		return false
	}
	return true
}

// PostInvestigationChat handles POST /v1/investigations/:id/chat. Uses Holmes /api/chat for one round.
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
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
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
	if !h.requireHolmesURL(w, settings) {
		return
	}

	// Load existing messages before persisting the new user message so conversation_history does not duplicate it.
	msgs, err := models.GetInvestigationMessages(h.db, id, 20, 0) //nolint:mnd
	if err != nil {
		h.l.Errorf("GetInvestigationMessages: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load messages")
		return
	}

	// Persist user message
	userMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID: id,
		Role:            "user",
		Content:         body.Message,
	}
	err = models.CreateInvestigationMessage(h.db, userMsg)
	if err != nil {
		h.l.Errorf("CreateInvestigationMessage: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save message")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), investigationChatTimeout)
	defer cancel()

	client := adre.NewClientFromSettings(settings)
	ctxStr := buildInvestigationContext(inv)
	investigationPrompt := adre.ResolveChatSystemPrompt(settings, "investigation")
	systemWithContext := investigationPrompt + "\n\nCurrent investigation context:\n" + ctxStr

	// Build history from existing messages only (oldest first); new user message is sent as Ask.
	var history []any
	for _, m := range slices.Backward(msgs) {
		if m.Role == "tool" {
			history = append(history, map[string]any{"role": "tool", "content": m.Content, "name": m.ToolName}) //nolint:goconst
		} else {
			history = append(history, map[string]any{"role": m.Role, "content": m.Content})
		}
	}

	// HolmesGPT requires the first item in conversation_history to be role "system" when history is non-empty.
	historyForHolmes := history
	if len(historyForHolmes) > 0 {
		withSystem := make([]any, 0, len(historyForHolmes)+1)
		withSystem = append(withSystem, map[string]any{"role": "system", "content": systemWithContext})
		withSystem = append(withSystem, historyForHolmes...)
		historyForHolmes = withSystem
	}
	maxN := adre.MaxConversationMessages(settings)
	historyForHolmes = adre.TrimConversationHistory(historyForHolmes, maxN)
	historyForHolmes = adre.EnsureHolmesLeadingSystemMessage(historyForHolmes)

	req := &adre.ChatRequest{
		Ask:                    body.Message,
		ConversationHistory:    historyForHolmes,
		AdditionalSystemPrompt: systemWithContext,
		BehaviorControls:       adre.ResolveBehaviorControlsForInvestigation(settings),
		Model:                  strings.TrimSpace(settings.Adre.InvestigationModel),
		Stream:                 false,
	}
	chatStart := time.Now()
	resp, err := client.Chat(ctx, req)
	if err != nil {
		h.l.Errorf("Holmes Chat: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Chat failed: "+err.Error())
		return
	}
	lastContent := resp.Analysis

	assistantMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID: id,
		Role:            "assistant",
		Content:         lastContent,
	}
	adre.ApplyHolmesUsageToInvestigationMessage(assistantMsg, adre.HolmesFeatureInvestigationChat, req.Model, resp.Metadata)
	if err := models.CreateInvestigationMessage(h.db, assistantMsg); err != nil { //nolint:noinlineerr
		h.l.Errorf("CreateInvestigationMessage assistant: %v", err)
	}
	msgID := assistantMsg.ID
	invID := id
	_, _ = adre.RecordHolmesUsage(ctx, adre.UsageRecordInput{
		DB:                     h.db,
		Feature:                adre.HolmesFeatureInvestigationChat,
		FeatureRef:             msgID,
		InvestigationID:        invID,
		Model:                  req.Model,
		Metadata:               resp.Metadata,
		TriggeredBy:            inv.CreatedBy,
		LatencyMs:              int(time.Since(chatStart).Milliseconds()),
		InvestigationMessageID: &msgID,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"content": lastContent}) //nolint:errchkjson // response already committed
}

// startInvestigationRun validates that the investigation can run, marks it "running", records the
// run-kickoff message, and returns the loaded settings so the caller can launch the background run.
// A non-nil *httpError explains why a run cannot start. It is shared by the HTTP run endpoint and the
// programmatic auto-investigate path so both go through identical validation and state transitions.
func (h *Handlers) startInvestigationRun(id string) (*models.Settings, *httpError) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			return nil, &httpError{http.StatusNotFound, "Investigation not found"}
		}
		return nil, &httpError{http.StatusInternalServerError, "Failed to get investigation"}
	}

	switch inv.Status {
	case "running":
		return nil, &httpError{http.StatusConflict, "Investigation is already running"}
	case "completed":
		return nil, &httpError{http.StatusConflict, "Investigation has already completed — create a new investigation to re-analyze"}
	case "failed":
		return nil, &httpError{http.StatusConflict, "Investigation previously failed — create a new investigation to retry"}
	case "resolved", "archived":
		return nil, &httpError{http.StatusConflict, "Investigation is " + inv.Status + " and cannot be re-run"}
	}

	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		return nil, &httpError{http.StatusInternalServerError, "Failed to get settings"}
	}
	if settings.GetAdreURL() == "" {
		return nil, &httpError{http.StatusBadRequest, "HolmesGPT is not configured. Set HolmesGPT URL in AI Assistant Settings."}
	}

	inv.Status = "running"
	err = models.UpdateInvestigation(h.db, inv)
	if err != nil {
		h.l.Errorf("UpdateInvestigation (running): %v", err)
		return nil, &httpError{http.StatusInternalServerError, "Failed to update investigation status"}
	}

	userMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID: id,
		Role:            "user",
		Content:         "Generate the full investigation report.",
	}
	if mErr := models.CreateInvestigationMessage(h.db, userMsg); mErr != nil {
		h.l.Warnf("CreateInvestigationMessage run user: %v", mErr)
	}
	return settings, nil
}

// PostInvestigationRun handles POST /v1/investigations/:id/run.
// Sets status to "running", returns 202 immediately, and runs the investigation in a background goroutine.
func (h *Handlers) PostInvestigationRun(w http.ResponseWriter, _ *http.Request, id string) {
	settings, hErr := h.startInvestigationRun(id)
	if hErr != nil {
		writeJSONError(w, hErr.status, hErr.msg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "running"}) //nolint:errchkjson // response already committed

	go h.runInvestigationBackground(id, nil, settings) //nolint:contextcheck // background run uses a fresh detached context
}

// StartRun launches a background investigation run programmatically (e.g. from auto-investigate),
// mirroring the HTTP run endpoint without an http.ResponseWriter. It returns an error if the run
// cannot be started (already running/completed, Holmes not configured, etc.).
func (h *Handlers) StartRun(_ context.Context, id string) error {
	settings, hErr := h.startInvestigationRun(id)
	if hErr != nil {
		return errors.New(hErr.msg)
	}
	go h.runInvestigationBackground(id, nil, settings)
	return nil
}

// runInvestigationBackground executes the investigation in a background goroutine.
// It intentionally creates a fresh context.Background-derived ctx so the client
// closing the HTTP request does not abort an in-flight run.
func (h *Handlers) runInvestigationBackground(id string, _ *models.Investigation, settings *models.Settings) { //nolint:gocognit
	ctx, cancel := context.WithTimeout(context.Background(), investigationRunTimeout)
	defer cancel()

	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		h.l.Errorf("runInvestigationBackground: failed to reload investigation %s: %v", id, err)
		return
	}

	ctxStr := buildInvestigationContext(inv)
	invPrompt := adre.ResolveChatSystemPrompt(settings, "investigation")
	client := adre.NewClientFromSettings(settings)
	ask := "Generate the full investigation report for this incident.\n\nContext:\n" + ctxStr
	chatReq := &adre.ChatRequest{
		Ask:                    ask,
		AdditionalSystemPrompt: invPrompt + "\n\nCurrent investigation context:\n" + ctxStr,
		BehaviorControls:       adre.ResolveBehaviorControlsForInvestigation(settings),
		Model:                  strings.TrimSpace(settings.Adre.InvestigationModel),
		Stream:                 false,
	}
	var lastContent string
	var runErr error
	var runMetadata json.RawMessage
	runStart := time.Now()
	chatResp, err := client.Chat(ctx, chatReq)
	if err != nil {
		runErr = fmt.Errorf("holmes chat (investigation run): %w", err)
	} else {
		lastContent = chatResp.Analysis
		runMetadata = chatResp.Metadata
	}

	if runErr != nil {
		h.l.Errorf("Investigation run failed [%s]: %v", id, runErr)
		inv.Status = "failed"
		err := models.UpdateInvestigation(h.db, inv)
		if err != nil {
			h.l.Errorf("UpdateInvestigation (failed): %v", err)
		}
		errMsg := &models.InvestigationMessage{
			ID:              models.NewInvestigationID(),
			InvestigationID: id,
			Role:            "assistant",
			Content:         "Investigation failed: " + runErr.Error(),
		}
		_ = models.CreateInvestigationMessage(h.db, errMsg)
		if h.reportNotifier != nil {
			h.reportNotifier.PostInvestigationReport(ctx, inv)
		}
		return
	}

	formatStart := time.Now()
	formattedJSON, formatMetadata, formatErr := FormatInvestigationReport(ctx, client, settings, lastContent)
	if formatErr == nil {
		_, _ = adre.RecordHolmesUsage(ctx, adre.UsageRecordInput{
			DB:              h.db,
			Feature:         adre.HolmesFeatureInvestigationFormat,
			FeatureRef:      id,
			InvestigationID: id,
			Metadata:        formatMetadata,
			TriggeredBy:     inv.CreatedBy,
			LatencyMs:       int(time.Since(formatStart).Milliseconds()),
		})
	}
	if formatErr == nil { //nolint:nestif
		report, parseErr := ParseFormattedReport(formattedJSON)
		if parseErr == nil {
			preserveUserRequest(inv)
			inv.Summary = report.Summary
			inv.SummaryDetailed = report.SummaryDetailed
			inv.RootCauseSummary = report.RootCauseSummary
			inv.ResolutionSummary = report.ResolutionSummary
			cfg := map[string]any{}
			if len(inv.Config) > 0 {
				_ = json.Unmarshal(inv.Config, &cfg)
			}
			cfg["confidence"] = confidencePayload{
				Band:      report.Confidence,
				Score:     report.ConfidenceScore,
				Rationale: report.ConfidenceRationale,
				Evidence:  report.Evidence,
			}
			b, mErr := json.Marshal(cfg)
			if mErr == nil {
				inv.Config = b
			}
			err := models.DeleteInvestigationBlocksForInvestigation(h.db, id)
			if err != nil {
				h.l.Warnf("DeleteInvestigationBlocksForInvestigation: %v", err)
			}
			err = models.DeleteInvestigationTimelineEventsForInvestigation(h.db, id)
			if err != nil {
				h.l.Warnf("DeleteInvestigationTimelineEventsForInvestigation: %v", err)
			}
			for pos, sec := range report.Sections {
				blockType := sec.Type
				switch blockType {
				case BlockTypeMarkdown,
					BlockTypeFinding,
					BlockTypeRemediationSteps,
					BlockTypeQueryResult,
					BlockTypeLogsView,
					BlockTypeSinglePanel,
					BlockTypePanelGroup,
					BlockTypeImage:
				default:
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
				err := models.CreateInvestigationBlock(h.db, block)
				if err != nil {
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
				teErr := models.CreateInvestigationTimelineEvent(h.db, event)
				if teErr != nil {
					h.l.Warnf("CreateInvestigationTimelineEvent: %v", teErr)
				}
			}
		} else {
			h.l.Warnf("ParseFormattedReport: %v", parseErr)
		}
	} else {
		h.l.Warnf("FormatInvestigationReport: %v (fallback: raw report only)", formatErr)
	}

	inv.Status = "completed"
	uErr := models.UpdateInvestigation(h.db, inv)
	if uErr != nil {
		h.l.Errorf("UpdateInvestigation (completed): %v", uErr)
	}

	assistantMsg := &models.InvestigationMessage{
		ID:              models.NewInvestigationID(),
		InvestigationID: id,
		Role:            "assistant",
		Content:         lastContent,
	}
	adre.ApplyHolmesUsageToInvestigationMessage(assistantMsg, adre.HolmesFeatureInvestigationRun, chatReq.Model, runMetadata)
	_ = models.CreateInvestigationMessage(h.db, assistantMsg)
	runMsgID := assistantMsg.ID
	invID := id
	_, _ = adre.RecordHolmesUsage(ctx, adre.UsageRecordInput{
		DB:                     h.db,
		Feature:                adre.HolmesFeatureInvestigationRun,
		FeatureRef:             runMsgID,
		InvestigationID:        invID,
		Model:                  chatReq.Model,
		Metadata:               runMetadata,
		TriggeredBy:            inv.CreatedBy,
		LatencyMs:              int(time.Since(runStart).Milliseconds()),
		InvestigationMessageID: &runMsgID,
	})

	// Post the completed report into the alert's Slack thread (no-op unless this investigation was
	// scraped from a Slack alert). inv.Config still carries the slack thread ref merged above.
	if h.reportNotifier != nil {
		h.reportNotifier.PostInvestigationReport(ctx, inv)
	}
}

// alertSnapshotEntry is a single alert from Grafana Alertmanager (labels, annotations, fingerprint, etc.).
type alertSnapshotEntry struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Fingerprint  string            `json:"fingerprint"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

func buildInvestigationContext(inv *models.Investigation) string { //nolint:gocognit
	userRequest := userRequestFromInvestigation(inv)
	contextLine := "Summary: " + inv.Summary
	if userRequest != "" {
		contextLine = "User request: " + userRequest
	}
	s := fmt.Sprintf("Title: %s\nStatus: %s\nTime range: %s — %s\n%s",
		inv.Title, inv.Status,
		inv.TimeFrom.Format(time.RFC3339), inv.TimeTo.Format(time.RFC3339),
		contextLine)
	if len(inv.Config) > 0 { //nolint:nestif
		var cfg map[string]any
		err := json.Unmarshal(inv.Config, &cfg)
		if err == nil {
			if v, _ := cfg["node_name"].(string); v != "" {
				s += "\nNode: " + v
			}
			if v, _ := cfg["service_name"].(string); v != "" {
				s += "\nService: " + v
			}
			if v, _ := cfg["cluster_name"].(string); v != "" {
				s += "\nCluster: " + v
			}
			if raw, ok := cfg["alert_snapshot"].(string); ok && raw != "" {
				var alerts []alertSnapshotEntry
				err := json.Unmarshal([]byte(raw), &alerts)
				if err == nil && len(alerts) > 0 {
					s += "\n\nFull alert(s):"
					var sSb399 strings.Builder
					for i, a := range alerts {
						fmt.Fprintf(&sSb399, "\n[Alert %d]", i+1)
						if len(a.Labels) > 0 {
							pairs := make([]string, 0, len(a.Labels))
							for k, v := range a.Labels {
								pairs = append(pairs, k+"="+v)
							}
							sSb399.WriteString("\nLabels: " + strings.Join(pairs, ", "))
						}
						if len(a.Annotations) > 0 {
							pairs := make([]string, 0, len(a.Annotations))
							for k, v := range a.Annotations {
								pairs = append(pairs, k+"="+v)
							}
							sSb399.WriteString("\nAnnotations: " + strings.Join(pairs, ", "))
						}
						if a.Fingerprint != "" {
							sSb399.WriteString("\nFingerprint: " + a.Fingerprint)
						}
					}
					s += sSb399.String()
				} else {
					var single alertSnapshotEntry
					err := json.Unmarshal([]byte(raw), &single)
					if err == nil {
						s += "\n\nFull alert(s):\n[Alert 1]"
						if len(single.Labels) > 0 {
							pairs := make([]string, 0, len(single.Labels))
							for k, v := range single.Labels {
								pairs = append(pairs, k+"="+v)
							}
							s += "\nLabels: " + strings.Join(pairs, ", ")
						}
						if len(single.Annotations) > 0 {
							pairs := make([]string, 0, len(single.Annotations))
							for k, v := range single.Annotations {
								pairs = append(pairs, k+"="+v)
							}
							s += "\nAnnotations: " + strings.Join(pairs, ", ")
						}
						if single.Fingerprint != "" {
							s += "\nFingerprint: " + single.Fingerprint
						}
					}
				}
			}
		}
	}
	return s
}
