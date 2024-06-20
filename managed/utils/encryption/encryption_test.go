package encryption

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryption(t *testing.T) {
	secret := "password1"

	require.NoError(t, create(DefaultEncryptionKeyPath))
	cipherText, err := Encrypt(secret)
	require.NoError(t, err)
	require.NotEmpty(t, cipherText)
	decryptedSecret, err := Decrypt(cipherText)
	require.NoError(t, err)
	require.Equal(t, secret, decryptedSecret)

	c := &DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "",
		EncryptedItems: []EncryptedItem{
			{
				Database:       "pmm-managed",
				Table:          "agents",
				Identificators: []string{"agent_id"},
				Columns:        []string{"username", "password"},
			},
		},
	}

	ctx := context.Background()
	require.NoError(t, EncryptDB(ctx, c))
	require.NoError(t, DecryptDB(ctx, c))
}
