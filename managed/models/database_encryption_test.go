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

func TestRecordedEncryptedItems(t *testing.T) {
	t.Parallel()

	agents := []string{"pmm-managed.agents.username", "pmm-managed.agents.password"}
	bootstrap := []string{"pmm-managed.om_bootstrap_secrets.mongodb_password"}

	t.Run("a fresh database records what it just encrypted", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, append(append([]string{}, agents...), bootstrap...),
			recordedEncryptedItems(nil, append(append([]string{}, agents...), bootstrap...), true))
	})

	t.Run("an upgrade keeps what earlier passes encrypted", func(t *testing.T) {
		t.Parallel()

		// The regression this guards: a second table meant only its own columns were
		// prepared, and writing that list alone left agents.* unrecorded. The next
		// start re-encrypted the already-encrypted values, one layer per restart,
		// until the application read back ciphertext.
		after := recordedEncryptedItems(agents, bootstrap, true)

		assert.Equal(t, append(append([]string{}, agents...), bootstrap...), after)
	})

	t.Run("nothing to do leaves the record as it was", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, agents, recordedEncryptedItems(agents, nil, true))
	})

	t.Run("decrypting drops only the columns it decrypted", func(t *testing.T) {
		t.Parallel()

		recorded := append(append([]string{}, agents...), bootstrap...)

		assert.Equal(t, agents, recordedEncryptedItems(recorded, bootstrap, false))
	})

	t.Run("decrypting everything empties the record", func(t *testing.T) {
		t.Parallel()

		recorded := append(append([]string{}, agents...), bootstrap...)

		assert.Empty(t, recordedEncryptedItems(recorded, recorded, false))
	})
}
