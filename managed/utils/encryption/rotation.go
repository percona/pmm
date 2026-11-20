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

package encryption

import (
	"fmt"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
)

// AddNewPrimaryKey adds a new AES-256-GCM key to the keyset, makes it the
// primary and persists the updated keyset via the provider. Existing keys
// stay in the keyset, so previously encrypted values remain decryptable.
func AddNewPrimaryKey(p KeyProvider) (uint32, error) {
	handle, err := p.Load()
	if err != nil {
		return 0, err
	}

	manager := keyset.NewManagerFromHandle(handle)
	keyID, err := manager.Add(aead.AES256GCMKeyTemplate())
	if err != nil {
		return 0, fmt.Errorf("failed to add encryption key: %w", err)
	}
	err = manager.SetPrimary(keyID)
	if err != nil {
		return 0, fmt.Errorf("failed to promote encryption key: %w", err)
	}

	updated, err := manager.Handle()
	if err != nil {
		return 0, fmt.Errorf("failed to update encryption keyset: %w", err)
	}
	err = p.Store(updated)
	if err != nil {
		return 0, err
	}

	return keyID, nil
}

// PruneRetiredKeys deletes all non-primary keys from the keyset and persists
// it, returning the removed key IDs. Call it only after verifying that no
// stored value still references a retired key.
func PruneRetiredKeys(p KeyProvider) ([]uint32, error) {
	handle, err := p.Load()
	if err != nil {
		return nil, err
	}
	primary, err := handle.Primary()
	if err != nil {
		return nil, fmt.Errorf("failed to determine primary encryption key: %w", err)
	}

	var retired []uint32
	for i := range handle.Len() {
		entry, err := handle.Entry(i)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption keyset entry: %w", err)
		}
		if entry.KeyID() != primary.KeyID() {
			retired = append(retired, entry.KeyID())
		}
	}
	if len(retired) == 0 {
		return nil, nil
	}

	manager := keyset.NewManagerFromHandle(handle)
	for _, id := range retired {
		err = manager.Delete(id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete encryption key %d: %w", id, err)
		}
	}

	updated, err := manager.Handle()
	if err != nil {
		return nil, fmt.Errorf("failed to update encryption keyset: %w", err)
	}
	err = p.Store(updated)
	if err != nil {
		return nil, err
	}

	return retired, nil
}
