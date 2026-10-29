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
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func chatRows(t *testing.T) []models.AdreMessage {
	t.Helper()
	at := time.Date(2026, 10, 17, 21, 0, 0, 0, time.UTC)
	tool := func(name string, params map[string]any) models.AdreMessage {
		b, err := json.Marshal(map[string]any{"tool_name": name, "result": map[string]any{"status": "success", "params": params}})
		require.NoError(t, err)
		return models.AdreMessage{Role: "tool", ToolName: name, ToolResultJSON: b, CreatedAt: at}
	}
	return []models.AdreMessage{
		{Role: "user", Content: "Why do shop_app logins fail?", CreatedAt: at},
		{Role: "assistant", Content: "## Summary\n\nshop_app has the wrong password.\n\n## Details\n\nMore.", CreatedAt: at.Add(time.Minute)},
		tool("pmm_clickhouse_query", map[string]any{"database": "otel", "query": "SELECT count() FROM logs"}),
		tool("pmm_logs_patterns", map[string]any{"start": "2026-10-17T20:41:24Z", "end": "2026-10-17T21:11:25Z", "text": "shop_app"}),
		tool("TodoWrite", map[string]any{"todos": []any{}}),
		{Role: "user", Content: "And now?", CreatedAt: at.Add(2 * time.Minute)},
		{Role: "assistant", Content: "Still failing.", CreatedAt: at.Add(3 * time.Minute)},
	}
}

func TestChatTurns(t *testing.T) {
	turns := chatTurns(chatRows(t))

	require.Len(t, turns, 2)
	assert.Equal(t, "Why do shop_app logins fail?", turns[0].Question)
	require.Len(t, turns[0].Tools, 2, "the to-do list is the Assistant's own bookkeeping, not evidence")
	assert.Equal(t, "pmm_clickhouse_query", turns[0].Tools[0].Name)
	assert.Empty(t, turns[1].Tools)

	md := turns[0].markdown()
	assert.True(t, strings.HasPrefix(md, "## Summary\n\nshop_app has the wrong password."))
	assert.Contains(t, md, "#### Evidence")
	assert.Contains(t, md, "```sql\nSELECT count() FROM logs\n```")
	assert.Contains(t, md, "`pmm_logs_patterns`")
	assert.Contains(t, md, `"text": "shop_app"`)
	assert.NotContains(t, turns[1].markdown(), "Evidence")
}

func TestChatInvestigation(t *testing.T) {
	rows := chatRows(t)
	conv := &models.AdreConversation{ID: 7, Title: "Failed logins"}

	inv := chatInvestigation(conv, rows, chatTurns(rows), "alice")

	assert.Equal(t, "Chat: Failed logins", inv.Title)
	assert.Equal(t, "chat", inv.SourceType)
	assert.Equal(t, "7", inv.SourceRef)
	assert.Equal(t, "alice", inv.CreatedBy)
	assert.Equal(t, "shop_app has the wrong password.", inv.Summary, "the lead of the chat's longest answer")
	assert.Equal(t, time.Date(2026, 10, 17, 20, 41, 24, 0, time.UTC), inv.TimeFrom, "the window the tools looked at")
	assert.Equal(t, time.Date(2026, 10, 17, 21, 11, 25, 0, time.UTC), inv.TimeTo)
	assert.Equal(t, "Why do shop_app logins fail?", userRequestFromInvestigation(inv))

	untitled := chatInvestigation(&models.AdreConversation{ID: 8}, rows[:2], chatTurns(rows[:2]), "alice")
	assert.Equal(t, "Chat: Why do shop_app logins fail?", untitled.Title)
	assert.Equal(t, rows[0].CreatedAt.Add(-time.Hour), untitled.TimeFrom, "an hour before the chat when no tool says")
	assert.Equal(t, rows[1].CreatedAt, untitled.TimeTo)
}

func TestChatTranscript(t *testing.T) {
	turns := chatTurns(chatRows(t))

	transcript := chatTranscript(turns, 1<<20)
	assert.Contains(t, transcript, "## Question: Why do shop_app logins fail?")
	assert.Contains(t, transcript, "## Question: And now?")

	short := chatTranscript(turns, 60)
	assert.LessOrEqual(t, len(short), 60)
	assert.Contains(t, short, "Still failing.", "a long chat keeps its latest turns")
}
