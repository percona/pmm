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

package encryption_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/encryption"
)

func TestEncryption(t *testing.T) {
	secret := "password1"

	cipherText, err := encryption.Encrypt(secret)
	require.NoError(t, err)
	require.NotEmpty(t, cipherText)
	decryptedSecret, err := encryption.Decrypt(cipherText)
	require.NoError(t, err)
	require.Equal(t, secret, decryptedSecret)

	c := &encryption.DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "",
	}
	ctx := context.Background()

	itemsToEncrypt := []encryption.Database{
		{
			Database: "pmm-managed",
			Tables: []encryption.Table{
				{
					Table:          "agents",
					Identificators: []string{"agent_id"},
					Columns: []encryption.Column{
						{Column: "username"},
						{Column: "password"},
					},
				},
			},
		},
	}
	_, err = encryption.EncryptItems(ctx, c, itemsToEncrypt)
	require.NoError(t, err)

	itemsToDecrypt := []encryption.Database{
		{
			Database: "pmm-managed",
			Tables: []encryption.Table{
				{
					Table:          "agents",
					Identificators: []string{"agent_id"},
					Columns: []encryption.Column{
						{Column: "username"},
						{Column: "password"},
					},
				},
			},
		},
	}
	_, err = encryption.DecryptItems(ctx, c, itemsToDecrypt)
	require.NoError(t, err)
}
