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
	"strings"

	"github.com/percona/pmm/managed/utils/encryption"
)

// redacted is how an encrypted column type renders in logs and struct dumps.
const redacted = "***"

// encryptedValue encrypts plaintext bound to column, so that it does not decrypt in another one.
func encryptedValue(plaintext, column string) (driver.Value, error) {
	c, err := encryption.DefaultCipher()
	if err != nil {
		return nil, err
	}

	stored, err := c.Encrypt(plaintext, []byte(column))
	if err != nil {
		return nil, err
	}

	return stored, nil
}

// scanEncrypted decrypts a stored value of column into dst, for Scan of an encrypted column type.
func scanEncrypted(dst *string, src any, column string) error {
	var stored string
	switch v := src.(type) {
	case nil:
		*dst = ""
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
	plaintext, err := c.Decrypt(stored, []byte(column))
	if err != nil {
		return err
	}
	*dst = plaintext

	return nil
}

// reveal returns the plaintext s holds; nil yields "".
func reveal[T ~string](s *T) string {
	if s == nil {
		return ""
	}

	return string(*s)
}

// EncryptedStringOrNil returns s as *T, or nil if s is empty.
func EncryptedStringOrNil[T ~string](s string) *T {
	if s == "" {
		return nil
	}
	es := T(s)

	return &es
}

// AgentUsername is agents.username, encrypted at rest.
type AgentUsername string

// Value implements database/sql/driver.Valuer interface.
func (s AgentUsername) Value() (driver.Value, error) {
	return encryptedValue(string(s), "agents.username")
}

// Scan implements database/sql.Scanner interface.
func (s *AgentUsername) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "agents.username")
}

// String implements fmt.Stringer; the secret is redacted.
func (AgentUsername) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (AgentUsername) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *AgentUsername) Reveal() string { return reveal(s) }

// AgentPassword is agents.password, encrypted at rest.
type AgentPassword string

// Value implements database/sql/driver.Valuer interface.
func (s AgentPassword) Value() (driver.Value, error) {
	return encryptedValue(string(s), "agents.password")
}

// Scan implements database/sql.Scanner interface.
func (s *AgentPassword) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "agents.password")
}

// String implements fmt.Stringer; the secret is redacted.
func (AgentPassword) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (AgentPassword) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *AgentPassword) Reveal() string { return reveal(s) }

// AgentAgentPassword is agents.agent_password, encrypted at rest.
type AgentAgentPassword string

// Value implements database/sql/driver.Valuer interface.
func (s AgentAgentPassword) Value() (driver.Value, error) {
	return encryptedValue(string(s), "agents.agent_password")
}

// Scan implements database/sql.Scanner interface.
func (s *AgentAgentPassword) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "agents.agent_password")
}

// String implements fmt.Stringer; the secret is redacted.
func (AgentAgentPassword) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (AgentAgentPassword) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *AgentAgentPassword) Reveal() string { return reveal(s) }

// OmMongoDBUsername is om_bootstrap_secrets.mongodb_username, encrypted at rest.
type OmMongoDBUsername string

// Value implements database/sql/driver.Valuer interface.
func (s OmMongoDBUsername) Value() (driver.Value, error) {
	return encryptedValue(string(s), "om_bootstrap_secrets.mongodb_username")
}

// Scan implements database/sql.Scanner interface.
func (s *OmMongoDBUsername) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "om_bootstrap_secrets.mongodb_username")
}

// String implements fmt.Stringer; the secret is redacted.
func (OmMongoDBUsername) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (OmMongoDBUsername) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *OmMongoDBUsername) Reveal() string { return reveal(s) }

// OmMongoDBPassword is om_bootstrap_secrets.mongodb_password, encrypted at rest.
type OmMongoDBPassword string

// Value implements database/sql/driver.Valuer interface.
func (s OmMongoDBPassword) Value() (driver.Value, error) {
	return encryptedValue(string(s), "om_bootstrap_secrets.mongodb_password")
}

// Scan implements database/sql.Scanner interface.
func (s *OmMongoDBPassword) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "om_bootstrap_secrets.mongodb_password")
}

// String implements fmt.Stringer; the secret is redacted.
func (OmMongoDBPassword) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (OmMongoDBPassword) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *OmMongoDBPassword) Reveal() string { return reveal(s) }

// OmKeyFile is om_bootstrap_secrets.key_file, encrypted at rest.
type OmKeyFile string

// Value implements database/sql/driver.Valuer interface.
func (s OmKeyFile) Value() (driver.Value, error) {
	return encryptedValue(string(s), "om_bootstrap_secrets.key_file")
}

// Scan implements database/sql.Scanner interface.
func (s *OmKeyFile) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "om_bootstrap_secrets.key_file")
}

// String implements fmt.Stringer; the secret is redacted.
func (OmKeyFile) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (OmKeyFile) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *OmKeyFile) Reveal() string { return reveal(s) }

// OmAdminUsername is om_bootstrap_secrets.admin_username, encrypted at rest.
type OmAdminUsername string

// Value implements database/sql/driver.Valuer interface.
func (s OmAdminUsername) Value() (driver.Value, error) {
	return encryptedValue(string(s), "om_bootstrap_secrets.admin_username")
}

// Scan implements database/sql.Scanner interface.
func (s *OmAdminUsername) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "om_bootstrap_secrets.admin_username")
}

// String implements fmt.Stringer; the secret is redacted.
func (OmAdminUsername) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (OmAdminUsername) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *OmAdminUsername) Reveal() string { return reveal(s) }

// OmAdminPassword is om_bootstrap_secrets.admin_password, encrypted at rest.
type OmAdminPassword string

// Value implements database/sql/driver.Valuer interface.
func (s OmAdminPassword) Value() (driver.Value, error) {
	return encryptedValue(string(s), "om_bootstrap_secrets.admin_password")
}

// Scan implements database/sql.Scanner interface.
func (s *OmAdminPassword) Scan(src any) error {
	return scanEncrypted((*string)(s), src, "om_bootstrap_secrets.admin_password")
}

// String implements fmt.Stringer; the secret is redacted.
func (OmAdminPassword) String() string { return redacted }

// GoString implements fmt.GoStringer; the secret is redacted.
func (OmAdminPassword) GoString() string { return redacted }

// Reveal returns the plaintext; safe to call on nil, which yields "".
func (s *OmAdminPassword) Reveal() string { return reveal(s) }

// optionAssociatedData binds a secret field of an options blob to its column and JSON key.
func optionAssociatedData(column, field string) []byte {
	return []byte(column + "." + field)
}

// encryptTag marks string fields of options structs that hold secrets;
// encryptedJSONValue and encryptedJSONScan transform exactly those fields.
const encryptTag = "encrypt"

// applyToSecretFields replaces each tagged secret field of the struct v points
// to with what fn returns for its JSON key and value.
func applyToSecretFields(v any, fn func(field, value string) (string, error)) error {
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

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" {
			name = field.Name
		}
		transformed, err := fn(name, val.Field(i).String())
		if err != nil {
			return fmt.Errorf("field %s.%s: %w", typ.Name(), field.Name, err)
		}
		val.Field(i).SetString(transformed)
	}

	return nil
}

// redactSecrets masks the tagged secret fields of the struct v points to, for
// String and GoString: log lines and struct dumps must not carry secrets.
func redactSecrets(v any) {
	_ = applyToSecretFields(v, func(_, s string) (string, error) {
		if s == "" {
			return "", nil
		}
		return "***", nil
	})
}

// encryptedJSONValue marshals v, stored in column, to JSON with its tagged
// secret fields encrypted; the caller's struct is never mutated.
func encryptedJSONValue[T any](v T, column string) (driver.Value, error) {
	c, err := encryption.DefaultCipher()
	if err != nil {
		return nil, err
	}

	cp := v
	err = applyToSecretFields(&cp, func(field, s string) (string, error) {
		return c.Encrypt(s, optionAssociatedData(column, field))
	})
	if err != nil {
		return nil, err
	}

	return jsonValue(cp)
}

// encryptedJSONScan unmarshals a JSON column into v and decrypts its tagged
// secret fields.
func encryptedJSONScan(v, src any, column string) error {
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

	return applyToSecretFields(v, func(field, s string) (string, error) {
		return c.Decrypt(s, optionAssociatedData(column, field))
	})
}
