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
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

// CreateDataDir creates/updates directories with the given permissions in the persistent volume.
func CreateDataDir(path, username, groupname string, perm os.FileMode) error {
	// store the first encountered error, but continue as far as possible
	var storedErr error

	if err := os.MkdirAll(path, perm); err != nil {
		storedErr = errors.Wrapf(err, "cannot create path %q", path)
	}

	if err := os.Chmod(path, perm); err != nil && storedErr == nil {
		storedErr = errors.Wrapf(err, "cannot chmod path %q", path)
	}

	if err := chown(path, username, groupname); err != nil && storedErr == nil {
		storedErr = err // already wrapped
	}

	return storedErr
}

// FindFilesWithExtensions reads path directory and returns all files satisfying provided extensions.
// File name is joined with provided path.
func FindFilesWithExtensions(path string, extensions ...string) ([]string, error) {
	var paths []string
	match := func(ext string) bool {
		for _, e := range extensions {
			if e == ext {
				return true
			}
		}
		return false
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if ext := filepath.Ext(entry.Name()); len(ext) > 0 && match(ext[1:]) {
			paths = append(paths, filepath.Join(path, entry.Name()))
		}
	}
	return paths, nil
}

// chown is like os.Chown, but with names instead of numerical IDs.
func chown(path, username, groupname string) error {
	userInfo, err := user.Lookup(username)
	if err != nil {
		return errors.WithStack(err)
	}
	groupInfo, err := user.LookupGroup(groupname)
	if err != nil {
		return errors.WithStack(err)
	}

	uid, err := strconv.Atoi(userInfo.Uid)
	if err != nil {
		return errors.Wrapf(err, "cannot convert uid %q", userInfo.Uid)
	}
	gid, err := strconv.Atoi(groupInfo.Gid)
	if err != nil {
		return errors.Wrapf(err, "cannot convert gid %q", groupInfo.Gid)
	}

	if err = os.Chown(path, uid, gid); err != nil {
		return errors.Wrapf(err, "cannot chown path %q", path)
	}

	return nil
}
