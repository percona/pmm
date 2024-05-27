package encryption

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/google/tink/go/tink"
)

type Encryption struct {
	Path string
	Key  string
}

func New(keyPath string) (*Encryption, error) {
	e := new(Encryption)
	e.Path = keyPath

	bytes, err := os.ReadFile(e.Path)
	switch {
	case os.IsNotExist(err):
		err = e.generateKey()
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		e.Key = string(bytes)
	}

	return e, nil
}

func (e Encryption) encrypt(secret string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}
	cipherText, err := primitive.Encrypt([]byte(secret), []byte(""))
	if err != nil {
		return "", err
	}

	return string(cipherText), nil
}

func (e Encryption) decrypt(cipherText string) (string, error) {
	primitive, err := e.getPrimitive()
	if err != nil {
		return "", err
	}
	secret, err := primitive.Decrypt([]byte(cipherText), []byte(""))
	if err != nil {
		return "", err
	}

	return string(secret), nil
}

func (e Encryption) getPrimitive() (tink.AEAD, error) {
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

func (e Encryption) generateKey() error {
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

func (e Encryption) saveKeyToFile() error {
	return os.WriteFile(e.Path, []byte(e.Key), 0644)
}

func (e Encryption) Migrate(c *DatabaseConnection) error {
	connection := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", c.Host, c.Port, c.User, c.Password)
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	if len(c.EncryptedItems) == 0 {
		return errors.New("Migration: Database with target tables/columns not defined")
	}

	for _, item := range c.EncryptedItems {
		// TODO read and update for all rows in scope of 1 transcation
		tx, err := db.BeginTx(context.TODO(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// TODO injection?
		what := append(item.Identificators, item.Columns...)
		query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(what, ","), item.Table)
		rows, err := tx.Query(query)
		if err != nil {
			return err
		}

		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			return err
		}
		columns := make(map[string]string)
		for _, columnType := range columnTypes {
			columns[columnType.Name()] = columnType.DatabaseTypeName()
		}

		encryptedRows := []string{}
		for rows.Next() {
			row := make([]any, len(what))
			i := 0
			for _, t := range columns {
				switch t {
				case "VARCHAR":
					row[i] = new(string)
				default:
					return fmt.Errorf("unsupported identificator type %s", t)
				}

				i++
			}

			err = rows.Scan(
				row...,
			)
			if err != nil {
				return err
			}

			where := []string{}
			for k, id := range item.Identificators {
				where = append(where, fmt.Sprintf("%s = '%s'", id, *row[k].(*string)))
			}
			whereSQL := fmt.Sprintf("WHERE %s", strings.Join(where, " AND "))

			encryptedValues := []string{}
			i = 0
			for _, v := range row[len(item.Identificators):] {
				s, err := e.encrypt(*v.(*string))
				if err != nil {
					return err
				}
				encryptedValues = append(encryptedValues, fmt.Sprintf("%s = '%s'", item.Columns[i], base64.StdEncoding.EncodeToString([]byte(s))))
				i++
			}
			setSQL := fmt.Sprintf("SET %s", strings.Join(encryptedValues, ", "))

			sql := fmt.Sprintf("UPDATE %s %s %s", item.Table, setSQL, whereSQL)
			encryptedRows = append(encryptedRows, sql)
		}
		err = rows.Close()
		if err != nil {
			return err
		}

		for _, r := range encryptedRows {
			_, err := tx.Exec(r)
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
