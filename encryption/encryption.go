package encryption

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

type encryption struct {
	path string
	key  string
}

func New(keyPath string) (*encryption, error) {
	e := new(encryption)
	e.path = keyPath

	bytes, err := os.ReadFile(e.path)
	switch {
	case os.IsNotExist(err):
		fmt.Println("not exists")
		err = e.generateKey()
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		e.key = string(bytes)
	}

	return e, nil
}

func (e encryption) encrypt(secret string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}
	cipherText, err := primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return "", err
	}

	return string(cipherText), nil
}

func (e encryption) decrypt(cipherText string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}
	secret, err := primitive.Decrypt([]byte(cipherText), []byte(""))
	if err != nil {
		return "", err
	}

	return string(secret), nil
}

func (e encryption) getPrimitive() (tink.AEAD, error) {
	serializedKeyset, err := base64.StdEncoding.DecodeString(e.key)
	if err != nil {
		return nil, err
	}

	binaryReader := keyset.NewBinaryReader(bytes.NewBuffer(serializedKeyset))
	parsedHandle, err := insecurecleartextkeyset.Read(binaryReader)
	if err != nil {
		return nil, err
	}

	return aead.New(parsedHandle)
}

func (e encryption) generateKey() error {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return err
	}

	buff := &bytes.Buffer{}
	err = insecurecleartextkeyset.Write(handle, keyset.NewBinaryWriter(buff))
	if err != nil {
		return err
	}

	e.key = base64.StdEncoding.EncodeToString(buff.Bytes())
	return e.saveKeyToFile()
}

func (e encryption) saveKeyToFile() error {
	return os.WriteFile(e.path, []byte(e.key), 0644)
}
