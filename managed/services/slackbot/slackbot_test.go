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

package slackbot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripMentions(t *testing.T) {
	assert.Equal(t, "hello", stripMentions("<@U123> hello"))
	assert.Equal(t, "hello", stripMentions("  <@UABC>  hello  "))
}

func TestNeedsGraphRetry(t *testing.T) {
	assert.False(t, NeedsGraphRetry("", "x"))
	assert.False(t, NeedsGraphRetry("show graph", ""))
	assert.False(t, NeedsGraphRetry("hello", "<<{\"x\":1}>>"))
	assert.True(t, NeedsGraphRetry("show me a graph", "still <<{\"tool\":1}>>"))
	assert.False(t, NeedsGraphRetry("show graph", "see /v1/grafana/render/blob/"+strings.Repeat("a", 64)+".png"))
	assert.False(t, NeedsGraphRetry("show graph", "![x](http://example.com/y.png)"))
}

func TestExtractBlobHashes(t *testing.T) {
	h := strings.Repeat("a", 64)
	text := "Panel: /v1/grafana/render/blob/" + h + ".png ok"
	got := ExtractBlobHashes(text)
	require.Len(t, got, 1)
	assert.Equal(t, h, got[0])
}

func TestFormatAnswerForSlack(t *testing.T) {
	h := strings.Repeat("b", 64)
	raw := `Hello <<{"tool":"x"}>> see (/v1/grafana/render/blob/` + h + `.png)`
	out := FormatAnswerForSlack(raw, "https://pmm.example.com", false)
	assert.NotContains(t, out, "<<")
	assert.Contains(t, out, "https://pmm.example.com/v1/grafana/render/blob/")
}

func TestFormatAnswerForSlack_markdownLinkToSlackMrkdwn(t *testing.T) {
	raw := `Here is the graph. [Open in Grafana](https://pmm-server:8443/graph/d/mysql-instance-summary?viewPanel=53)`
	out := FormatAnswerForSlack(raw, "", false)
	assert.NotContains(t, out, "[Open in Grafana]")
	assert.Contains(t, out, "<https://pmm-server:8443/graph/d/mysql-instance-summary?viewPanel=53|Open in Grafana>")
}

func TestSlackEventDedupe(t *testing.T) {
	d := newRingDedupe(4)
	assert.True(t, d.firstSeen("T", "C", "1.0"))
	assert.False(t, d.firstSeen("T", "C", "1.0"))
	assert.True(t, d.firstSeen("T", "C", "1.1"))
	assert.True(t, d.firstSeen("T", "C", "1.2"))
	assert.True(t, d.firstSeen("T", "C", "1.3"))
	// ring full; oldest evicted — "1.0" can be seen again
	assert.True(t, d.firstSeen("T", "C", "1.4"))
	assert.True(t, d.firstSeen("T", "C", "1.0"))
}

func TestSlackEventDedupeForget(t *testing.T) {
	d := newRingDedupe(8)
	assert.True(t, d.firstSeen("a", "b", "c"))
	assert.False(t, d.firstSeen("a", "b", "c"))
	d.forget("a", "b", "c")
	assert.True(t, d.firstSeen("a", "b", "c"))
}

func TestThreadStoreUndoLastUser(t *testing.T) {
	ts := NewThreadStore()
	k := ThreadKey{TeamID: "t", ChannelID: "c", ThreadTS: "1"}
	ts.AppendUser(k, "hello")
	ts.UndoLastUserMessage(k, "hello")
	assert.Empty(t, ts.Snapshot(k))
	ts.AppendUser(k, "a")
	ts.AppendAssistant(k, "b")
	ts.UndoLastUserMessage(k, "a")
	// Last message is assistant; undo only strips trailing user — no change.
	assert.Len(t, ts.Snapshot(k), 2)
}

func TestThreadStoreCap(t *testing.T) {
	ts := NewThreadStore()
	k := ThreadKey{TeamID: "t", ChannelID: "c", ThreadTS: "1"}
	for range maxThreadMessagesRAM + 10 {
		ts.AppendUser(k, "u")
	}
	assert.Len(t, ts.Snapshot(k), maxThreadMessagesRAM)
}
