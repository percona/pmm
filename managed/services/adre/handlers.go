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

const (
	adreDisabledMsg  = "ADRE is disabled. Enable it in Settings."
	adreURLNotSetMsg = "HolmesGPT URL is not configured. Set it in Settings."
)

// Handlers provides HTTP handlers for the ADRE proxy API.
type Handlers struct {
	db           reform.DBTX
	vmalertURL   string
	reqTimeout   time.Duration
	streamTimeout time.Duration
	l            *logrus.Entry
}

// NewHandlers creates new ADRE HTTP handlers.
func NewHandlers(db reform.DBTX, vmalertURL string) *Handlers {
	vmalertURL = strings.TrimSuffix(vmalertURL, "/")
	return &Handlers{
		db:            db,
		vmalertURL:    vmalertURL,
		reqTimeout:    60 * time.Second,
		streamTimeout: 5 * time.Minute,
		l:             logrus.WithField("component", "adre-handlers"),
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
	resp := struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}{
		Enabled: settings.IsAdreEnabled(),
		URL:     settings.GetAdreURL(),
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
		Enabled *bool  `json:"enabled"`
		URL     *string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Enabled == nil && body.URL == nil {
		writeJSONError(w, http.StatusBadRequest, "No changes provided (set enabled and/or url)")
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
	params := &models.ChangeSettingsParams{
		EnableAdre: body.Enabled,
		AdreURL:    body.URL,
	}
	if _, err := models.UpdateSettings(h.db, params); err != nil {
		h.l.Errorf("UpdateSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	settings, _ := models.GetSettings(h.db)
	resp := struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}{
		Enabled: settings.IsAdreEnabled(),
		URL:     settings.GetAdreURL(),
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

// PostChat handles POST /v1/adre/chat. If body has "stream": true, streams the response.
func (h *Handlers) PostChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	client := NewClient(settings.GetAdreURL())
	if req.Stream {
		ctx, cancel := context.WithTimeout(r.Context(), h.streamTimeout)
		defer cancel()
		streamBody, err := client.ChatStream(ctx, &req)
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
	resp, err := client.Chat(ctx, &req)
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

// GetAlerts handles GET /v1/adre/alerts — fetches firing alerts from VMAlert.
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.vmalertURL+"/api/v1/alerts", nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	client := &http.Client{Timeout: h.reqTimeout}
	resp, err := client.Do(req)
	if err != nil {
		h.l.Warnf("VMAlert alerts: %v", err)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch alerts: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("VMAlert %s: %s", resp.Status, string(body)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.l.Errorf("Copy alerts: %v", err)
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
	req := &InvestigateRequest{
		Source:      body.Source,
		Title:       body.Title,
		Description: body.Description,
		Subject:     body.Subject,
		Context:     body.Context,
		Model:       body.Model,
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
