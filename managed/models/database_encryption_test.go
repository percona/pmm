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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMergeEncryptedItems guards the fix that makes adding a table to DefaultAgentEncryptionColumnsV3
// safe: the tracked EncryptedItems set must keep already-encrypted columns (e.g. agents) when only a
// newly-added table's columns are processed. A wholesale replace would drop agents and re-encrypt
// (corrupt) them on the next startup.
func TestMergeEncryptedItems(t *testing.T) {
	t.Parallel()

	t.Run("encrypt keeps existing and adds new (the upgrade case)", func(t *testing.T) {
		t.Parallel()
		got := mergeEncryptedItems(
			[]string{"pmm.agents.username", "pmm.agents.password"},                      // already encrypted
			[]string{"pmm.adre_models.api_key", "pmm.adre_provisioning.holmes_api_key"}, // just encrypted
			true,
		)
		assert.Equal(t, []string{
			"pmm.adre_models.api_key",
			"pmm.adre_provisioning.holmes_api_key",
			"pmm.agents.password",
			"pmm.agents.username",
		}, got)
	})

	t.Run("encrypt is idempotent (no duplicates)", func(t *testing.T) {
		t.Parallel()
		got := mergeEncryptedItems(
			[]string{"pmm.agents.username"},
			[]string{"pmm.agents.username"},
			true,
		)
		assert.Equal(t, []string{"pmm.agents.username"}, got)
	})

	t.Run("decrypt removes processed columns, keeps the rest", func(t *testing.T) {
		t.Parallel()
		got := mergeEncryptedItems(
			[]string{"pmm.agents.username", "pmm.adre_models.api_key"},
			[]string{"pmm.adre_models.api_key"},
			false,
		)
		assert.Equal(t, []string{"pmm.agents.username"}, got)
	})

	t.Run("empty inputs yield empty set", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, mergeEncryptedItems(nil, nil, true))
	})
}
