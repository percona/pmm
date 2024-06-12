package encryption

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryption(t *testing.T) {
	secret := "password1"

	err := create(DefaultEncryptionKeyPath)
	require.NoError(t, err)
	cipherText, err := Encrypt(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, cipherText)
	decryptedSecret, err := Decrypt(cipherText)
	require.NoError(t, err)
	assert.Equal(t, secret, decryptedSecret)

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
	assert.NoError(t, EncryptDB(ctx, c))
	assert.NoError(t, DecryptDB(ctx, c))
}
