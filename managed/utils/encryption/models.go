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

import "github.com/google/tink/go/tink"

// Encryption contains fields required for encryption.
type Encryption struct {
	Path      string
	Key       string
	Primitive tink.AEAD
}

// DatabaseConnection represents DB connection and it's encrypted items.
type DatabaseConnection struct {
	Host, User, Password string
	Port                 int16
	DBName               string
	SSLMode              string
	SSLCAPath            string
	SSLKeyPath           string
	SSLCertPath          string
}

// EncryptedItem resresents DB name, table, encrypted columns and it's identificators.
type Database struct {
	Database string
	Tables   []Table
}

type Table struct {
	Table          string
	Identificators []string
	Columns        []Column
}

type Column struct {
	Column        string
	CustomHandler func(e *Encryption, val any) (any, error)
}

// QueryValues represents query to update row after encrypt/decrypt.
type QueryValues struct {
	Query       string
	SetValues   [][]any
	WhereValues [][]any
}
