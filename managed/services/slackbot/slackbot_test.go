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

	"github.com/percona/pmm/managed/models"
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

func TestFormatAnswerForSlack_keepsMarkdownLinks(t *testing.T) {
	raw := `Here is the graph. [Open in Grafana](https://pmm-server:8443/graph/d/mysql-instance-summary?viewPanel=53)`
	assert.Equal(t, raw, FormatAnswerForSlack(raw, "", false))
}

func TestSlackMrkdwn(t *testing.T) {
	md := "## Summary\nThis PMM has **14 nodes**. [Open in Grafana](https://pmm-server:8443/graph/d/mysql-instance-summary?viewPanel=53)\n### MySQL (2) ##\n| a | b |"
	assert.Equal(t,
		"*Summary*\nThis PMM has *14 nodes*. <https://pmm-server:8443/graph/d/mysql-instance-summary?viewPanel=53|Open in Grafana>\n*MySQL (2)*\n| a | b |",
		slackMrkdwn(md))
}

func TestNotificationText(t *testing.T) {
	assert.Equal(t, "This PMM has 14 nodes and 15 services.",
		notificationText("## Summary\n\nThis PMM has **14 nodes** and `15` services.\n\n| a | b |"))
	assert.Equal(t, "Restart the agent", notificationText("```sql\nSELECT 1;\n```\n- [Restart the agent](https://x/y)"))
	assert.Equal(t, "PMM Assistant", notificationText("| a | b |\n|---|---|"))

	long := notificationText(strings.Repeat("word ", 100))
	assert.True(t, strings.HasSuffix(long, "…"))
	assert.LessOrEqual(t, len([]rune(long)), slackNotificationLen+1)
}

func TestBuildReportSummary(t *testing.T) {
	inv := &models.Investigation{
		Title: "Alert: replication stopped", RootCauseSummary: "A script writes to the replica.",
		ResolutionSummary: "Set super_read_only.",
	}
	assert.Equal(t,
		"✅ Investigation complete: **Alert: replication stopped**\n\n**Root cause:** A script writes to the replica.\n\n"+
			"**Resolution:** Set super_read_only.\n\n[Full report](https://pmm.example.com/pmm-ui/investigations/inv1)",
		buildReportSummary(inv, "https://pmm.example.com/pmm-ui/investigations/inv1"))
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
