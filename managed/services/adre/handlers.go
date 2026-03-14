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
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// GrafanaAlertsFetcher fetches firing alerts from Grafana's Alertmanager API.
type GrafanaAlertsFetcher interface {
	GetAlertmanagerAlerts(ctx context.Context, authHeaders http.Header) ([]byte, error)
}

const (
	adreDisabledMsg  = "ADRE is disabled. Enable it in Settings."
	adreURLNotSetMsg = "HolmesGPT URL is not configured. Set it in Settings."
)

// Handlers provides HTTP handlers for the ADRE proxy API.
type Handlers struct {
	db                 reform.DBTX
	grafanaAlertsFetch GrafanaAlertsFetcher
	reqTimeout         time.Duration
	streamTimeout      time.Duration
	l                  *logrus.Entry
}

// NewHandlers creates new ADRE HTTP handlers.
func NewHandlers(db reform.DBTX, grafanaAlertsFetch GrafanaAlertsFetcher) *Handlers {
	return &Handlers{
		db:                 db,
		grafanaAlertsFetch: grafanaAlertsFetch,
		reqTimeout:         60 * time.Second,
		streamTimeout:      5 * time.Minute,
		l:                  logrus.WithField("component", "adre-handlers"),
	}
}

// checkAdreEnabled returns (settings, true) if ADRE is enabled and URL is set; otherwise writes an error and returns (nil, false).
func (h *Handlers) checkAdreEnabled(w http.ResponseWriter) (*models.Settings, bool) {
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return nil, false
	}
	if !settings.IsAdreEnabled() {
		writeJSONError(w, http.StatusBadRequest, adreDisabledMsg)
		return nil, false
	}
	url := settings.GetAdreURL()
	if url == "" {
		writeJSONError(w, http.StatusBadRequest, adreURLNotSetMsg)
		return nil, false
	}
	return settings, true
}

// GetSettings handles GET /v1/adre/settings.
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	chatPromptDisplay := settings.Adre.ChatPrompt
	if chatPromptDisplay == "" {
		chatPromptDisplay = DefaultChatPrompt
	}
	investigationPromptDisplay := settings.Adre.InvestigationPrompt
	if investigationPromptDisplay == "" {
		investigationPromptDisplay = DefaultInvestigationPrompt
	}
	agentPromptDisplay := settings.Adre.AgentPrompt
	if agentPromptDisplay == "" {
		agentPromptDisplay = DefaultPMMAgentPrompt
	}
	resp := struct {
		Enabled                    bool   `json:"enabled"`
		URL                        string `json:"url"`
		ChatPrompt                 string `json:"chat_prompt"`
		InvestigationPrompt        string `json:"investigation_prompt"`
		ChatPromptDisplay          string `json:"chat_prompt_display"`
		InvestigationPromptDisplay string `json:"investigation_prompt_display"`
		DefaultChatMode            string `json:"default_chat_mode"`
		ChatBackend                string `json:"chat_backend"`
		ChatHistoryLength          int    `json:"chat_history_length"`
		AgentPrompt                string `json:"agent_prompt"`
		AgentPromptDisplay         string `json:"agent_prompt_display"`
	}{
		Enabled:                    settings.IsAdreEnabled(),
		URL:                        settings.GetAdreURL(),
		ChatPrompt:                 settings.Adre.ChatPrompt,
		InvestigationPrompt:        settings.Adre.InvestigationPrompt,
		ChatPromptDisplay:          chatPromptDisplay,
		InvestigationPromptDisplay: investigationPromptDisplay,
		DefaultChatMode:            settings.Adre.DefaultChatMode,
		ChatBackend:                settings.Adre.ChatBackend,
		ChatHistoryLength:         settings.Adre.ChatHistoryLength,
		AgentPrompt:                settings.Adre.AgentPrompt,
		AgentPromptDisplay:         agentPromptDisplay,
	}
	if resp.DefaultChatMode == "" {
		resp.DefaultChatMode = "chat"
	}
	if resp.ChatBackend == "" {
		resp.ChatBackend = "holmesgpt"
	}
	if resp.ChatHistoryLength <= 0 {
		resp.ChatHistoryLength = 20
	}
	body, err := json.Marshal(resp)
	if err != nil {
		h.l.Errorf("Marshal settings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		h.l.Warnf("Write settings response: %v", err)
	}
}

// PostSettings handles POST /v1/adre/settings.
func (h *Handlers) PostSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Enabled             *bool   `json:"enabled"`
		URL                 *string `json:"url"`
		ChatPrompt          *string `json:"chat_prompt"`
		InvestigationPrompt *string `json:"investigation_prompt"`
		DefaultChatMode     *string `json:"default_chat_mode"`
		ChatBackend         *string `json:"chat_backend"`
		ChatHistoryLength   *int    `json:"chat_history_length"`
		AgentPrompt         *string `json:"agent_prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	hasChange := body.Enabled != nil || body.URL != nil || body.ChatPrompt != nil ||
		body.InvestigationPrompt != nil || body.DefaultChatMode != nil ||
		body.ChatBackend != nil || body.ChatHistoryLength != nil || body.AgentPrompt != nil
	if !hasChange {
		writeJSONError(w, http.StatusBadRequest, "No changes provided (set enabled, url, chat_prompt, investigation_prompt, default_chat_mode, chat_backend, chat_history_length, and/or agent_prompt)")
		return
	}
	if body.URL != nil {
		trimmed := strings.TrimSpace(*body.URL)
		body.URL = &trimmed
		if trimmed != "" {
			if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
				writeJSONError(w, http.StatusBadRequest, "URL must start with http:// or https://")
				return
			}
			parsed, err := url.Parse(trimmed)
			if err != nil || parsed.Host == "" {
				writeJSONError(w, http.StatusBadRequest, "URL must have a valid host")
				return
			}
		}
	}
	if body.ChatPrompt != nil && len(*body.ChatPrompt) > models.AdrePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chat_prompt: max %d bytes", models.AdrePromptMaxBytes))
		return
	}
	if body.InvestigationPrompt != nil && len(*body.InvestigationPrompt) > models.AdrePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("investigation_prompt: max %d bytes", models.AdrePromptMaxBytes))
		return
	}
	if body.DefaultChatMode != nil {
		mode := strings.TrimSpace(*body.DefaultChatMode)
		if mode != "chat" && mode != "investigation" {
			writeJSONError(w, http.StatusBadRequest, "default_chat_mode: must be \"chat\" or \"investigation\"")
			return
		}
		body.DefaultChatMode = &mode
	}
	if body.ChatBackend != nil {
		cb := strings.TrimSpace(*body.ChatBackend)
		if cb != "holmesgpt" && cb != "holmes_agent" {
			writeJSONError(w, http.StatusBadRequest, "chat_backend: must be \"holmesgpt\" or \"holmes_agent\"")
			return
		}
		body.ChatBackend = &cb
	}
	if body.ChatHistoryLength != nil {
		n := *body.ChatHistoryLength
		if n < 5 || n > 100 {
			writeJSONError(w, http.StatusBadRequest, "chat_history_length: must be between 5 and 100")
			return
		}
	}
	if body.AgentPrompt != nil && len(*body.AgentPrompt) > models.AdrePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("agent_prompt: max %d bytes", models.AdrePromptMaxBytes))
		return
	}
	params := &models.ChangeSettingsParams{
		EnableAdre:              body.Enabled,
		AdreURL:                 body.URL,
		AdreChatPrompt:          body.ChatPrompt,
		AdreInvestigationPrompt: body.InvestigationPrompt,
		AdreDefaultChatMode:     body.DefaultChatMode,
		ChatBackend:             body.ChatBackend,
		ChatHistoryLength:       body.ChatHistoryLength,
		AgentPrompt:             body.AgentPrompt,
	}
	if _, err := models.UpdateSettings(h.db, params); err != nil {
		h.l.Errorf("UpdateSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	settings, _ := models.GetSettings(h.db)
	chatPromptDisplay := settings.Adre.ChatPrompt
	if chatPromptDisplay == "" {
		chatPromptDisplay = DefaultChatPrompt
	}
	investigationPromptDisplay := settings.Adre.InvestigationPrompt
	if investigationPromptDisplay == "" {
		investigationPromptDisplay = DefaultInvestigationPrompt
	}
	agentPromptDisplayPost := settings.Adre.AgentPrompt
	if agentPromptDisplayPost == "" {
		agentPromptDisplayPost = DefaultPMMAgentPrompt
	}
	resp := struct {
		Enabled                    bool   `json:"enabled"`
		URL                        string `json:"url"`
		ChatPrompt                 string `json:"chat_prompt"`
		InvestigationPrompt        string `json:"investigation_prompt"`
		ChatPromptDisplay          string `json:"chat_prompt_display"`
		InvestigationPromptDisplay string `json:"investigation_prompt_display"`
		DefaultChatMode            string `json:"default_chat_mode"`
		ChatBackend                string `json:"chat_backend"`
		ChatHistoryLength          int    `json:"chat_history_length"`
		AgentPrompt                string `json:"agent_prompt"`
		AgentPromptDisplay         string `json:"agent_prompt_display"`
	}{
		Enabled:                    settings.IsAdreEnabled(),
		URL:                        settings.GetAdreURL(),
		ChatPrompt:                 settings.Adre.ChatPrompt,
		InvestigationPrompt:        settings.Adre.InvestigationPrompt,
		ChatPromptDisplay:          chatPromptDisplay,
		InvestigationPromptDisplay: investigationPromptDisplay,
		DefaultChatMode:            settings.Adre.DefaultChatMode,
		ChatBackend:                settings.Adre.ChatBackend,
		ChatHistoryLength:         settings.Adre.ChatHistoryLength,
		AgentPrompt:                settings.Adre.AgentPrompt,
		AgentPromptDisplay:         agentPromptDisplayPost,
	}
	if resp.DefaultChatMode == "" {
		resp.DefaultChatMode = "chat"
	}
	if resp.ChatBackend == "" {
		resp.ChatBackend = "holmesgpt"
	}
	if resp.ChatHistoryLength <= 0 {
		resp.ChatHistoryLength = 20
	}
	respBody, err := json.Marshal(resp)
	if err != nil {
		h.l.Errorf("Marshal settings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(respBody); err != nil {
		h.l.Warnf("Write settings response: %v", err)
	}
}

// GetModels handles GET /v1/adre/models.
func (h *Handlers) GetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	client := NewClient(settings.GetAdreURL())
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	modelsList, err := client.Models(ctx)
	if err != nil {
		h.l.Warnf("HolmesGPT Models: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	resp := struct {
		ModelName []string `json:"model_name"`
	}{ModelName: modelsList}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.l.Errorf("Encode models: %v", err)
	}
}

// chatRequestBody is the incoming POST /v1/adre/chat body. Mode is used only server-side to pick the prompt; it is not sent to Holmes.
type chatRequestBody struct {
	ChatRequest
	Mode *string `json:"mode,omitempty"`
}

// resolvePMMAgentPrompt returns the system prompt for the PMM Agent. Empty settings value uses built-in default.
func resolvePMMAgentPrompt(settings *models.Settings) string {
	if settings.Adre.AgentPrompt != "" {
		return settings.Adre.AgentPrompt
	}
	return DefaultPMMAgentPrompt
}

// resolveChatPrompt returns the additional_system_prompt for chat from settings and mode. Empty settings value uses built-in default.
func resolveChatPrompt(settings *models.Settings, mode string) string {
	if mode == "investigation" {
		if settings.Adre.InvestigationPrompt != "" {
			return settings.Adre.InvestigationPrompt
		}
		return DefaultInvestigationPrompt
	}
	if settings.Adre.ChatPrompt != "" {
		return settings.Adre.ChatPrompt
	}
	return DefaultChatPrompt
}

// PostChat handles POST /v1/adre/chat. If body has "stream": true, streams the response.
// Only holmes_agent (PMM Agent) and holmesgpt (Holmes Agent direct) are supported.
func (h *Handlers) PostChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	if !settings.IsAdreEnabled() {
		writeJSONError(w, http.StatusBadRequest, adreDisabledMsg)
		return
	}
	cb := settings.Adre.ChatBackend
	if cb == "" {
		cb = "holmesgpt"
	}
	if cb != "holmes_agent" && cb != "holmesgpt" {
		writeJSONError(w, http.StatusBadRequest, "Chat backend must be PMM Agent (holmes_agent) or Holmes Agent (holmesgpt). Configure it in AI Assistant Settings.")
		return
	}
	if settings.GetAdreURL() == "" {
		writeJSONError(w, http.StatusBadRequest, adreURLNotSetMsg)
		return
	}
	var body chatRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if cb == "holmes_agent" {
		if strings.TrimSpace(body.Ask) == "" {
			writeJSONError(w, http.StatusBadRequest, "ask is required")
			return
		}
		if body.Stream {
			RunPMMAgentChatStream(w, r, h.db, h.l, settings, body.Ask, body.ConversationHistory, h.streamTimeout)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "PMM Agent chat requires stream: true")
		return
	}
	// holmesgpt: direct Holmes chat
	mode := "chat"
	if body.Mode != nil && (*body.Mode == "chat" || *body.Mode == "investigation") {
		mode = *body.Mode
	} else if settings.Adre.DefaultChatMode == "investigation" {
		mode = "investigation"
	} else {
		mode = "chat"
	}
	req := &body.ChatRequest
	req.AdditionalSystemPrompt = resolveChatPrompt(settings, mode)
	client := NewClient(settings.GetAdreURL())
	if req.Stream {
		ctx, cancel := context.WithTimeout(r.Context(), h.streamTimeout)
		defer cancel()
		streamBody, err := client.ChatStream(ctx, req)
		if err != nil {
			h.l.Warnf("HolmesGPT ChatStream: %v", err)
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		defer streamBody.Close()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := streamBody.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					h.l.Warnf("ChatStream write: %v", werr)
					return
				}
				flusher.Flush()
			}
			if err != nil {
				if err != io.EOF {
					h.l.Warnf("ChatStream read: %v", err)
				}
				return
			}
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	resp, err := client.Chat(ctx, req)
	if err != nil {
		h.l.Warnf("HolmesGPT Chat: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.l.Errorf("Encode chat: %v", err)
	}
}

// GetAlerts handles GET /v1/adre/alerts — fetches firing alerts from Grafana's Alertmanager API.
func (h *Handlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	authHeaders := make(http.Header)
	if v := r.Header.Get("Authorization"); v != "" {
		authHeaders.Set("Authorization", v)
	}
	if v := r.Header.Get("Cookie"); v != "" {
		authHeaders.Set("Cookie", v)
	}
	raw, err := h.grafanaAlertsFetch.GetAlertmanagerAlerts(ctx, authHeaders)
	if err != nil {
		h.l.Warnf("Grafana Alertmanager alerts: %v", err)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch alerts: %v", err))
		return
	}
	// Grafana returns an array; frontend expects data.alerts or data.data.alerts. Wrap as {"alerts": raw}.
	var alerts json.RawMessage
	if err := json.Unmarshal(raw, &alerts); err != nil {
		h.l.Warnf("Parse alerts: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Invalid alerts response")
		return
	}
	out := map[string]json.RawMessage{"alerts": alerts}
	body, err := json.Marshal(out)
	if err != nil {
		h.l.Errorf("Marshal alerts: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		h.l.Errorf("Write alerts: %v", err)
	}
}

// PostInvestigate handles POST /v1/adre/investigate. If body has "stream": true, streams the response.
func (h *Handlers) PostInvestigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	var body struct {
		Source      string      `json:"source"`
		Title       string      `json:"title"`
		Description string      `json:"description"`
		Subject     interface{} `json:"subject,omitempty"`
		Context     interface{} `json:"context,omitempty"`
		Model       string      `json:"model,omitempty"`
		Stream      bool        `json:"stream,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	investigationPrompt := settings.Adre.InvestigationPrompt
	if investigationPrompt == "" {
		investigationPrompt = DefaultInvestigationPrompt
	}
	subject := body.Subject
	if subject == nil {
		subject = map[string]interface{}{} // HolmesGPT InvestigateRequest requires subject
	}
	req := &InvestigateRequest{
		Source:                 body.Source,
		Title:                  body.Title,
		Description:            body.Description,
		Subject:                subject,
		Context:                body.Context,
		Model:                  body.Model,
		AdditionalSystemPrompt: investigationPrompt,
	}
	client := NewClient(settings.GetAdreURL())
	if body.Stream {
		ctx, cancel := context.WithTimeout(r.Context(), h.streamTimeout)
		defer cancel()
		streamBody, err := client.InvestigateStream(ctx, req)
		if err != nil {
			h.l.Warnf("HolmesGPT InvestigateStream: %v", err)
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		defer streamBody.Close()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := streamBody.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					h.l.Warnf("InvestigateStream write: %v", werr)
					return
				}
				flusher.Flush()
			}
			if err != nil {
				if err != io.EOF {
					h.l.Warnf("InvestigateStream read: %v", err)
				}
				return
			}
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	resp, err := client.Investigate(ctx, req)
	if err != nil {
		h.l.Warnf("HolmesGPT Investigate: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.l.Errorf("Encode investigate: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
