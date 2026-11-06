// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginate(t *testing.T) {
	t.Parallel()

	items := func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate([]int{}, 0, 0, 50, 200)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, page{Total: 0, Offset: 0, Limit: 50, Truncated: false}, p)
		assert.Empty(t, p.marker())
	})

	t.Run("ExactlyLimit", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate(items(50), 0, 0, 50, 200)
		require.NoError(t, err)
		assert.Len(t, got, 50)
		assert.False(t, p.Truncated)
		assert.Empty(t, p.marker())
	})

	t.Run("OneOver", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate(items(51), 0, 0, 50, 200)
		require.NoError(t, err)
		assert.Len(t, got, 50)
		assert.Equal(t, page{Total: 51, Offset: 0, Limit: 50, Truncated: true}, p)
		assert.Equal(t, "showing 1–50 of 51; pass offset=50 for more", p.marker())
	})

	t.Run("SecondPage", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate(items(51), 50, 0, 50, 200)
		require.NoError(t, err)
		assert.Equal(t, []int{50}, got)
		assert.False(t, p.Truncated)
	})

	t.Run("OffsetPastEnd", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate(items(3), 10, 0, 50, 200)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, page{Total: 3, Offset: 10, Limit: 50, Truncated: false}, p)
	})

	t.Run("ExplicitLimit", func(t *testing.T) {
		t.Parallel()

		got, p, err := paginate(items(10), 2, 3, 50, 200)
		require.NoError(t, err)
		assert.Equal(t, []int{2, 3, 4}, got)
		assert.Equal(t, 3, p.Limit)
		assert.Equal(t, "showing 3–5 of 10; pass offset=5 for more", p.marker())
	})

	for name, tc := range map[string]struct{ offset, limit int }{
		"OverMax":        {0, 201},
		"NegativeLimit":  {0, -1},
		"NegativeOffset": {-1, 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, _, err := paginate(items(5), tc.offset, tc.limit, 50, 200)
			require.Error(t, err)
			assert.Equal(t, codeInvalidInput, mapError(err).code)
		})
	}
}

func TestCapText(t *testing.T) {
	t.Parallel()

	got, cut := capText("hello", 5)
	assert.Equal(t, "hello", got)
	assert.False(t, cut)

	got, cut = capText("hello world", 5)
	assert.Equal(t, "hello", got)
	assert.True(t, cut)

	got, cut = capText("aé", 2)
	assert.Equal(t, "a", got)
	assert.True(t, cut)

	got, cut = capText(strings.Repeat("日", 10), 10)
	assert.Equal(t, strings.Repeat("日", 3), got)
	assert.True(t, cut)
}

func TestPartErrorOf(t *testing.T) {
	t.Parallel()

	pe := partErrorOf(newToolError(codeNotFound, "gone"))
	require.NotNil(t, pe)
	assert.Equal(t, codeNotFound, pe.Code)
	assert.Equal(t, "gone", pe.Message)

	pe = partErrorOf(assert.AnError)
	assert.Equal(t, codePMMUnavailable, pe.Code)
}

func TestResultTypesJSON(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	to := from.Add(time.Hour)

	b, err := json.Marshal(window{From: from, To: to})
	require.NoError(t, err)
	assert.JSONEq(t, `{"period_from":"2026-01-02T03:04:05Z","period_to":"2026-01-02T04:04:05Z"}`, string(b))

	b, err = json.Marshal(queryRef{QueryID: "Q1", PeriodFrom: from, PeriodTo: to})
	require.NoError(t, err)
	assert.JSONEq(t, `{"queryid":"Q1","period_from":"2026-01-02T03:04:05Z","period_to":"2026-01-02T04:04:05Z"}`, string(b))
}
