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
