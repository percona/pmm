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

	"github.com/sirupsen/logrus"
)

// DefaultEncryptionKeyPath contains default PMM encryption key path.
const DefaultEncryptionKeyPath = "/srv/pmm-encryption.key"

var (
	// ErrEncryptionNotInitialized is error in case of encryption is not initialized.
	ErrEncryptionNotInitialized = errors.New("encryption is not initialized")
)

var DefaultEncryption = New(DefaultEncryptionKeyPath)

// New create encryption, if key on path doesnt exists will be generated.
func New(keyPath string) *Encryption {
	e := new(Encryption)
	e.Path = keyPath

	bytes, err := os.ReadFile(e.Path)
	switch {
	case os.IsNotExist(err):
		err = e.generateKey()
		if err != nil {
			logrus.Errorf("Encryption: %v", err)
		}
	case err != nil:
		logrus.Errorf("Encryption: %v", err)
	default:
		e.Key = string(bytes)
	}

	primitive, err := e.getPrimitive()
	if err != nil {
		logrus.Errorf("Encryption: %v", err)
	}
	e.Primitive = primitive

	return e
}

// Encrypt is wrapper around DefaultEncryption.Encrypt.
func Encrypt(secret string) (string, error) {
	return DefaultEncryption.Encrypt(secret)
}

// Encrypt returns input string encrypted.
func (e *Encryption) Encrypt(secret string) (string, error) {
	if e == nil || e.Primitive == nil {
		return "", ErrEncryptionNotInitialized
	}

	cipherText, err := e.Primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return secret, err
	}

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// EncryptDB is wrapper around DefaultEncryption.EncryptDB.
func EncryptDB(ctx context.Context, c *DatabaseConnection) error {
	return DefaultEncryption.EncryptDB(ctx, c)
}

// EncryptDB will encrypt all columns provided in DB connection.
func (e *Encryption) EncryptDB(ctx context.Context, c *DatabaseConnection) error {
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

				encrypted, err := e.Encrypt(value)
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

// Decrypt is wrapper around DefaultEncryption.Decrypt.
func Decrypt(cipherText string) (string, error) {
	return DefaultEncryption.Decrypt(cipherText)
}

// Decrypt returns input string decrypted.
func (e *Encryption) Decrypt(cipherText string) (string, error) {
	if e == nil || e.Primitive == nil {
		return "", ErrEncryptionNotInitialized
	}

	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return cipherText, err
	}
	secret, err := e.Primitive.Decrypt(decoded, []byte(""))
	if err != nil {
		return cipherText, err
	}

	return string(secret), nil
}

// DecryptDB is wrapper around DefaultEncryption.DecryptDB.
func DecryptDB(ctx context.Context, c *DatabaseConnection) error {
	return DefaultEncryption.DecryptDB(ctx, c)
}

// DecryptDB will decrypt all columns provided in DB connection.
func (e *Encryption) DecryptDB(ctx context.Context, c *DatabaseConnection) error {
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

				decrypted, err := e.Decrypt(value.String)
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
