package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

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

func (e Encryption) Encrypt(secret string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}
	cipherText, err := primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func (e Encryption) EncryptDB(ctx context.Context, c *DatabaseConnection) error {
	db, err := c.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	if len(c.EncryptedItems) == 0 {
		return errors.New("DB Connection: Database target tables/columns not defined")
	}
	for _, item := range c.EncryptedItems {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		res, err := item.Read(tx)
		if err != nil {
			return err
		}

		for k, v := range res.SetValues {
			for i, val := range v {
				encrypted, err := e.Encrypt(*val.(*string))
				if err != nil {
					return err
				}
				res.SetValues[k][i] = base64.StdEncoding.EncodeToString([]byte(encrypted))
			}
			data := append([]any{}, v...)
			data = append(data, res.WhereValues[k]...)
			_, err := tx.Exec(res.Query, data...)
			if err != nil {
				return err
			}
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

func (e Encryption) Decrypt(cipherText string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	secret, err := primitive.Decrypt([]byte(decoded), []byte(""))
	if err != nil {
		return "", err
	}

	return string(secret), nil
}

func (e Encryption) DecryptDB(ctx context.Context, c *DatabaseConnection) error {
	db, err := c.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	if len(c.EncryptedItems) == 0 {
		return errors.New("DB Connection: Database target tables/columns not defined")
	}

	for _, item := range c.EncryptedItems {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		res, err := item.Read(tx)
		if err != nil {
			return err
		}

		for k, v := range res.SetValues {
			for i, val := range v {
				decoded, err := base64.StdEncoding.DecodeString(*val.(*string))
				if err != nil {
					return err
				}
				decrypted, err := e.Decrypt(string(decoded))
				if err != nil {
					return err
				}
				res.SetValues[k][i] = decrypted
			}
			data := append([]any{}, v...)
			data = append(data, res.WhereValues[k]...)
			_, err := tx.Exec(res.Query, data...)
			if err != nil {
				return err
			}
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
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
	return os.WriteFile(e.Path, []byte(e.Key), 0o644)
}
