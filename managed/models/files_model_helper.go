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

// InsertFile inserts file.
func InsertFile(q *reform.Querier, fp InsertFileParams) (File, error) {
	file := File{Name: fp.Name, Content: fp.Content, UpdatedAt: Now()}
	err := q.Insert(&file)
	return file, err
}

// UpsertFile inserts file and updates content on name duplicate.
func UpsertFile(ctx context.Context, q *reform.Querier, fp InsertFileParams) (file File, err error) {
	const query = `
INSERT INTO files(name, content, updated_at) 
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
content = EXCLUDED.content,
updated_at = EXCLUDED.updated_at
RETURNING name, content, updated_at
`
	err = q.WithContext(ctx).QueryRow(query, fp.Name, fp.Content, Now()).Scan(&file.Name, &file.Content, &file.UpdatedAt)
	return
}

// GetFile retrieves a file by its name.
func GetFile(q *reform.Querier, name string) (file File, err error) {
	file.Name = name
	if err = q.Reload(&file); err != nil && errors.As(err, &reform.ErrNoRows) {
		return file, ErrFileNotFound
	}
	return
}

// ReadAndUpsertFiles reads files from provided paths and returns file names in given order. Inserts empty content on not found.
func ReadAndUpsertFiles(ctx context.Context, q *reform.Querier, paths ...string) ([]string, error) {
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path) //nolint:gosec
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, `inserting file from path: %s`, path)
		}

		fp := InsertFileParams{Name: filepath.Base(path), Content: content}
		if err = fp.Validate(); err != nil {
			return nil, errors.Wrapf(err, `inserting file from path: %s`, path)
		}

		file, err := UpsertFile(ctx, q, fp)
		if err != nil {
			return nil, errors.Wrapf(err, `inserting file from path: %s`, path)
		}
		names = append(names, file.Name)
	}
	return names, nil
}

// findAndLockFile retrieves a file by name and locks it for update.
func findAndLockFile(q *reform.Querier, name string) (file File, err error) {
	const query = `WHERE name = $1 FOR NO KEY UPDATE`
	if err = q.SelectOneTo(&file, query, name); err != nil && errors.As(err, reform.ErrNoRows) {
		return file, ErrFileNotFound
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
		return ErrFileNotFound
	}
	return
}
