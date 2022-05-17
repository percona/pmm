package rsa_encryptor

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed test.key
var privateKey []byte

//go:embed test.pub
var publicKey []byte

func TestDecryptEmbeddedWithFormat(t *testing.T) {
	key := "key1"
	sut, _ := NewFromPrivateKey(key, privateKey)

	t.Run("Happy path", func(t *testing.T) {
		text := "abc"
		expectedText := fmt.Sprintf("--arg=\"%s\"", text)
		encrypted, err := sut.EncryptBase32([]byte(text))
		assert.Nil(t, err)
		block, err := sut.EncodeCipherBlock(encrypted)
		assert.Nil(t, err)
		textWithEmbeddedBlock := fmt.Sprintf("--arg=%s", block)

		fmt.Println(textWithEmbeddedBlock)

		actualTextDecrypted, err := sut.DecryptEmbeddedWithFormat(textWithEmbeddedBlock, func(str string) string {
			return fmt.Sprintf("\"%s\"", str)
		})
		assert.Nil(t, err)
		assert.Equal(t, expectedText, actualTextDecrypted)
	})
}

func TestDecodeCipherBlock(t *testing.T) {
	key := "key1"
	ciphertext := "abc123"
	sut, _ := NewFromPrivateKey(key, privateKey)

	t.Run("Happy path", func(t *testing.T) {
		block := EncryptedTextBlockStart + EncryptedTextBlockParamsDelimiter + key + EncryptedTextBlockCipherStart +
			ciphertext + EncryptedTextBlockCipherEnd
		text, err := sut.DecodeCipherBlock(block)
		assert.Nil(t, err)
		assert.NotNil(t, text)
		assert.Equal(t, text.key, key)
		assert.Equal(t, text.ciphertext, ciphertext)
	})

	t.Run("Invalid prefix", func(t *testing.T) {
		invalidBlock := "x123" + EncryptedTextBlockParamsDelimiter + key + EncryptedTextBlockCipherStart +
			ciphertext + EncryptedTextBlockCipherEnd
		_, err := sut.DecodeCipherBlock(invalidBlock)
		assert.NotNil(t, err)
	})
}

func TestEncodeCipherBlock(t *testing.T) {
	key := "key1"
	sut, _ := NewFromPrivateKey(key, privateKey)

	ciphertext := "xyz123"
	expectedCipherBlock := EncryptedTextBlockStart + EncryptedTextBlockParamsDelimiter +
		key + EncryptedTextBlockCipherStart + ciphertext + EncryptedTextBlockCipherEnd

	t.Run("Happy path", func(t *testing.T) {
		cipherBlock, err := sut.EncodeCipherBlock([]byte(ciphertext))
		assert.Nil(t, err)
		assert.Equal(t, cipherBlock, expectedCipherBlock)
	})
	t.Run("Should not allow EncryptedTextBlockStart", func(t *testing.T) {
		_, err := sut.EncodeCipherBlock([]byte(EncryptedTextBlockStart))
		assert.NotNil(t, err)
	})
}
