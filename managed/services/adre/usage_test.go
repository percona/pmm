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

const sampleHolmesMetadata = `{
  "usage": {
    "prompt_tokens": 22904,
    "completion_tokens": 21,
    "total_tokens": 22925,
    "cached_tokens": 21120
  },
  "costs": {
    "total_cost": 0.014,
    "prompt_tokens": 0.003,
    "completion_tokens": 0.001,
    "cached_tokens": 0.002
  },
  "tokens": {
    "system_tokens": 22000
  }
}`

func TestParseHolmesMetadata(t *testing.T) {
	t.Parallel()

	t.Run("full metadata", func(t *testing.T) {
		t.Parallel()
		u := ParseHolmesMetadata([]byte(sampleHolmesMetadata))
		require.NotNil(t, u)
		require.NotNil(t, u.PromptTokens)
		assert.Equal(t, int32(22904), *u.PromptTokens)
		require.NotNil(t, u.CompletionTokens)
		assert.Equal(t, int32(21), *u.CompletionTokens)
		require.NotNil(t, u.TotalTokens)
		assert.Equal(t, int32(22925), *u.TotalTokens)
		require.NotNil(t, u.CachedTokens)
		assert.Equal(t, int32(21120), *u.CachedTokens)
		require.NotNil(t, u.TotalCost)
		assert.InDelta(t, 0.014, *u.TotalCost, 0.0001)
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
	assert.Equal(t, int32(22904), *p)
	assert.Equal(t, int32(21), *c)
	assert.Equal(t, int32(22925), *tot)
}
