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

// Package dir contains utilities for creating directories.
package dir

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// CreateDataDir creates/updates directories with the given permissions in the persistent volume.
func CreateDataDir(path string, perm os.FileMode) error {
	// store the first encountered error, but continue as far as possible
	var storedErr error

	err := os.MkdirAll(path, perm)
	if err != nil {
		storedErr = fmt.Errorf("cannot create path %q: %w", path, err)
	}

	err = os.Chmod(path, perm)
	if err != nil && storedErr == nil {
		storedErr = fmt.Errorf("cannot chmod path %q: %w", path, err)
	}

	return storedErr
}

// WriteFileAtomic writes a file so that a reader never observes a partial one: the content goes to
// a temporary file in the same directory, which is then renamed over the destination. Needed where
// another process may read the file at any moment, such as Grafana scanning its provisioning
// directory while pmm-managed rewrites it.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir, name := filepath.Split(path)
	if dir == "" {
		dir = "."
	}

	// The temporary file has to share a filesystem with the destination, otherwise the rename is
	// not atomic and may fail outright.
	tmp, err := os.CreateTemp(dir, "."+name+".tmp")
	if err != nil {
		return fmt.Errorf("cannot create a temporary file for %q: %w", path, err)
	}
	tmpName := tmp.Name()

	defer func() {
		// A no-op once the rename has succeeded.
		_ = os.Remove(tmpName)
	}()

	_, err = tmp.Write(data)
	if err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return fmt.Errorf("cannot write %q: %w", path, err)
	}

	// CreateTemp always uses 0600, so the requested permissions have to be set explicitly.
	err = tmp.Chmod(perm)
	if err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return fmt.Errorf("cannot chmod %q: %w", path, err)
	}

	err = tmp.Close()
	if err != nil {
		return fmt.Errorf("cannot close %q: %w", path, err)
	}

	err = os.Rename(tmpName, path)
	if err != nil {
		return fmt.Errorf("cannot replace %q: %w", path, err)
	}

	return nil
}

// FindFilesWithExtensions reads path directory and returns all files satisfying provided extensions.
// File name is joined with provided path.
func FindFilesWithExtensions(path string, extensions ...string) ([]string, error) {
	var paths []string
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if ext := filepath.Ext(entry.Name()); len(ext) > 0 && slices.Contains(extensions, ext[1:]) {
			paths = append(paths, filepath.Join(path, entry.Name()))
		}
	}
	return paths, nil
}
