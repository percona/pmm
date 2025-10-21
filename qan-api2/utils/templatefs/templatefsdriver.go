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

package templatefs

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/golang-migrate/migrate/v4/source"
)

// Driver implements the golang-migrate source.Driver interface for TemplateFS.
// It allows golang-migrate to use TemplateFS as a migration source with runtime templating.
type Driver struct {
	fs    *TemplateFS
	files []fs.DirEntry
	dir   string
}

// NewDriver creates a new Driver for the given TemplateFS and directory.
func NewDriver(fs *TemplateFS, dir string) (*Driver, error) {
	files, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	return &Driver{fs: fs, files: files, dir: dir}, nil
}

// Open returns the driver itself for the given URL (required by source.Driver interface).
func (d *Driver) Open(_ string) (source.Driver, error) {
	return d, nil
}

// Close closes the driver (no-op for TemplateFS).
func (d *Driver) Close() error {
	return nil
}

// First returns the version of the first migration.
func (d *Driver) First() (uint, error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, err := parseVersion(f.Name())
			return v, err
		}
	}
	return 0, io.EOF
}

// Prev returns the previous migration version before the given version.
func (d *Driver) Prev(version uint) (uint, error) {
	var last uint
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if v < version && v > last {
				last = v
			}
		}
	}
	if last == 0 {
		return 0, io.EOF
	}
	return last, nil
}

// Next returns the next migration version after the given version.
func (d *Driver) Next(version uint) (uint, error) {
	found := false
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if found {
				return v, nil
			}
			if v == version {
				found = true
			}
		}
	}
	return 0, io.EOF
}

// ReadUp returns an io.ReadCloser for the up migration of the given version.
func (d *Driver) ReadUp(version uint) (io.ReadCloser, string, error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if v == version {
				b, err := d.fs.ReadFile(path.Join(d.dir, f.Name()))
				if err != nil {
					return nil, "", err
				}
				return io.NopCloser(strings.NewReader(string(b))), f.Name(), nil
			}
		}
	}
	return nil, "", io.EOF
}

// ReadDown returns an io.ReadCloser for the down migration of the given version.
func (d *Driver) ReadDown(version uint) (io.ReadCloser, string, error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".down.sql") {
			v, _ := parseVersion(f.Name())
			if v == version {
				b, err := d.fs.ReadFile(path.Join(d.dir, f.Name()))
				if err != nil {
					return nil, "", err
				}
				return io.NopCloser(strings.NewReader(string(b))), f.Name(), nil
			}
		}
	}
	return nil, "", io.EOF
}

// parseVersion extracts the migration version from the filename.
func parseVersion(name string) (uint, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", name)
	}
	var v uint
	_, err := fmt.Sscanf(parts[0], "%d", &v)
	return v, err
}
