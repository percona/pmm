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
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkForSlack(t *testing.T) {
	t.Parallel()

	assert.Nil(t, chunkForSlack(""))
	assert.Nil(t, chunkForSlack("\n\n"))

	// Short input → single chunk, unchanged.
	short := "## Summary\nall good"
	assert.Equal(t, []string{short}, chunkForSlack(short))

	// Long, newline-delimited report → multiple chunks, each within Slack's limit.
	var sb strings.Builder
	for range 600 {
		sb.WriteString("line of investigation output that is reasonably long to fill space\n")
	}
	long := sb.String()
	require.Greater(t, len(long), slackMaxMessageLen)
	chunks := chunkForSlack(long)
	require.Greater(t, len(chunks), 1, "long report must be split")
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), slackMaxMessageLen, "chunk exceeds Slack limit")
		assert.True(t, utf8.ValidString(c))
		assert.NotEmpty(t, c)
	}

	// A single very long line with no newlines → hard cut, still within limit.
	for _, c := range chunkForSlack(strings.Repeat("x", slackMaxMessageLen*2+10)) {
		assert.LessOrEqual(t, len(c), slackMaxMessageLen)
	}

	// Multibyte, no newlines → hard cut must back up to a rune boundary (em dash = 3 bytes).
	for _, c := range chunkForSlack(strings.Repeat("—", slackMaxMessageLen)) {
		assert.True(t, utf8.ValidString(c), "multibyte chunk must stay valid UTF-8")
		assert.LessOrEqual(t, len(c), slackMaxMessageLen)
	}
}

func TestChunkMarkdownForSlack(t *testing.T) {
	t.Parallel()

	assert.Nil(t, chunkMarkdownForSlack("\n\n"))
	assert.Equal(t, []string{"## Summary\nall good"}, chunkMarkdownForSlack("\n## Summary\nall good\n"))

	// A report of paragraphs, tables and code blocks with blank lines inside the code: each message
	// stays within the limit and keeps its tables and code blocks whole.
	table := "| Service | Node |\n|---|---|\n" + strings.Repeat("| ps_pmm_replication_8_4_1 | ps_pmm_replication_8_4_1 |\n", 20)
	code := "```sql\nSELECT 1;\n\nSELECT 2;\n```"
	parts := make([]string, 0, 120)
	for range 40 {
		parts = append(parts, "### Part\nSome prose about this part of the inventory.", table, code)
	}
	md := strings.Join(parts, "\n\n")
	require.Greater(t, len(md), slackMaxMarkdownLen)

	chunks := chunkMarkdownForSlack(md)
	require.Greater(t, len(chunks), 1)
	assert.Equal(t, md, strings.Join(chunks, "\n\n"), "splitting must only happen at blank lines")
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), slackMaxMarkdownLen)
		assert.Equal(t, 0, strings.Count(c, "```")%2, "code block cut in two")
		assert.True(t, strings.HasPrefix(c, "### Part") || strings.HasPrefix(c, "| Service") || strings.HasPrefix(c, "```sql"),
			"message starts mid-block: %.40s", c)
	}

	// A code block with blank lines crosses the limit → the cut goes before the block, not inside it.
	prose := strings.Repeat("prose line\n", 820)
	block := "```sql\n" + strings.Repeat("SELECT 1;\n\n", 200) + "```"
	chunks = chunkMarkdownForSlack(prose + "\n" + block)
	require.Len(t, chunks, 2)
	assert.Equal(t, strings.TrimRight(prose, "\n"), chunks[0])
	assert.Equal(t, block, chunks[1])

	// No blank line to cut at → the plain split still keeps each message within the limit.
	for _, c := range chunkMarkdownForSlack(strings.Repeat("line\n", slackMaxMarkdownLen)) {
		assert.LessOrEqual(t, len(c), slackMaxMarkdownLen)
	}
}
