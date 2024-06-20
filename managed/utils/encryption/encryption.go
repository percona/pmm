package encryption

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"sync"
)

const DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"

var (
	config    *Encryption
	configMtx sync.RWMutex
)

func Init(keyPath string) error {
	err := create(keyPath)
	if err != nil {
		return err
	}

	return nil
}

func InitFromEnv() error {
	encryption := os.Getenv("PMM_ENCRYPTION")
	if encryption == "0" {
		return nil
	}

	keyPath := os.Getenv("PMM_ENCRYPTION_KEY")
	if keyPath == "" {
		keyPath = DefaultEncryptionKeyPath
	}

	err := create(keyPath)
	if err != nil {
		return err
	}

	return nil
}

func create(keyPath string) error {
	e := new(Encryption)
	e.Path = keyPath

	bytes, err := os.ReadFile(e.Path)
	switch {
	case os.IsNotExist(err):
		err = e.generateKey()
		if err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		e.Key = string(bytes)
	}

	primitive, err := e.getPrimitive()
	if err != nil {
		return err
	}
	e.Primitive = primitive

	configMtx.Lock()
	config = e
	configMtx.Unlock()

	return nil
}

func Encrypt(secret string) (string, error) {
	configMtx.RLock()
	primitive := config.Primitive
	configMtx.RUnlock()
	cipherText, err := primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return secret, err
	}

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func EncryptDB(ctx context.Context, c *DatabaseConnection) error {
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
				value := *val.(*string)
				_, err := base64.StdEncoding.DecodeString(value)
				if err == nil {
					continue
				}

				encrypted, err := Encrypt(value)
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

func Decrypt(cipherText string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return cipherText, err
	}

	configMtx.RLock()
	primitive := config.Primitive
	configMtx.RUnlock()
	secret, err := primitive.Decrypt([]byte(decoded), []byte(""))
	if err != nil {
		return cipherText, err
	}

	return string(secret), nil
}

func DecryptDB(ctx context.Context, c *DatabaseConnection) error {
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
				decrypted, err := Decrypt(string(decoded))
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
