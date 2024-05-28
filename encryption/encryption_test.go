package encryption

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryption(t *testing.T) {
	testPath := "/srv/pmm-encryption.key"
	secret := "password1"

	e, err := New(testPath)
	require.NoError(t, err)
	assert.Equal(t, testPath, e.Path)
	assert.NotEmpty(t, e.Key)
	cipherText, err := e.Encrypt(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, cipherText)
	decryptedSecret, err := e.Decrypt(cipherText)
	require.NoError(t, err)
	assert.Equal(t, secret, decryptedSecret)

	c := &DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "pmm-agent",
		Password: "pmm-agent-password",
		EncryptedItems: []EncryptedItem{
			{
				Database:       "pmm-agent",
				Table:          "acc",
				Identificators: []string{"ID"},
				Columns:        []string{"username", "password"},
			},
		},
	}

	ctx := context.Background()
	assert.NoError(t, e.EncryptDB(ctx, c))
	assert.NoError(t, e.DecryptDB(ctx, c))
}
