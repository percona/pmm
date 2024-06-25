// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package encryption

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"slices"
	"sync"
)

// DefaultEncryptionKeyPath contains default PMM encryption key path.
const DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"

var (
	config    *Encryption
	configMtx sync.RWMutex
	// ErrEncryptionNotInitialized is error in case of encryption is not initialized.
	ErrEncryptionNotInitialized = errors.New("encryption is not initialized")
)

// Init initialize encryption.
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

// Encrypt returns input string encrypted.
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

// EncryptDB will encrypt all columns provided in DB connection.
func EncryptDB(ctx context.Context, c *DatabaseConnection) error {
	for _, item := range c.EncryptedItems {
		c.DBName = item.Database
		db, err := c.Connect()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck

		if len(c.EncryptedItems) == 0 {
			return errors.New("DB Connection: Database target tables/columns not defined")
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint:errcheck

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
			data := slices.Concat([]any{}, v)
			data = slices.Concat(data, res.WhereValues[k])
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

// Decrypt returns input string decrypted.
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
	secret, err := primitive.Decrypt(decoded, []byte(""))
	if err != nil {
		return cipherText, err
	}

	return string(secret), nil
}

// DecryptDB will decrypt all columns provided in DB connection.
func DecryptDB(ctx context.Context, c *DatabaseConnection) error {
	for _, item := range c.EncryptedItems {
		c.DBName = item.Database
		db, err := c.Connect()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck

		if len(c.EncryptedItems) == 0 {
			return errors.New("DB Connection: Database target tables/columns not defined")
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint:errcheck

		res, err := item.Read(tx)
		if err != nil {
			return err
		}

		for k, v := range res.SetValues {
			for i, val := range v {
				value := val.(*sql.NullString)
				if !value.Valid {
					res.SetValues[k][i] = sql.NullString{}
					continue
				}

				decrypted, err := Decrypt(value.String)
				if err != nil {
					return err
				}
				if decrypted == "" {
					res.SetValues[k][i] = sql.NullString{}
					continue
				}
				res.SetValues[k][i] = decrypted
			}
			data := slices.Concat([]any{}, v)
			data = slices.Concat(data, res.WhereValues[k])
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
