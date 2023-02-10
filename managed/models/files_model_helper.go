// Copyright (C) 2017 Percona LLC
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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// InsertFile inserts file. Does nothing on duplicate. Retrieves stored.
func InsertFile(q *reform.Querier, fp InsertFileParams) (file File, err error) {
	const query = `
INSERT INTO files(name, content, updated_at) 
VALUES ($1, $2, $3)
ON CONFLICT (name) DO NOTHING
RETURNING name, content, updated_at
	`
	err = q.QueryRow(query, fp.Name, fp.Content, Now()).Scan(&file.Name, &file.Content, &file.UpdatedAt)
	if err != nil && errors.As(err, &reform.ErrNoRows) {
		file.Name = fp.Name
		err = q.Reload(&file)
	}
	return
}

// UpsertFile inserts file and updates content on name duplicate.
func UpsertFile(q *reform.Querier, fp InsertFileParams) (file File, err error) {
	const query = `
INSERT INTO files(name, content, updated_at) 
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
content = EXCLUDED.content,
updated_at = EXCLUDED.updated_at
RETURNING name, content, updated_at
`
	err = q.QueryRow(query, fp.Name, fp.Content, Now()).Scan(&file.Name, &file.Content, &file.UpdatedAt)
	return
}

// GetFile retrieves a file by its name.
func GetFile(q *reform.Querier, name string) (file File, err error) {
	file.Name = name
	if err = q.Reload(&file); err != nil && errors.As(err, &reform.ErrNoRows) {
		return file, ErrNotFound
	}
	return
}

// GetOrInsertFile gets file by base of path as name. Inserts on not found.
func GetOrInsertFile(q *reform.Querier, path string) (file File, err error) {
	name := filepath.Base(path)
	content, err := os.ReadFile(path) //nolint:gosec
	if err != nil && !os.IsNotExist(err) {
		return file, errors.Wrapf(err, `get or insert file from path: %s`, path)
	}

	fp := InsertFileParams{Name: name, Content: content}
	if err = fp.Validate(); err != nil {
		return file, errors.Wrapf(err, `get or insert file from path: %s`, path)
	}

	if file, err = GetFile(q, name); err != nil {
		if !errors.Is(err, ErrNotFound) {
			return file, errors.Wrapf(err, `get or insert file from path: %s`, path)
		}
		if file, err = InsertFile(q, fp); err != nil {
			return file, errors.Wrapf(err, `get or insert file from path: %s`, path)
		}
	}
	return
}

// findAndLockFile retrieves a file by name and locks it for update.
func findAndLockFile(q *reform.Querier, name string) (file File, err error) {
	const query = `WHERE name = $1 FOR NO KEY UPDATE`
	if err = q.SelectOneTo(&file, query, name); err != nil && errors.As(err, reform.ErrNoRows) {
		return file, ErrNotFound
	}
	return
}

// UpdateFile updates file with given content.
func UpdateFile(ctx context.Context, db reform.DBTXContext, fp UpdateFileParams) (file File, err error) {
	const query = `
UPDATE files
SET content = $1,
updated_at = $2
WHERE name = $3
RETURNING name, content, updated_at
`
	updateFile := func(t *reform.TX) (txErr error) {
		if file, txErr = findAndLockFile(t.Querier, fp.Name); txErr != nil {
			return
		}

		if bytes.Compare(fp.Content, file.Content) != 0 {
			file.Content = fp.Content
		}

		if txErr = t.Querier.QueryRow(query, &file.Content, Now(), &file.Name).Scan(&file.Name, &file.Content, &file.UpdatedAt); txErr != nil {
			txErr = errors.Wrap(txErr, "failed to update file")
		}
		return
	}

	switch v := db.(type) {
	case *reform.DB:
		err = v.InTransactionContext(ctx, nil, updateFile)
	case *reform.TX:
		err = updateFile(v)
	default:
		err = fmt.Errorf("unsupported *reform.DBTXContext; want: *reform.TX or *reform.DB; got: %T", v)
	}
	return
}

// DeleteFile deletes file by its name.
func DeleteFile(q *reform.Querier, name string) (err error) {
	if err = q.Delete(&File{Name: name}); err != nil && errors.As(err, reform.ErrNoRows) {
		return ErrNotFound
	}
	return
}
