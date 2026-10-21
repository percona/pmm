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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func msg(role, content string) map[string]any {
	return map[string]any{"role": role, "content": content}
}

// turns builds a system message followed by n turns of user, assistant tool call, tool result, assistant answer.
func turns(n int) []any {
	hist := make([]any, 0, 1+4*n)
	hist = append(hist, msg("system", "sys"))
	for range n {
		hist = append(hist,
			msg("user", "q"),
			map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": "t"}}},
			map[string]any{"role": "tool", "tool_call_id": "t", "content": "r"},
			msg("assistant", "a"),
		)
	}
	return hist
}

func TestTrimHolmesHistory(t *testing.T) {
	t.Parallel()

	t.Run("within caps is unchanged", func(t *testing.T) {
		t.Parallel()
		hist := turns(5)
		assert.Equal(t, hist, TrimHolmesHistory(hist, 40, holmesHistoryMaxBytes))
	})

	t.Run("over the message cap drops oldest whole turns to half the cap", func(t *testing.T) {
		t.Parallel()
		hist := turns(12) // 49 messages
		got := TrimHolmesHistory(hist, 40, holmesHistoryMaxBytes)
		require.Len(t, got, 1+5*4)
		assert.Equal(t, "system", holmesMessageRole(got[0]))
		assert.Equal(t, "user", holmesMessageRole(got[1]), "kept history must start at a turn boundary")
		assert.Equal(t, hist[len(hist)-1], got[len(got)-1])
	})

	t.Run("over the byte cap", func(t *testing.T) {
		t.Parallel()
		hist := turns(4)
		hist[3] = map[string]any{"role": "tool", "tool_call_id": "t", "content": strings.Repeat("x", 1000)}
		got := TrimHolmesHistory(hist, 40, 1000)
		require.Len(t, got, 1+3*4)
		assert.Equal(t, "user", holmesMessageRole(got[1]))
	})

	t.Run("one oversized turn is kept", func(t *testing.T) {
		t.Parallel()
		hist := turns(1)
		assert.Equal(t, hist, TrimHolmesHistory(hist, 4, 10))
	})
}

func TestDecodeHolmesHistory(t *testing.T) {
	t.Parallel()
	assert.Nil(t, decodeHolmesHistory(nil))
	assert.Nil(t, decodeHolmesHistory([]byte(`null`)))
	assert.Nil(t, decodeHolmesHistory([]byte(`[{"role":"user","content":"q"},{"role":"assistant","content":"a"}]`)))
	got := decodeHolmesHistory([]byte(`[{"role":"system","content":"s"},{"role":"user","content":"q"}]`))
	require.Len(t, got, 2)
	assert.Equal(t, "user", holmesMessageRole(got[1]))
}

func TestParseHolmesSSEStreamHistory(t *testing.T) {
	t.Parallel()

	t.Run("answer end", func(t *testing.T) {
		t.Parallel()
		stream := "event: ai_answer_end\n" +
			`data: {"analysis":"done","conversation_history":[{"role":"system","content":"s"},{"role":"user","content":"q"}],"metadata":{}}` +
			"\n\n"
		out, sawErr, err := parseHolmesSSEStream(strings.NewReader(stream), nil)
		require.NoError(t, err)
		assert.False(t, sawErr)
		assert.False(t, out.Paused)
		assert.Equal(t, "done", out.Analysis)
		assert.JSONEq(t, `[{"role":"system","content":"s"},{"role":"user","content":"q"}]`, string(out.ConversationHistory))
	})

	t.Run("paused for frontend tools", func(t *testing.T) {
		t.Parallel()
		stream := "event: approval_required\n" +
			`data: {"analysis":"","conversation_history":[{"role":"system","content":"s"}],"pending_frontend_tool_calls":[{"tool_call_id":"t"}]}` +
			"\n\n"
		out, _, err := parseHolmesSSEStream(strings.NewReader(stream), nil)
		require.NoError(t, err)
		assert.True(t, out.Paused)
		assert.NotEmpty(t, out.ConversationHistory)
	})

	t.Run("large history line", func(t *testing.T) {
		t.Parallel()
		big := strings.Repeat("x", 3<<20)
		stream := "event: ai_answer_end\n" +
			`data: {"analysis":"ok","conversation_history":[{"role":"system","content":"` + big + `"}]}` + "\n\n"
		out, _, err := parseHolmesSSEStream(strings.NewReader(stream), nil)
		require.NoError(t, err)
		assert.Equal(t, "ok", out.Analysis)
		assert.Greater(t, len(out.ConversationHistory), 3<<20)
	})
}

func TestStripThinking(t *testing.T) {
	t.Parallel()
	hist := decodeHolmesHistory([]byte(`[
		{"role":"system","content":"s"},
		{"role":"user","content":"q","thinking_blocks":"kept on user"},
		{"role":"assistant","content":"a","reasoning_content":"r","thinking_blocks":[{"type":"thinking","signature":"x"}],
		 "provider_specific_fields":{"thinking_blocks":[{"type":"thinking","signature":"x"}]}},
		{"role":"assistant","content":[{"type":"thinking","signature":"x"},{"type":"redacted_thinking"},{"type":"text","text":"t"}]}
	]`))
	got := stripThinking(hist)
	require.Len(t, got, 4)
	assert.Equal(t, map[string]any{"role": "assistant", "content": "a"}, got[2])
	assert.Equal(t, []any{map[string]any{"type": "text", "text": "t"}}, got[3].(map[string]any)["content"])
	assert.Equal(t, "kept on user", got[1].(map[string]any)["thinking_blocks"])
}
