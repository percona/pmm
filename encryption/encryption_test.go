package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	testPath := "/Users/jiri.ctvrtka/test.key"
	secret := "secret"

	e, err := New(testPath)
	require.NoError(t, err)
	assert.Equal(t, testPath, e.Path)
	assert.NotEmpty(t, e.Key)
	cipherText, err := e.encrypt(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, cipherText)
	decryptedSecret, err := e.decrypt(cipherText)
	require.NoError(t, err)
	assert.Equal(t, secret, decryptedSecret)

	c := &DatabaseConnection{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "pmm-agent",
		Password: "pmm-agent-password",
		EncryptedItems: []EncryptedItem{{
			Database:       "pmm-agent",
			Table:          "acc",
			Identificators: []string{"ID"},
			Columns:        []string{"username", "password"}},
		},
	}

	assert.NoError(t, e.Migrate(c))
}
