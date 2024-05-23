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
	assert.Equal(t, testPath, e.path)
	assert.NotEmpty(t, e.key)
	cipherText, err := e.encrypt(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, cipherText)
	decryptedSecret, err := e.decrypt(cipherText)
	require.NoError(t, err)
	assert.Equal(t, secret, decryptedSecret)
}
