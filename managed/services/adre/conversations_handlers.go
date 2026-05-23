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
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/percona/pmm/managed/models"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson // response already committed; nothing actionable on encode failure
}

func writeNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
}

func writeRateLimited(w http.ResponseWriter, retryAfterSec int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson // response already committed; nothing actionable on encode failure
		"code":            "rate_limited",
		"message":         "Too many search requests. Try again later.",
		"retry_after_sec": retryAfterSec,
	})
}

// ListConversations handles GET /v1/adre/conversations.
func (h *Handlers) ListConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	qv := r.URL.Query()
	limit, _ := strconv.Atoi(qv.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	titleQ := strings.TrimSpace(qv.Get("q"))
	t, id, _ := models.DecodeAdreConversationCursor(qv.Get("cursor"))
	rows, err := models.ListAdreConversations(h.db, login, titleQ, limit, t, id)
	if err != nil {
		h.l.Errorf("ListAdreConversations: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to list conversations")
		return
	}
	var nextCursor string
	if len(rows) > 0 && len(rows) >= limit {
		last := rows[len(rows)-1]
		nextCursor = models.EncodeAdreConversationCursor(last.LastMessageAt, last.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversations": rows,
		"next_cursor":   nextCursor,
	})
}

type createConversationBody struct {
	Title *string `json:"title"`
}

// CreateConversation handles POST /v1/adre/conversations.
func (h *Handlers) CreateConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	var body createConversationBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	title := models.DefaultAdreChatTitle
	if body.Title != nil {
		t := strings.TrimSpace(*body.Title)
		if t != "" {
			title = models.TruncateAdreTitle(t)
		}
	}
	if title == "" {
		writeJSONError(w, http.StatusBadRequest, "title must be 1–50 characters")
		return
	}
	c := &models.AdreConversation{
		Title:     title,
		CreatedBy: login,
	}
	err = models.CreateAdreConversation(h.db, c)
	if err != nil {
		h.l.Errorf("CreateAdreConversation: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create conversation")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              c.ID,
		"title":           c.Title,
		"created_at":      c.CreatedAt, //nolint:goconst
		"updated_at":      c.UpdatedAt,
		"last_message_at": c.LastMessageAt,
	})
}

// ServeConversationSubroutes handles /v1/adre/conversations/{id}[...].
func (h *Handlers) ServeConversationSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/adre/conversations/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getConversation(w, r, id)
		case http.MethodPatch:
			h.patchConversation(w, r, id)
		case http.MethodDelete:
			h.deleteConversation(w, r, id)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 2 && parts[1] == "messages" && r.Method == http.MethodGet {
		h.listMessages(w, r, id)
		return
	}
	http.NotFound(w, r)
}

func (h *Handlers) getConversation(w http.ResponseWriter, r *http.Request, id int64) {
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	c, err := models.GetAdreConversationOwned(h.db, id, login)
	if err != nil {
		h.l.Errorf("GetAdreConversationOwned: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if c == nil {
		writeNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":              c.ID,
		"title":           c.Title,
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
		"last_message_at": c.LastMessageAt,
		"metadata":        json.RawMessage(c.MetadataJSON),
	})
}

type patchConversationBody struct {
	Title *string `json:"title"`
}

func (h *Handlers) patchConversation(w http.ResponseWriter, r *http.Request, id int64) {
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	var body patchConversationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Title == nil {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	t := strings.TrimSpace(*body.Title)
	if t == "" {
		writeJSONError(w, http.StatusBadRequest, "title must not be empty")
		return
	}
	t = models.TruncateAdreTitle(t)
	if t == "" {
		writeJSONError(w, http.StatusBadRequest, "title must be 1–50 characters")
		return
	}
	c, err := models.GetAdreConversationOwned(h.db, id, login)
	if err != nil {
		h.l.Errorf("GetAdreConversationOwned: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if c == nil {
		writeNotFound(w)
		return
	}
	c.Title = t
	if err := models.UpdateAdreConversation(h.db, c); err != nil { //nolint:noinlineerr
		h.l.Errorf("UpdateAdreConversation: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to update conversation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         c.ID,
		"title":      c.Title,
		"updated_at": c.UpdatedAt,
	})
}

func (h *Handlers) deleteConversation(w http.ResponseWriter, r *http.Request, id int64) {
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	if h.streams != nil {
		h.streams.Abort(id)
	}
	deleted, err := models.DeleteAdreConversation(h.db, id, login)
	if err != nil {
		h.l.Errorf("DeleteAdreConversation: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete conversation")
		return
	}
	if !deleted {
		writeNotFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) listMessages(w http.ResponseWriter, r *http.Request, conversationID int64) {
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	c, err := models.GetAdreConversationOwned(h.db, conversationID, login)
	if err != nil {
		h.l.Errorf("GetAdreConversationOwned: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if c == nil {
		writeNotFound(w)
		return
	}
	qv := r.URL.Query()
	limit, _ := strconv.Atoi(qv.Get("limit"))
	var beforeID, afterID *int64
	if s := strings.TrimSpace(qv.Get("before")); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid before")
			return
		}
		beforeID = &v
	}
	if s := strings.TrimSpace(qv.Get("after")); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid after")
			return
		}
		afterID = &v
	}
	if beforeID != nil && afterID != nil {
		writeJSONError(w, http.StatusBadRequest, "specify only one of before or after")
		return
	}
	msgs, err := models.ListAdreMessages(h.db, conversationID, beforeID, afterID, limit)
	if err != nil {
		h.l.Errorf("ListAdreMessages: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load messages")
		return
	}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		row := map[string]any{
			"id":              m.ID,
			"conversation_id": m.ConversationID,
			"role":            m.Role,    //nolint:goconst
			"content":         m.Content, //nolint:goconst
			"created_at":      m.CreatedAt,
			"model":           m.Model,
		}
		if m.ToolName != "" {
			row["tool_name"] = m.ToolName
		}
		if len(m.ToolResultJSON) > 0 {
			row["tool_result_json"] = json.RawMessage(m.ToolResultJSON)
		}
		if m.PromptTokens != nil {
			row["prompt_tokens"] = *m.PromptTokens
		}
		if m.CompletionTokens != nil {
			row["completion_tokens"] = *m.CompletionTokens
		}
		if m.TotalTokens != nil {
			row["total_tokens"] = *m.TotalTokens
		}
		if m.CachedTokens != nil {
			row["cached_tokens"] = *m.CachedTokens
		}
		if m.TotalCost != nil {
			row["total_cost"] = *m.TotalCost
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

// SearchMessages handles GET /v1/adre/messages/search.
func (h *Handlers) SearchMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	if h.searchLimiter != nil {
		if ok, retry := h.searchLimiter.Allow(login); !ok {
			writeRateLimited(w, retry)
			return
		}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSONError(w, http.StatusBadRequest, "q is required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits, err := models.SearchAdreMessagesFTS(h.db, login, q, limit)
	if err != nil {
		h.l.Errorf("SearchAdreMessagesFTS: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Search failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (h *Handlers) resolveUserLogin(w http.ResponseWriter, r *http.Request) (string, bool) {
	ctx := r.Context()
	headers := grafanaAuthHeadersFromRequest(r)
	login, err := h.grafana.GetCurrentUserLogin(ctx, headers)
	if err != nil {
		h.l.Debugf("GetCurrentUserLogin: %v", err)
		writeJSONError(w, http.StatusUnauthorized, "Authentication required")
		return "", false
	}
	return login, true
}

func (h *Handlers) userLoginFromRequest(r *http.Request) string {
	login, err := h.grafana.GetCurrentUserLogin(r.Context(), grafanaAuthHeadersFromRequest(r))
	if err != nil {
		return ""
	}
	return login
}
