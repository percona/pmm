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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

type fakeChatInvestigator struct {
	conv   *models.AdreConversation
	msgs   []models.AdreMessage
	login  string
	reused bool
}

func (f *fakeChatInvestigator) InvestigationFromChat(
	_ context.Context, conv *models.AdreConversation, msgs []models.AdreMessage, login string,
) (string, bool, error) {
	f.conv, f.msgs, f.login = conv, msgs, login
	return "inv-1", f.reused, nil
}

var (
	conversationColumns = []string{"id", "title", "created_by", "created_at", "updated_at", "last_message_at", "metadata_json"}
	messageColumns      = []string{
		"id", "conversation_id", "role", "content", "tool_name", "tool_result_json", "model",
		"prompt_tokens", "completion_tokens", "total_tokens", "cached_tokens", "total_cost", "usage_event_id", "created_at",
	}
)

func chatInvestigationHandlers(t *testing.T) (*Handlers, sqlmock.Sqlmock, *fakeChatInvestigator) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	h := NewHandlers(reform.NewDB(sqlDB, postgresql.Dialect, nil), &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
	ci := &fakeChatInvestigator{}
	h.SetChatInvestigator(ci)
	return h, mock, ci
}

func expectOwnedConversation(mock sqlmock.Sqlmock, found bool) {
	rows := sqlmock.NewRows(conversationColumns)
	if found {
		now := time.Now()
		rows.AddRow(7, "Failed logins", "testuser", now, now, now, []byte("{}"))
	}
	mock.ExpectQuery(`FROM "adre_conversations" WHERE id = \$1 AND created_by = \$2`).
		WithArgs(int64(7), "testuser").
		WillReturnRows(rows)
}

func expectMessages(mock sqlmock.Sqlmock, roles ...string) {
	rows := sqlmock.NewRows(messageColumns)
	for i, role := range roles {
		rows.AddRow(i+1, 7, role, role+" text", "", nil, "", nil, nil, nil, nil, nil, nil, time.Now())
	}
	mock.ExpectQuery(`FROM adre_messages\s+WHERE conversation_id = \$1 AND id < \$2`).WillReturnRows(rows)
}

func postChatInvestigation(h *Handlers) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeConversationSubroutes(rec, httptest.NewRequest(http.MethodPost, "/v1/adre/conversations/7/investigation", nil))
	return rec
}

func TestPostConversationInvestigation(t *testing.T) {
	t.Run("creates an investigation from the owner's chat", func(t *testing.T) {
		h, mock, ci := chatInvestigationHandlers(t)
		expectOwnedConversation(mock, true)
		expectMessages(mock, "user", "assistant", "tool")

		rec := postChatInvestigation(h)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var body struct {
			ID     string `json:"id"`
			Reused bool   `json:"reused"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, "inv-1", body.ID)
		assert.False(t, body.Reused)
		assert.Equal(t, "testuser", ci.login)
		assert.Equal(t, int64(7), ci.conv.ID)
		assert.Len(t, ci.msgs, 3)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns the investigation already made from it", func(t *testing.T) {
		h, mock, ci := chatInvestigationHandlers(t)
		ci.reused = true
		expectOwnedConversation(mock, true)
		expectMessages(mock, "user", "assistant")

		assert.Equal(t, http.StatusOK, postChatInvestigation(h).Code)
	})

	t.Run("refuses a chat with no answer yet", func(t *testing.T) {
		h, mock, ci := chatInvestigationHandlers(t)
		expectOwnedConversation(mock, true)
		expectMessages(mock, "user")

		assert.Equal(t, http.StatusBadRequest, postChatInvestigation(h).Code)
		assert.Nil(t, ci.conv)
	})

	t.Run("hides other users' chats", func(t *testing.T) {
		h, mock, ci := chatInvestigationHandlers(t)
		expectOwnedConversation(mock, false)

		assert.Equal(t, http.StatusNotFound, postChatInvestigation(h).Code)
		assert.Nil(t, ci.conv)
	})
}
