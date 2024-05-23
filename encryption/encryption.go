package encryption

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

type Encryption struct {
	Path string
	Key  string
}

func New(keyPath string) (*Encryption, error) {
	e := new(Encryption)
	e.Path = keyPath

	bytes, err := os.ReadFile(e.Path)
	switch {
	case os.IsNotExist(err):
		err = e.generateKey()
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		e.Key = string(bytes)
	}

	return e, nil
}

func (e Encryption) encrypt(secret string) (string, error) {
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

func (e Encryption) decrypt(cipherText string) (string, error) {
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

func (e Encryption) getPrimitive() (tink.AEAD, error) {
	serializedKeyset, err := base64.StdEncoding.DecodeString(e.Key)
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

func (e Encryption) generateKey() error {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return err
	}

	buff := &bytes.Buffer{}
	err = insecurecleartextkeyset.Write(handle, keyset.NewBinaryWriter(buff))
	if err != nil {
		return err
	}

	e.Key = base64.StdEncoding.EncodeToString(buff.Bytes())
	return e.saveKeyToFile()
}

func (e Encryption) saveKeyToFile() error {
	return os.WriteFile(e.Path, []byte(e.Key), 0644)
}

func (e Encryption) Migrate(c *DatabaseConnection) error {
	connection := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", c.Host, c.Port, c.User, c.Password)
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	// TODO read and update for all rows in scope of 1 transcation
	tx, err := db.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query("")
	if err != nil {
		return err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		// TODO How to identify row
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
