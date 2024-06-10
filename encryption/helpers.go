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
			row = append(row, new(string))
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
