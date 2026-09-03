// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envvars

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		input       string
		expectedErr string
	}{
		{"uppercase", "KRB5_KTNAME", ""},
		{"lowercase", "https_proxy", ""},
		{"mixed case", "Https_Proxy", ""},
		{"leading underscore", "_VAR", ""},
		{"digits", "VAR2", ""},
		{"empty", "", "cannot be empty"},
		{"leading digit", "5VAR", "invalid environment variable name"},
		{"dash", "KRB5-KTNAME", "invalid environment variable name"},
		{"assignment", "VAR=value", "invalid environment variable name"},
		{"reserved uppercase", "PMM_AGENT_SERVER_PASSWORD", "reserved for pmm-agent"},
		{"reserved lowercase", "pmm_agent_server_password", "reserved for pmm-agent"},
		{"too long", strings.Repeat("A", MaxNameLength+1), "too long"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateName(tt.input)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNormalizeNames(t *testing.T) {
	t.Parallel()

	t.Run("valid list", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNames([]string{"KRB5_KTNAME", "KRB5_CONFIG"})
		require.NoError(t, err)
		assert.Equal(t, []string{"KRB5_KTNAME", "KRB5_CONFIG"}, names)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNames(nil)
		require.NoError(t, err)
		assert.Empty(t, names)
	})

	t.Run("trims and collapses duplicates, keeping the first occurrence", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNames([]string{" KRB5_KTNAME ", "KRB5_KTNAME", "KRB5_CONFIG"})
		require.NoError(t, err)
		assert.Equal(t, []string{"KRB5_KTNAME", "KRB5_CONFIG"}, names)
	})

	t.Run("one invalid name fails the whole list", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNames([]string{"KRB5_KTNAME", "PMM_AGENT_SERVER_PASSWORD"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved for pmm-agent")
		assert.Nil(t, names)
	})

	t.Run("duplicates of the same name never hit the limit", func(t *testing.T) {
		t.Parallel()

		names := make([]string, MaxNames*2)
		for i := range names {
			names[i] = "VAR"
		}

		got, err := NormalizeNames(names)
		require.NoError(t, err)
		assert.Equal(t, []string{"VAR"}, got)
	})

	t.Run("exactly MaxNames unique names is accepted", func(t *testing.T) {
		t.Parallel()

		names := make([]string, MaxNames)
		for i := range names {
			names[i] = fmt.Sprintf("VAR%d", i)
		}

		got, err := NormalizeNames(names)
		require.NoError(t, err)
		assert.Len(t, got, MaxNames)
	})

	t.Run("more than MaxNames unique names is rejected", func(t *testing.T) {
		t.Parallel()

		names := make([]string, MaxNames+1)
		for i := range names {
			names[i] = fmt.Sprintf("VAR%d", i)
		}

		got, err := NormalizeNames(names)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many environment variable names")
		assert.Nil(t, got)
	})
}

func TestNormalizeNamesAllowing(t *testing.T) {
	t.Parallel()

	// grandfatheredSet builds the set the caller derives from an agent's currently-stored names.
	grandfatheredSet := func(names ...string) map[string]struct{} {
		set := make(map[string]struct{}, len(names))
		for _, name := range names {
			set[name] = struct{}{}
		}

		return set
	}

	t.Run("a stored name that would fail ValidateName is carried forward", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNamesAllowing(
			[]string{"KRB5-KTNAME", "KRB5_CONFIG"},
			grandfatheredSet("KRB5-KTNAME"),
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"KRB5-KTNAME", "KRB5_CONFIG"}, names)
	})

	t.Run("a name that is not stored is still validated", func(t *testing.T) {
		t.Parallel()

		names, err := NormalizeNamesAllowing(
			[]string{"KRB5-KTNAME", "KRB5_CONFIG-2"},
			grandfatheredSet("KRB5-KTNAME"),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment variable name")
		assert.Nil(t, names)
	})

	t.Run("an oversized stored list can be shrunk", func(t *testing.T) {
		t.Parallel()

		// An agent from before MaxNames existed: the owner drops one name and resends the rest.
		stored := make([]string, MaxNames+8)
		for i := range stored {
			stored[i] = fmt.Sprintf("VAR%d", i)
		}

		names, err := NormalizeNamesAllowing(stored[1:], grandfatheredSet(stored...))
		require.NoError(t, err)
		assert.Len(t, names, MaxNames+7)
	})

	t.Run("an oversized stored list cannot be grown", func(t *testing.T) {
		t.Parallel()

		stored := make([]string, MaxNames+8)
		for i := range stored {
			stored[i] = fmt.Sprintf("VAR%d", i)
		}

		names, err := NormalizeNamesAllowing(append(stored, "EXTRA"), grandfatheredSet(stored...)) //nolint:gocritic
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many environment variable names")
		assert.Nil(t, names)
	})

	t.Run("a list within MaxNames is bounded by MaxNames, not by the stored count", func(t *testing.T) {
		t.Parallel()

		names := make([]string, MaxNames+1)
		for i := range names {
			names[i] = fmt.Sprintf("VAR%d", i)
		}

		got, err := NormalizeNamesAllowing(names, grandfatheredSet("VAR0"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many environment variable names")
		assert.Nil(t, got)
	})
}
