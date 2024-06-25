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
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

func prepareRowPointers(rows *sql.Rows) ([]any, error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	columns := make(map[string]string)
	for _, columnType := range columnTypes {
		columns[columnType.Name()] = columnType.DatabaseTypeName()
	}

	row := []any{}
	for _, t := range columns {
		switch t {
		case "VARCHAR":
			row = append(row, new(sql.NullString))
		default:
			// TODO support more identificators types
			return nil, fmt.Errorf("unsupported identificator type %s", t)
		}
	}

	return row, nil
}

func (e *Encryption) getPrimitive() (tink.AEAD, error) {
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

func (e *Encryption) generateKey() error {
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

func (e *Encryption) saveKeyToFile() error {
	return os.WriteFile(e.Path, []byte(e.Key), 0o644)
}
