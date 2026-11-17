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
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripHolmesEmbeds(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, in, want string
	}{
		{
			name: "markdown paragraph",
			in:   "## Member state\n\n<< {\"type\": \"promql\", \"tool_name\": \"execute_prometheus_range_query\", \"tool_call_id\": \"toolu_01NJ\"} >>\n\n| a | b |",
			want: "## Member state\n\n\n\n| a | b |",
		},
		{
			name: "two embeds inline",
			in:   `x << {"type": "promql", "tool_call_id": "1"} >> y <<{"type":"promql","tool_call_id":"2"}>> z`,
			want: "x  y  z",
		},
		{
			name: "not an embed",
			in:   "shift: 1 << 2; template << {\"a\": 1} >> stays",
			want: "shift: 1 << 2; template << {\"a\": 1} >> stays",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, stripHolmesEmbeds(tc.in))
		})
	}

	t.Run("JSON-escaped in an SSE payload", func(t *testing.T) {
		t.Parallel()
		// Holmes (Python json.dumps) keeps < and > literal and escapes the quotes.
		payload := `{"analysis": "Lag stayed under 4 s.\n\n<< {\"type\": \"promql\", \"tool_name\": \"execute_prometheus_range_query\", \"tool_call_id\": \"toolu_01Pd\"} >>"}`

		var out map[string]string
		require.NoError(t, json.Unmarshal([]byte(stripHolmesEmbeds(payload)), &out))
		assert.Equal(t, "Lag stayed under 4 s.\n\n", out["analysis"])
	})
}

func TestEmbedStripReader(t *testing.T) {
	t.Parallel()

	in := "event: ai_answer_end\n" +
		`data: {"analysis": "## Summary\n\n<< {\"type\": \"promql\", \"tool_call_id\": \"t1\"} >>\nDone"}` + "\n" +
		"\n" +
		"data: tail without newline"
	want := "event: ai_answer_end\n" +
		`data: {"analysis": "## Summary\n\n\nDone"}` + "\n" +
		"\n" +
		"data: tail without newline"

	r := newEmbedStripReader(io.NopCloser(iotest.OneByteReader(strings.NewReader(in))))
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
	require.NoError(t, r.Close())
}
