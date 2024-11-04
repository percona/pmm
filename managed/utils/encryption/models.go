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

// Table represents table name, it's identifiers and columns to be encrypted/decrypted.
type Table struct {
	Name        string
	Identifiers []string
	Columns     []Column
}

// Column represents column name and column's custom handler (if needed).
type Column struct {
	Name          string
	CustomHandler func(e *Encryption, val any) (any, error)
}

// QueryValues represents query to update row after encrypt/decrypt.
type QueryValues struct {
	Query       string
	SetValues   [][]any
	WhereValues [][]any
}
