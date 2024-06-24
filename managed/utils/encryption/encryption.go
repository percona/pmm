package encryption

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"sync"
)

const DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"

var (
	config                      *Encryption
	configMtx                   sync.RWMutex
	ErrEncryptionNotInitialized = errors.New("encryption is not initialized")
)

func Init(keyPath string) error {
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
	if config == nil {
		configMtx.RUnlock()
		return "", ErrEncryptionNotInitialized
	}
	primitive := config.Primitive
	configMtx.RUnlock()

	cipherText, err := primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return secret, err
	}

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func EncryptDB(ctx context.Context, c *DatabaseConnection) error {
	for _, item := range c.EncryptedItems {
		c.DBName = item.Database
		db, err := c.Connect()
		if err != nil {
			return err
		}
		defer db.Close()

		if len(c.EncryptedItems) == 0 {
			return errors.New("DB Connection: Database target tables/columns not defined")
		}

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
				var value string
				if v, ok := val.(*sql.NullString); ok {
					value = v.String
				}

				if value != "" {
					_, err := base64.StdEncoding.DecodeString(value)
					if err == nil {
						continue
					}
				}

				encrypted, err := Encrypt(value)
				if err != nil {
					return err
				}
				res.SetValues[k][i] = encrypted
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
	configMtx.RLock()
	if config == nil {
		configMtx.RUnlock()
		return "", ErrEncryptionNotInitialized
	}
	primitive := config.Primitive
	configMtx.RUnlock()

	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return cipherText, err
	}
	secret, err := primitive.Decrypt([]byte(decoded), []byte(""))
	if err != nil {
		return cipherText, err
	}

	return string(secret), nil
}

func DecryptDB(ctx context.Context, c *DatabaseConnection) error {
	for _, item := range c.EncryptedItems {
		c.DBName = item.Database
		db, err := c.Connect()
		if err != nil {
			return err
		}
		defer db.Close()

		if len(c.EncryptedItems) == 0 {
			return errors.New("DB Connection: Database target tables/columns not defined")
		}

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
				value := val.(*sql.NullString)
				if !value.Valid {
					res.SetValues[k][i] = ""
					continue
				}

				decrypted, err := Decrypt(value.String)
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
