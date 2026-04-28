// Copyright (C) 2026 Percona LLC
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// chatRequestBody is the incoming POST /v1/adre/chat body.
type chatRequestBody struct {
	ChatRequest
	ConversationID interface{} `json:"conversation_id"`
	Mode           *string     `json:"mode,omitempty"`
	DashboardContext string    `json:"dashboard_context,omitempty"`
}

func parseConversationID(v interface{}) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("conversation_id is required")
	}
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case json.Number:
		return t.Int64()
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, fmt.Errorf("conversation_id is required")
		}
		return strconv.ParseInt(s, 10, 64)
	default:
		return 0, fmt.Errorf("invalid conversation_id type")
	}
}

func extractUsageFromMetadata(raw json.RawMessage) (prompt, completion, total *int32) {
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var meta struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, nil, nil
	}
	if len(meta.Usage) == 0 {
		return nil, nil, nil
	}
	var u struct {
		PromptTokens     *int32 `json:"prompt_tokens"`
		CompletionTokens *int32 `json:"completion_tokens"`
		TotalTokens      *int32 `json:"total_tokens"`
	}
	if json.Unmarshal(meta.Usage, &u) != nil {
		return nil, nil, nil
	}
	return u.PromptTokens, u.CompletionTokens, u.TotalTokens
}

func toolNameFromHolmesToolPayload(raw []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	if s, ok := m["tool_name"].(string); ok && s != "" {
		return s
	}
	if s, ok := m["name"].(string); ok && s != "" {
		return s
	}
	return ""
}

func persistAdreToolJSON(q *reform.DB, conversationID int64, raw []byte) error {
	name := toolNameFromHolmesToolPayload(raw)
	content := ""
	msg := &models.AdreMessage{
		ConversationID: conversationID,
		Role:           "tool",
		Content:        content,
		ToolName:       name,
		ToolResultJSON: append([]byte(nil), raw...),
	}
	return models.CreateAdreMessage(q, msg)
}

func persistAdreToolCalls(q *reform.DB, conversationID int64, calls []interface{}) error {
	for _, c := range calls {
		raw, err := json.Marshal(c)
		if err != nil {
			continue
		}
		if err := persistAdreToolJSON(q, conversationID, raw); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) postChatWithPersistence(w http.ResponseWriter, r *http.Request, settings *models.Settings, body *chatRequestBody) {
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	convID, err := parseConversationID(body.ConversationID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	conv, err := models.GetAdreConversationOwned(h.db, convID, login)
	if err != nil {
		h.l.Errorf("GetAdreConversationOwned: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if conv == nil {
		writeNotFound(w)
		return
	}

	ask := strings.TrimSpace(body.Ask)
	userMsg := &models.AdreMessage{
		ConversationID: convID,
		Role:           "user",
		Content:        ask,
	}
	if err := models.CreateAdreMessage(h.db, userMsg); err != nil {
		h.l.Errorf("CreateAdreMessage user: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save message")
		return
	}
	if err := models.TouchAdreConversationLastMessage(h.db, convID, userMsg.CreatedAt); err != nil {
		h.l.Warnf("TouchAdreConversationLastMessage: %v", err)
	}

	prior, err := models.LoadAdreMessagesForHolmesHistory(h.db, convID, userMsg.ID)
	if err != nil {
		h.l.Errorf("LoadAdreMessagesForHolmesHistory: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load history")
		return
	}
	hist := AdreMessagesToHolmesHistory(prior)
	maxMsgs := MaxConversationMessages(settings)
	hist = TrimConversationHistory(hist, maxMsgs)
	hist = EnsureHolmesLeadingSystemMessage(hist)

	mode := "fast"
	if body.Mode != nil {
		m := strings.TrimSpace(*body.Mode)
		if m == "investigation" {
			mode = "investigation"
		} else if m == "fast" || m == "chat" {
			mode = "fast"
		}
	} else if settings.Adre.DefaultChatMode == "investigation" {
		mode = "investigation"
	}
	req := &body.ChatRequest
	req.ConversationHistory = hist
	req.Model = resolveChatModel(settings, mode, req.Model)
	req.BehaviorControls = ResolveBehaviorControlsForPostChat(settings, mode)
	req.AdditionalSystemPrompt = ResolveChatSystemPrompt(settings, mode)
	if dc := strings.TrimSpace(body.DashboardContext); dc != "" {
		if len(dc) > maxDashboardContextBytes {
			dc = dc[:maxDashboardContextBytes] + "\n... (truncated)"
		}
		req.AdditionalSystemPrompt = strings.TrimRight(req.AdditionalSystemPrompt, "\n") + "\n\n" + dc
	}

	client := NewClient(settings.GetAdreURL())
	if req.Stream {
		h.postChatStream(w, r, settings, client, req, conv, login, ask, userMsg.ID)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	resp, err := client.Chat(ctx, req)
	if err != nil {
		h.l.Warnf("HolmesGPT Chat: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	promptT, compT, totT := extractUsageFromMetadata(resp.Metadata)
	asst := &models.AdreMessage{
		ConversationID:   convID,
		Role:             "assistant",
		Content:          resp.Analysis,
		Model:            req.Model,
		PromptTokens:     promptT,
		CompletionTokens: compT,
		TotalTokens:      totT,
	}
	if err := models.CreateAdreMessage(h.db, asst); err != nil {
		h.l.Errorf("CreateAdreMessage assistant: %v", err)
	}
	_ = models.TouchAdreConversationLastMessage(h.db, convID, asst.CreatedAt)
	if err := persistAdreToolCalls(h.db, convID, resp.ToolCalls); err != nil {
		h.l.Warnf("persistAdreToolCalls: %v", err)
	}
	h.maybeAutotitleConversation(convID, login, ask)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.l.Errorf("Encode chat: %v", err)
	}
}

func (h *Handlers) maybeAutotitleConversation(convID int64, login, ask string) {
	conv, err := models.GetAdreConversationOwned(h.db, convID, login)
	if err != nil || conv == nil {
		return
	}
	if conv.Title != models.DefaultAdreChatTitle {
		return
	}
	t := models.TruncateAdreTitle(ask)
	if t == "" {
		return
	}
	conv.Title = t
	if err := models.UpdateAdreConversation(h.db, conv); err != nil {
		h.l.Warnf("UpdateAdreConversation autotitle: %v", err)
	}
}

func (h *Handlers) postChatStream(w http.ResponseWriter, r *http.Request, settings *models.Settings, client *Client, req *ChatRequest, conv *models.AdreConversation, login, ask string, _ int64) {
	convID := conv.ID
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	unreg := h.streams.Register(convID, cancel)
	defer unreg()

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

	var acc bytes.Buffer
	tee := io.TeeReader(streamBody, &acc)
	buf := make([]byte, 32*1024)
	var copyErr error
	for {
		n, err := tee.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				copyErr = werr
				break
			}
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				copyErr = err
			}
			break
		}
	}
	if copyErr != nil {
		h.l.Warnf("ChatStream copy: %v", copyErr)
	}

	out, sawErr, parseErr := parseHolmesSSEStream(bytes.NewReader(acc.Bytes()), nil)
	if parseErr != nil {
		h.l.Warnf("parseHolmesSSEStream: %v", parseErr)
	}
	if ctx.Err() != nil {
		h.l.Debug("Chat stream context canceled; assistant message not persisted")
		return
	}
	if sawErr {
		h.l.Debug("Holmes stream ended with error event; assistant message not persisted")
		return
	}
	if copyErr != nil && copyErr != io.EOF {
		h.l.Warnf("ChatStream client write: %v", copyErr)
	}
	promptT, compT, totT := out.PromptTokens, out.CompletionTokens, out.TotalTokens
	asst := &models.AdreMessage{
		ConversationID:   convID,
		Role:             "assistant",
		Content:          out.Analysis,
		Model:            req.Model,
		PromptTokens:     promptT,
		CompletionTokens: compT,
		TotalTokens:      totT,
	}
	if err := models.CreateAdreMessage(h.db, asst); err != nil {
		h.l.Errorf("CreateAdreMessage assistant (stream): %v", err)
		return
	}
	_ = models.TouchAdreConversationLastMessage(h.db, convID, asst.CreatedAt)
	for _, raw := range out.ToolResultJSONRows {
		if err := persistAdreToolJSON(h.db, convID, raw); err != nil {
			h.l.Warnf("persistAdreToolJSON stream: %v", err)
		}
	}
	h.maybeAutotitleConversation(convID, login, ask)
}
