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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleHolmesMetadata is the ai_answer_end metadata shape of Holmes 0.45 for a request with several LLM calls:
// "usage" is the last call, "costs" the whole request.
const sampleHolmesMetadata = `{
  "usage": {
    "prompt_tokens": 54934,
    "completion_tokens": 4580,
    "total_tokens": 59514,
    "cached_tokens": 53083
  },
  "costs": {
    "total_cost": 0.4129714,
    "total_tokens": 212887,
    "cached_tokens": 152557,
    "prompt_tokens": 207499,
    "completion_tokens": 5388,
    "cache_creation_tokens": 54932,
    "max_prompt_tokens_per_call": 54934
  },
  "tokens": {
    "system_tokens": 10284
  }
}`

func TestParseHolmesMetadata(t *testing.T) {
	t.Parallel()

	t.Run("full metadata", func(t *testing.T) {
		t.Parallel()
		u := ParseHolmesMetadata([]byte(sampleHolmesMetadata))
		require.NotNil(t, u)
		require.NotNil(t, u.PromptTokens)
		assert.Equal(t, int32(207499), *u.PromptTokens, "tokens must cover every LLM call, not the last one")
		require.NotNil(t, u.CompletionTokens)
		assert.Equal(t, int32(5388), *u.CompletionTokens)
		require.NotNil(t, u.TotalTokens)
		assert.Equal(t, int32(212887), *u.TotalTokens)
		require.NotNil(t, u.CachedTokens)
		assert.Equal(t, int32(152557), *u.CachedTokens)
		require.NotNil(t, u.TotalCost)
		assert.InDelta(t, 0.4129714, *u.TotalCost, 0.0000001)
	})

	t.Run("costs without token counts falls back to usage", func(t *testing.T) {
		t.Parallel()
		u := ParseHolmesMetadata([]byte(`{"usage":{"total_tokens":100,"prompt_tokens":90.0},"costs":{"total_cost":0.5}}`))
		require.NotNil(t, u)
		require.NotNil(t, u.TotalTokens)
		assert.Equal(t, int32(100), *u.TotalTokens)
		require.NotNil(t, u.PromptTokens)
		assert.Equal(t, int32(90), *u.PromptTokens)
		require.NotNil(t, u.TotalCost)
		assert.InDelta(t, 0.5, *u.TotalCost, 0.0001)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, ParseHolmesMetadata(nil))
		assert.Nil(t, ParseHolmesMetadata([]byte("")))
	})

	t.Run("partial usage only", func(t *testing.T) {
		t.Parallel()
		u := ParseHolmesMetadata([]byte(`{"usage":{"total_tokens":100}}`))
		require.NotNil(t, u)
		require.NotNil(t, u.TotalTokens)
		assert.Equal(t, int32(100), *u.TotalTokens)
		assert.Nil(t, u.TotalCost)
	})
}

func TestResolveModelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "gpt-4.1", ResolveModelName("gpt-4.1", nil))
	u := ParseHolmesMetadata([]byte(`{"model":"meta-model"}`))
	assert.Equal(t, "from-req", ResolveModelName("from-req", u))
	assert.Equal(t, "meta-model", ResolveModelName("", u))
}

func TestExtractUsageFromMetadata(t *testing.T) {
	t.Parallel()
	p, c, tot := extractUsageFromMetadata([]byte(sampleHolmesMetadata))
	require.NotNil(t, p)
	require.NotNil(t, c)
	require.NotNil(t, tot)
	assert.Equal(t, int32(207499), *p)
	assert.Equal(t, int32(5388), *c)
	assert.Equal(t, int32(212887), *tot)
}
