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
	"context"
	"math"
	"net/http"
	"slices"

	"github.com/percona/pmm/managed/models"
)

// ChatInvestigator turns an Assistant chat into an investigation; investigations.Handlers implements it.
type ChatInvestigator interface {
	// InvestigationFromChat returns the id of the investigation made from the chat, and whether it already existed.
	InvestigationFromChat(ctx context.Context, conv *models.AdreConversation, msgs []models.AdreMessage, login string) (string, bool, error)
}

// SetChatInvestigator injects what turns chats into investigations; without it the endpoint is unavailable.
func (h *Handlers) SetChatInvestigator(ci ChatInvestigator) {
	h.chatInvestigator = ci
}

// postConversationInvestigation handles POST /v1/adre/conversations/:id/investigation.
func (h *Handlers) postConversationInvestigation(w http.ResponseWriter, r *http.Request, id int64) {
	if h.chatInvestigator == nil {
		writeJSONError(w, http.StatusNotImplemented, "Investigations are not available")
		return
	}
	login, ok := h.resolveUserLogin(w, r)
	if !ok {
		return
	}
	conv, err := models.GetAdreConversationOwned(h.db, id, login)
	if err != nil {
		h.l.Errorf("GetAdreConversationOwned: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if conv == nil {
		writeNotFound(w)
		return
	}
	msgs, err := models.LoadAdreMessagesForHolmesHistory(h.db, id, math.MaxInt64)
	if err != nil {
		h.l.Errorf("LoadAdreMessagesForHolmesHistory: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load messages")
		return
	}
	if !slices.ContainsFunc(msgs, func(m models.AdreMessage) bool { return m.Role == "assistant" }) {
		writeJSONError(w, http.StatusBadRequest, "This chat has no answer yet")
		return
	}
	invID, reused, err := h.chatInvestigator.InvestigationFromChat(r.Context(), conv, msgs, login)
	if err != nil {
		h.l.WithField("conversation_id", id).Errorf("InvestigationFromChat: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create the investigation")
		return
	}
	status := http.StatusCreated
	if reused {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"id": invID, "reused": reused})
}
