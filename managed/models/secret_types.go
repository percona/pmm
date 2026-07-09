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

package models

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/percona/pmm/managed/utils/encryption"
)

// EncryptedString is a string that is transparently encrypted before it
// reaches the database driver and decrypted when scanned back, so every code
// path that persists it produces ciphertext. In memory it always holds
// plaintext; read it with Reveal. It renders as "***" in logs and struct
// dumps.
type EncryptedString string

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (s EncryptedString) Value() (driver.Value, error) {
	c, err := encryption.DefaultCipher()
	if err != nil {
		return nil, err
	}

	stored, err := c.Encrypt(string(s))
	if err != nil {
		return nil, err
	}

	return stored, nil
}

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (s *EncryptedString) Scan(src any) error {
	var stored string
	switch v := src.(type) {
	case nil:
		*s = ""
		return nil
	case string:
		stored = v
	case []byte:
		stored = string(v)
	default:
		return fmt.Errorf("expected string or []byte, got %T", src)
	}

	c, err := encryption.DefaultCipher()
	if err != nil {
		return err
	}
	plaintext, err := c.Decrypt(stored)
	if err != nil {
		return err
	}
	*s = EncryptedString(plaintext)

	return nil
}

// String implements fmt.Stringer; the secret is redacted.
func (s EncryptedString) String() string {
	return "***"
}

// GoString implements fmt.GoStringer; the secret is redacted.
func (s EncryptedString) GoString() string {
	return "***"
}

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *EncryptedString) Reveal() string {
	if s == nil {
		return ""
	}

	return string(*s)
}

// EncryptedStringOrNil returns s as *EncryptedString, or nil if s is empty.
func EncryptedStringOrNil(s string) *EncryptedString {
	if s == "" {
		return nil
	}
	es := EncryptedString(s)

	return &es
}

// encryptTag marks string fields of options structs that hold secrets;
// encryptedJSONValue and encryptedJSONScan transform exactly those fields.
const encryptTag = "encrypt"

func applyToSecretFields(v any, fn func(string) (string, error)) error {
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Tag.Get(encryptTag) != "true" {
			continue
		}
		if field.Type.Kind() != reflect.String {
			return fmt.Errorf("field %s.%s is tagged %s but is not a string", typ.Name(), field.Name, encryptTag)
		}

		transformed, err := fn(val.Field(i).String())
		if err != nil {
			return fmt.Errorf("field %s.%s: %w", typ.Name(), field.Name, err)
		}
		val.Field(i).SetString(transformed)
	}

	return nil
}

// encryptedJSONValue marshals v to JSON with its tagged secret fields
// encrypted; the caller's struct is never mutated.
func encryptedJSONValue[T any](v T) (driver.Value, error) {
	c, err := encryption.DefaultCipher()
	if err != nil {
		return nil, err
	}

	cp := v
	err = applyToSecretFields(&cp, c.Encrypt)
	if err != nil {
		return nil, err
	}

	return jsonValue(cp)
}

// encryptedJSONScan unmarshals a JSON column into v and decrypts its tagged
// secret fields.
func encryptedJSONScan(v, src any) error {
	if src == nil {
		return jsonScan(v, src)
	}

	err := jsonScan(v, src)
	if err != nil {
		return err
	}

	c, err := encryption.DefaultCipher()
	if err != nil {
		return err
	}

	return applyToSecretFields(v, c.Decrypt)
}
