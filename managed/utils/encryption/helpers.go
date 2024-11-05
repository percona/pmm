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
	"slices"
	"strings"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
	"gopkg.in/reform.v1"
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
		case "VARCHAR", "JSONB":
			row = append(row, &sql.NullString{})
		default:
			return nil, fmt.Errorf("unsupported identificator type %s", t)
		}
	}

	return row, nil
}

func encryptColumnStringHandler(e *Encryption, val any) (any, error) {
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return sql.NullString{}, nil
	}

	encrypted, err := e.Encrypt(value.String)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

func decryptColumnStringHandler(e *Encryption, val any) (any, error) {
	value := val.(*sql.NullString) //nolint:forcetypeassert
	if !value.Valid {
		return nil, nil //nolint:nilnil
	}

	decrypted, err := e.Decrypt(value.String)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

func (e *Encryption) getPrimitive() (tink.AEAD, error) { //nolint:ireturn
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
	return os.WriteFile(e.Path, []byte(e.Key), 0o644) //nolint:gosec
}

func (table Table) columnsList() []string {
	res := []string{}
	for _, c := range table.Columns {
		res = append(res, c.Name)
	}

	return res
}

func (table Table) read(tx *reform.TX) (*QueryValues, error) {
	what := slices.Concat(table.Identifiers, table.columnsList())
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(what, ", "), table.Name)
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}

	q := &QueryValues{}
	for rows.Next() {
		row, err := prepareRowPointers(rows)
		if err != nil {
			return nil, err
		}
		err = rows.Scan(row...)
		if err != nil {
			return nil, err
		}

		i := 1
		set := []string{}
		setValues := []any{}
		for k, v := range row[len(table.Identifiers):] {
			set = append(set, fmt.Sprintf("%s = $%d", table.Columns[k].Name, i))
			setValues = append(setValues, v)
			i++
		}
		setSQL := fmt.Sprintf("SET %s", strings.Join(set, ", "))
		q.SetValues = append(q.SetValues, setValues)

		where := []string{}
		whereValues := []any{}
		for k, id := range table.Identifiers {
			where = append(where, fmt.Sprintf("%s = $%d", id, i))
			whereValues = append(whereValues, row[k])
			i++
		}
		whereSQL := "WHERE " + strings.Join(where, " AND ")
		q.WhereValues = append(q.WhereValues, whereValues)

		q.Query = fmt.Sprintf("UPDATE %s %s %s", table.Name, setSQL, whereSQL)
	}
	err = rows.Close() //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}

	return q, nil
}
